package app

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-via/via"
	"github.com/go-via/via/h"

	"github.com/joaomdsg/packets/internal/fabric"
	"github.com/joaomdsg/packets/internal/ledger"
	"github.com/joaomdsg/packets/internal/pipe"
	"github.com/joaomdsg/packets/internal/reanchor"
	"github.com/joaomdsg/packets/internal/surface"
)

// LiveConfig is the single catch cycle the live server drives: the two
// revisions, the anchored line, how to run the suite, and the mint-time bits.
type LiveConfig struct {
	RepoDir          string
	BaseRev          string
	FixRev           string
	TipRev           string
	Anchor           reanchor.Anchor
	TestCmd          []string
	LedgerPath       string
	SelfFlagged      bool
	WouldHaveShipped bool
	// MaxConcurrent caps how many catch cycles run at once (each cycle is several
	// full-suite executions — see internal/pipe and the #15 benchmark). Connects
	// beyond the cap QUEUE on a slot, they are never dropped. 0 means unbounded.
	MaxConcurrent int
	// DispatchBacklog is the ordered supply of DISTINCT work a card's Spends draw
	// down — the rev/anchor triple each funded order runs. A Spend consumes the next
	// not-yet-funded target head-first; an empty or fully-drawn-down backlog makes a
	// Spend a silent no-op (the honest scarcity signal — no distinct work to buy).
	DispatchBacklog []ledger.Target
}

// resolveCycle is the seam OnConnect runs the catch cycle through. It defaults to
// the real ResolveStreaming; tests swap it to drive the admission cap
// deterministically without spinning up real oracle work.
var resolveCycle = ResolveStreaming

// liveEntry is one session's wiring: the cycle config, its ledger, and its
// admission semaphore (a buffered channel of size cfg.MaxConcurrent, or nil when
// uncapped — a send acquires a cycle slot, a receive releases it).
type liveEntry struct {
	cfg LiveConfig
	log *ledger.Log
	sem chan struct{}
	// runMu serializes the per-key order runner so two concurrent Spends can't both
	// drain (and double-run) the same queued order. One drainer per session at a time.
	runMu sync.Mutex
	// seq is the registration ordinal — a monotonic stamp assigned when the session
	// is registered. The fleet board orders ties (equal queued counts) by it, since
	// sync.Map.Range is nondeterministic and a CatchRecord carries no timestamp to
	// order by; registration order is the only stable, honest ordinal.
	seq int
}

// regSeq is the monotonic source of liveEntry.seq — incremented once per session
// registration so the board's tie-break is deterministic across renders.
var regSeq int64

// defaultSessionKey is the one entry seeded today. The registry can hold an entry
// per session key so ≥2 distinct cards can coexist; until a second session is
// registered (a later slice), every connect falls back to this one entry, so the
// server behaves as the single-card demo it has been (one Lead, one card).
const defaultSessionKey = "default"

// liveReg maps a session key → *liveEntry. Via mounts LiveCard by type (zero-value
// per tab, no constructor injection), so the wiring is stashed here and looked up
// by a connect-derived key. A sync.Map is safe for the concurrent reads
// (View/Spend/OnConnect across tabs) and the connect-time write.
var liveReg sync.Map

// registerSession stores one keyed session's wiring (its own cfg, ledger, and
// admission sem) in the registry. Distinct keys get distinct entries with their
// own *ledger.Log, so ≥2 cards served off the one "/" mount are ISOLATED
// economies — a mint or spend on one key never touches another (the R18
// farm-denial verdict, enforced per session: the faucet is the sole credit
// source and a balance is non-transferable across keys).
func registerSession(key string, cfg LiveConfig, log *ledger.Log) {
	var sem chan struct{}
	if cfg.MaxConcurrent > 0 {
		sem = make(chan struct{}, cfg.MaxConcurrent)
	}
	liveReg.Store(key, &liveEntry{cfg: cfg, log: log, sem: sem, seq: int(atomic.AddInt64(&regSeq, 1))})
}

func setLiveState(cfg LiveConfig, log *ledger.Log) {
	registerSession(defaultSessionKey, cfg, log)
}

// ledgerInstance is the subject instance token every session's economy binds to.
// There is one economy per session, so the session key alone demuxes them; the
// instance is a fixed token completing the canonical subject.
const ledgerInstance = "ledger"

// liveFabric is the one embedded JetStream the server's sessions share — the
// single authoritative economy substrate (DESIGN-COUNCIL Round 28). NewServer
// starts it and gives the primary Log ownership of its lifecycle; AddSession
// binds further sessions to it under their own session token, so each session is
// an ISOLATED economy on the one stream. Set once per server; the live tests
// drive NewServer serially (they share this and liveReg), so it is not guarded.
var liveFabric *fabric.Fabric

