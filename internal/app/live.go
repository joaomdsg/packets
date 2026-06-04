package app

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-via/via"
	"github.com/go-via/via/h"

	"github.com/joaomdsg/agntpr/internal/ledger"
	"github.com/joaomdsg/agntpr/internal/pipe"
	"github.com/joaomdsg/agntpr/internal/reanchor"
	"github.com/joaomdsg/agntpr/internal/surface"
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
}

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
	liveReg.Store(key, &liveEntry{cfg: cfg, log: log, sem: sem})
}

func setLiveState(cfg LiveConfig, log *ledger.Log) {
	registerSession(defaultSessionKey, cfg, log)
}

// AddSession opens a session's ledger and registers it under key, so the one
// "/" mount also serves /?key=<key> with its OWN isolated economy (its own
// ledger + admission sem). The caller closes the returned ledger. This is the
// wiring entry the command uses to stand up a SECOND review target beyond the
// default card; the core keyed registration + cross-session isolation is
// registerSession, exercised by the live tests.
func AddSession(key string, cfg LiveConfig) (*ledger.Log, error) {
	log, err := ledger.Open(cfg.LedgerPath)
	if err != nil {
		return nil, err
	}
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
	dispatched := 0
	if log != nil {
		if recs, err := log.Records(); err == nil {
			stock = ledger.ConfirmedCatches(recs)
		}
		if b, err := log.Balance(); err == nil {
			balance = b
		}
		if d, err := log.PendingDispatches(); err == nil {
			dispatched = d
		}
	}
	return h.Div(
		surface.RenderStock(stock),
		surface.RenderBalance(balance),
		surface.RenderDispatch(dispatched),
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
	_, log := readLiveState(c.Key)
	if log == nil {
		return
	}
	if err := log.AppendDispatch("dispatch"); err != nil {
		return // over-budget / nothing to spend: a no-op, never an error to the Lead
	}
	if b, err := log.Balance(); err == nil {
		c.Balance.Write(ctx, strconv.Itoa(b)) // announce the drain
	}
	if d, err := log.PendingDispatches(); err == nil {
		c.Dispatch.Write(ctx, strconv.Itoa(d)) // announce the funded work-order so the dispatch row rises in the same render
	}
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
			_ = log.Append(*res.Record) // best-effort; a logging failure must not hang the card
		}
		result <- resolved{verdict: res.Verdict, land: string(res.Land)}
	}()
	var accrued []string
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
		select {
		case r := <-result:
			c.Verdict.Write(ctx, r.verdict)
			c.Land.Write(ctx, r.land)
		default:
		}
	})
	return nil
}

// NewServer wires the live review server: it opens the catch ledger, stashes
// the cycle config, mounts the LiveCard, and returns the Via app (an
// http.Handler) plus the ledger handle for the caller to close. Extra Via
// options (e.g. via.WithTestServer) are passed through.
func NewServer(cfg LiveConfig, opts ...via.Option) (*via.App, *ledger.Log, error) {
	log, err := ledger.Open(cfg.LedgerPath)
	if err != nil {
		return nil, nil, err
	}
	setLiveState(cfg, log)
	app := via.New(opts...)
	via.Mount[LiveCard](app, "/")
	return app, log, nil
}
