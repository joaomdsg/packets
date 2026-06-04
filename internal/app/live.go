package app

import (
	"context"
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

// liveState holds the process-wide config + ledger the LiveCard reads on
// connect. Via mounts compositions by type (zero-value per tab) with no
// constructor injection, so the wiring is stashed here once by NewServer. This
// is a single-instance demo server (one Lead, one card); a multi-card server
// would key this per session and is out of scope for the watchable wire.
var liveState struct {
	mu  sync.RWMutex
	cfg LiveConfig
	log *ledger.Log
	// sem is the bounded admission queue: a buffered channel of size
	// cfg.MaxConcurrent, or nil when uncapped. A send acquires a cycle slot, a
	// receive releases it; connects beyond the cap block on the send (queued).
	sem chan struct{}
}

func setLiveState(cfg LiveConfig, log *ledger.Log) {
	liveState.mu.Lock()
	defer liveState.mu.Unlock()
	liveState.cfg, liveState.log = cfg, log
	if cfg.MaxConcurrent > 0 {
		liveState.sem = make(chan struct{}, cfg.MaxConcurrent)
	} else {
		liveState.sem = nil
	}
}

func readLiveState() (LiveConfig, *ledger.Log) {
	liveState.mu.RLock()
	defer liveState.mu.RUnlock()
	return liveState.cfg, liveState.log
}

func cycleSem() chan struct{} {
	liveState.mu.RLock()
	defer liveState.mu.RUnlock()
	return liveState.sem
}

// LiveCard is the served review card. On connect it renders the in-flight state
// immediately, runs the catch cycle in the background, and resolves the card in
// place over SSE when the verdict lands — so a human watches one verdict go
// in-flight → resolved, with the catch (if any) appended to the ledger.
type LiveCard struct {
	Verdict via.StateTabStr
	Land    via.StateTabStr
	Beats   via.StateTabStr
}

// View renders the card's rows via the shared surface rendering: the retrospective
// confirmed-catch STOCK (re-derived read-only from the ledger on every render — the
// economy finally SHOWN, not just logged), the streamed beat row (the felt tempo),
// the oracle verdict row, and the integration (Land) row. One row never speaks for
// another. The stock is read-only: a ledger read failure degrades to an empty
// stock, never breaks the card.
func (c *LiveCard) View(ctx *via.CtxR) h.H {
	_, log := readLiveState()
	var stock ledger.Stock
	if log != nil {
		if recs, err := log.Records(); err == nil {
			stock = ledger.ConfirmedCatches(recs)
		}
	}
	return h.Div(
		surface.RenderStock(stock),
		surface.RenderBeats(c.Beats.Read(ctx)),
		surface.RenderVerdict(c.Verdict.Read(ctx)),
		surface.RenderLand(pipe.LandState(c.Land.Read(ctx))),
	)
}

// OnConnect kicks off the catch cycle and streams its beats live: each pipe
// transition (settle-base → oracle-base → … → catch → land) is flushed to the
// beat row as it happens, and the verdict + Land rows resolve only when the cycle
// completes. So the human feels the loop's tempo over the seconds of real oracle +
// rebase work, instead of watching a spinner snap to a verdict. The beats channel
// is buffered past the beat count so the cycle never blocks on a slow/gone client.
func (c *LiveCard) OnConnect(ctx *via.Ctx) error {
	cfg, log := readLiveState()
	sem := cycleSem()
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