// startLiveFabric stands up the shared economy fabric, rooting its durable store
// beside the configured ledger path (a dedicated dir per server, so two servers
// in one process never share a store). An empty path falls back to a temp store.
func startLiveFabric(ledgerPath string) (*fabric.Fabric, error) {
	dir := ledgerPath + "-fabric"
	if ledgerPath == "" {
		d, err := os.MkdirTemp("", "packets-fabric-*")
		if err != nil {
			return nil, fmt.Errorf("app: fabric store dir: %v", err)
		}
		dir = d
	} else if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("app: fabric store dir: %v", err)
	}
	return fabric.Start(context.Background(), dir)
}

// AddSession binds a session's economy to the shared fabric and registers it
// under key, so the one "/" mount also serves /?key=<key> with its OWN isolated
// economy (its own session subtree on the stream + admission sem). The returned
// Log does not own the fabric, so its Close is a no-op; the fabric's lifecycle
// belongs to the primary Log from NewServer. This is the wiring entry the command
// uses to stand up a SECOND review target beyond the default card; the core keyed
// registration + cross-session isolation is registerSession, exercised by the
// live tests.
func AddSession(key string, cfg LiveConfig) (*ledger.Log, error) {
	if liveFabric == nil {
		return nil, fmt.Errorf("app: AddSession before NewServer started the fabric")
	}
	log := ledger.Bind(liveFabric, key, ledgerInstance)
	registerSession(key, cfg, log)
	return log, nil
}

// lookupLiveEntry resolves a session key to its entry, falling back to the default
// session when the key isn't registered — so a connect whose key has no dedicated
// entry still drives the one seeded session (behavior-preserving while only
// defaultSessionKey is seeded). Returns nil only if nothing is registered at all.
func lookupLiveEntry(key string) *liveEntry {
	if v, ok := liveReg.Load(key); ok {
		return v.(*liveEntry)
	}
	if v, ok := liveReg.Load(defaultSessionKey); ok {
		return v.(*liveEntry)
	}
	return nil
}

func readLiveState(key string) (LiveConfig, *ledger.Log) {
	if e := lookupLiveEntry(key); e != nil {
		return e.cfg, e.log
	}
	return LiveConfig{}, nil
}

func cycleSem(key string) chan struct{} {
	if e := lookupLiveEntry(key); e != nil {
		return e.sem
	}
	return nil
}

// LiveCard is the served review card. On connect it renders the in-flight state
// immediately, runs the catch cycle in the background, and resolves the card in
// place over SSE when the verdict lands — so a human watches one verdict go
// in-flight → resolved, with the catch (if any) appended to the ledger.
type LiveCard struct {
	// Key selects the session this card drives — its registry entry (cfg, ledger,
	// sem). It is decoded from the ?key= query slot into the per-connection
	// instance (Via persists it per tab and re-decodes it on action POSTs). An
	// empty Key (the "/" route, no ?key) falls back to defaultSessionKey via the
	// registry lookup — so the single-card "/" wire is byte-identical.
	Key     string `query:"key"`
	Verdict via.StateTabStr
	Land    via.StateTabStr
	Beats   via.StateTabStr
	// Balance is the spend broadcast trigger: the balance ROW value is re-read
	// from the ledger in View (the source of truth), but the ledger is not
	// reactive — so Spend writes the new balance here to make the live SSE stream
	// re-render (a cell Write fans out a re-render; an action's auto-render only
	// returns in the action's own response).
	Balance via.StateTabStr
	// Dispatch is the same broadcast trigger for the dispatched-work tally: the
	// count is re-read from the ledger in View, but a Spend writes the new count
	// here so the dispatch row rises over the live SSE stream in the SAME render as
	// the balance drains. It carries no authoritative value — View is the source.
	Dispatch via.StateTabStr
}

// View renders the card's rows via the shared surface rendering: the retrospective
// confirmed-catch STOCK (re-derived read-only from the ledger on every render — the
// economy finally SHOWN, not just logged), the streamed beat row (the felt tempo),
// the oracle verdict row, and the integration (Land) row. One row never speaks for
// another. The stock is read-only: a ledger read failure degrades to an empty
// stock, never breaks the card.
func (c *LiveCard) View(ctx *via.CtxR) h.H {
	_, log := readLiveState(c.Key)
	var stock ledger.Stock
	balance := 0
	var dispatch ledger.DispatchCounts
	if log != nil {
		if recs, err := log.Records(); err == nil {
			stock = ledger.ConfirmedCatches(recs)
		}
		if b, err := log.Balance(); err == nil {
			balance = b
		}
		if c, err := log.DispatchStatusCounts(); err == nil {
			dispatch = c
		}
	}
	return h.Div(
		surface.RenderStock(stock),
		surface.RenderBalance(balance),
		surface.RenderDispatch(dispatch),
		surface.RenderBeats(c.Beats.Read(ctx)),
		surface.RenderVerdict(c.Verdict.Read(ctx)),
		surface.RenderLand(pipe.LandState(c.Land.Read(ctx))),
	)
}

// Spend funds one unit of dispatched work against the balance — the Lead's first
// ACTION on the stock, and the moment a catch finally BUYS something. It debits
// one catch AND fuels exactly one queued work-order in a single atomic ledger
// fact (AppendDispatch). An over-budget spend (balance already 0) is refused by
// the ledger and the action is a silent no-op (no broadcast). On success it
// writes BOTH the drained balance and the risen dispatch count to their trigger
// cells, whose Writes fan out a single re-render to the live SSE stream so the
// balance drains and the dispatch row rises together — the spend is visibly
// converted into work, not just a vanishing number.
func (c *LiveCard) Spend(ctx *via.Ctx) {
	cfg, log := readLiveState(c.Key)
	if log == nil {
		return
	}
	tgt, ok := nextUnconsumedTarget(cfg, log)
	if !ok {
		return // backlog exhausted / empty: no distinct work to buy — a silent no-op
	}
	if err := log.AppendDispatch("dispatch", tgt, ownTargetOf(cfg)); err != nil {
		return // over-budget / nothing to spend / own work: a no-op, never an error to the Lead
	}
	if b, err := log.Balance(); err == nil {
		c.Balance.Write(ctx, strconv.Itoa(b)) // announce the drain
	}
	if d, err := log.PendingDispatches(); err == nil {
		c.Dispatch.Write(ctx, strconv.Itoa(d)) // announce the funded work-order so the dispatch row rises in the same render
	}
	go drainQueuedOrders(c.Key) // the order RUNS in the background — spend-to-earn
}

// maxOrderAttempts bounds how many times the runner will pick a single queued
// order before giving up on it. A status write that fails permanently (e.g. a
// closed ledger handle) would otherwise leave an order forever queued and spin
// the suite-exec loop without end; the cap turns that into a bounded, abandoned
// order instead of an unbounded #15-multiplier burn.
const maxOrderAttempts = 3

// drainQueuedOrders runs every queued work-order for a session to completion — the
// second in-process producer. It serializes per session (runMu) so two concurrent
// Spends never double-run an order. Each order: mark running, run its DISTINCT
// target through the catch cycle under the admission sem (bounding the suite-exec
// cost), route any Catch through the idempotent Append stamped with the order's
// producer (a re-run that reproduces a seen identity mints nothing — an honest
// loss), then mark done. The mint is the only thing logged; intermediate beats
// stay off-ledger. An order whose status can never advance is retried at most
// maxOrderAttempts times then GIVEN UP (a best-effort terminal "failed" line, so
// it leaves the queued set when the log is writable), guaranteeing the drain
// always returns.
func drainQueuedOrders(key string) {
	e := lookupLiveEntry(key)
	if e == nil || e.log == nil {
		return
	}
	e.runMu.Lock()
	defer e.runMu.Unlock()
	attempts := map[int]int{}
	givenUp := map[int]bool{}
	for {
		queued, err := e.log.QueuedWorkOrders()
		if err != nil {
			return
		}
		var order *ledger.WorkOrderRecord
		for i := range queued {
			if !givenUp[queued[i].ID] {
				order = &queued[i]
				break
			}
		}
		if order == nil {
			return // nothing left that hasn't been given up
		}
		attempts[order.ID]++
		if attempts[order.ID] > maxOrderAttempts {
			givenUp[order.ID] = true
			_ = e.log.AppendStatus(order.ID, "failed") // best-effort terminal line; if this too fails, givenUp still bounds the loop
			continue
		}
		runOneOrder(e, *order)
	}
}

func runOneOrder(e *liveEntry, order ledger.WorkOrderRecord) {
	if err := e.log.AppendStatus(order.ID, "running"); err != nil {
		return // could not advance the order's status — don't run; the drain loop retries under the attempts cap
	}
	if e.sem != nil {
		e.sem <- struct{}{}
		defer func() { <-e.sem }()
	}
	beats := make(chan pipe.TraceEvent, 64)
	go func() { // discard beats: the dispatched run's tempo is off-ledger this round
		for range beats {
		}
	}()
	res, err := resolveCycle(context.Background(), e.cfg.RepoDir,
		order.Target.BaseRev, order.Target.FixRev, order.Target.TipRev,
		anchorFromTarget(order.Target), e.cfg.TestCmd, false, false, beats)
	close(beats) // the cycle only SENDS on beats; the caller owns the close, so the discard goroutine exits (mirrors OnConnect)
	if err == nil && res.Record != nil {
		res.Record.Producer = "wo:" + strconv.Itoa(order.ID)
		_ = e.log.Append(*res.Record) // deduped: a re-run of a seen identity mints nothing
	}
	_ = e.log.AppendStatus(order.ID, "done")
}

// anchorFromTarget reconstructs the re-anchor anchor a funded order runs against
// from the target's persisted rev/anchor fields.
func anchorFromTarget(t ledger.Target) reanchor.Anchor {
	return reanchor.Anchor{Path: t.Path, Start: t.Line, End: t.Line, LineHash: t.LineHash}
}

// OnConnect kicks off the catch cycle and streams its beats live: each pipe
// transition (settle-base → oracle-base → … → catch → land) is flushed to the
// beat row as it happens, and the verdict + Land rows resolve only when the cycle
// completes. So the human feels the loop's tempo over the seconds of real oracle +
// rebase work, instead of watching a spinner snap to a verdict. The beats channel
// is buffered past the beat count so the cycle never blocks on a slow/gone client.
func (c *LiveCard) OnConnect(ctx *via.Ctx) error {
	cfg, log := readLiveState(c.Key)
	sem := cycleSem(c.Key)
	type resolved struct{ verdict, land string }
	beats := make(chan pipe.TraceEvent, 16)
	result := make(chan resolved, 1)
	go func() {
		// Acquire a cycle slot (when capped): connects beyond MaxConcurrent block
		// here until a running cycle frees a slot — queued, never dropped. The
		// release covers every exit path (cycle error included), so a slot can't leak.
		if sem != nil {
			sem <- struct{}{}
			defer func() { <-sem }()
		}
		res, err := resolveCycle(context.Background(), cfg.RepoDir, cfg.BaseRev, cfg.FixRev, cfg.TipRev,
			cfg.Anchor, cfg.TestCmd, cfg.SelfFlagged, cfg.WouldHaveShipped, beats)
		close(beats)
		if err != nil {
			result <- resolved{} // leave the card in-flight on a cycle error
			return
		}
		if res.Record != nil && log != nil {
			res.Record.Producer = "connect" // provenance: the connect-cycle producer, demuxed from a dispatched run's "wo:<id>"
			_ = log.Append(*res.Record)     // best-effort; a logging failure must not hang the card
		}
		result <- resolved{verdict: res.Verdict, land: string(res.Land)}
	}()
	var accrued []string
	lastDispatch := -1
	via.Stream(ctx, 100*time.Millisecond, func(ctx *via.Ctx, _ time.Time) {
		for { // drain every beat available this tick, flushing the growing row
			select {
			case ev, ok := <-beats:
				if !ok {
					beats = nil // closed: stop selecting on it (a nil channel never fires)
					break
				}
				accrued = append(accrued, ev.Kind)
				c.Beats.Write(ctx, strings.Join(accrued, ","))
				continue
			default:
			}
			break
		}
		// Poll the dispatch tally so a BACKGROUND order runner (drainQueuedOrders has
		// no request ctx, cannot write cells) still surfaces over SSE: when the
		// per-status counts change, write the Dispatch cell to re-render, so the Lead
		// watches the order move queued→running→done live. Keyed on a cheap signature
		// so an unchanged tally writes nothing (no spurious frames).
		if log != nil {
			if cnt, err := log.DispatchStatusCounts(); err == nil {
				if sig := cnt.Queued*1_000_000 + cnt.Running*1_000 + cnt.Done; sig != lastDispatch {
					lastDispatch = sig
					c.Dispatch.Write(ctx, strconv.Itoa(sig))
				}
			}
		}
		select {
		case r := <-result:
			c.Verdict.Write(ctx, r.verdict)
			c.Land.Write(ctx, r.land)
		default:
		}
	})
	return nil
}

// NewServer wires the live review server: it starts the shared economy fabric,
// binds the default session's ledger (which OWNS the fabric's lifecycle), stashes
// the cycle config, mounts the LiveCard, and returns the Via app (an
// http.Handler) plus the ledger handle for the caller to close (closing it tears
// the fabric down). Extra Via options (e.g. via.WithTestServer) are passed through.
func NewServer(cfg LiveConfig, opts ...via.Option) (*via.App, *ledger.Log, error) {
	f, err := startLiveFabric(cfg.LedgerPath)
	if err != nil {
		return nil, nil, err
	}
	liveFabric = f
	log := ledger.BindOwning(f, defaultSessionKey, ledgerInstance)
	setLiveState(cfg, log)
	app := via.New(opts...)
	via.Mount[LiveCard](app, "/")
	via.Mount[BoardCard](app, "/board") // the cross-card fleet view (read-only projection of liveReg)
	return app, log, nil
}
