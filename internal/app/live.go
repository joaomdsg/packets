package app

import (
	"context"
	"sync"
	"time"

	"github.com/go-via/via"
	"github.com/go-via/via/h"

	"github.com/joaomdsg/agntpr/internal/ledger"
	"github.com/joaomdsg/agntpr/internal/reanchor"
	"github.com/joaomdsg/agntpr/internal/surface"
)

// LiveConfig is the single catch cycle the live server drives: the two
// revisions, the anchored line, how to run the suite, and the mint-time bits.
type LiveConfig struct {
	RepoDir          string
	BaseRev          string
	FixRev           string
	Anchor           reanchor.Anchor
	TestCmd          []string
	LedgerPath       string
	SelfFlagged      bool
	WouldHaveShipped bool
}

// liveState holds the process-wide config + ledger the LiveCard reads on
// connect. Via mounts compositions by type (zero-value per tab) with no
// constructor injection, so the wiring is stashed here once by NewServer. This
// is a single-instance demo server (one Lead, one card); a multi-card server
// would key this per session and is out of scope for the watchable wire.
var liveState struct {
	mu  sync.RWMutex
	cfg LiveConfig
	log *ledger.Log
}

func setLiveState(cfg LiveConfig, log *ledger.Log) {
	liveState.mu.Lock()
	defer liveState.mu.Unlock()
	liveState.cfg, liveState.log = cfg, log
}

func readLiveState() (LiveConfig, *ledger.Log) {
	liveState.mu.RLock()
	defer liveState.mu.RUnlock()
	return liveState.cfg, liveState.log
}

// LiveCard is the served review card. On connect it renders the in-flight state
// immediately, runs the catch cycle in the background, and resolves the card in
// place over SSE when the verdict lands — so a human watches one verdict go
// in-flight → resolved, with the catch (if any) appended to the ledger.
type LiveCard struct {
	Verdict via.StateTabStr
}

// View renders the current verdict via the shared surface rendering, so the
// live card and the reviewer card resolve to identical designed states.
func (c *LiveCard) View(ctx *via.CtxR) h.H {
	return surface.RenderVerdict(c.Verdict.Read(ctx))
}

// OnConnect kicks off the catch cycle for this card and streams the verdict in
// when it completes. The cycle runs in a goroutine (it is seconds of real
// oracle work); a short Stream poll writes the verdict the moment it is ready,
// flushing a single in-flight → resolved SSE patch.
func (c *LiveCard) OnConnect(ctx *via.Ctx) error {
	cfg, log := readLiveState()
	result := make(chan string, 1)
	go func() {
		res, err := Resolve(context.Background(), cfg.RepoDir, cfg.BaseRev, cfg.FixRev,
			cfg.Anchor, cfg.TestCmd, cfg.SelfFlagged, cfg.WouldHaveShipped)
		if err != nil {
			result <- "" // leave the card in-flight on a cycle error
			return
		}
		if res.Record != nil && log != nil {
			_ = log.Append(*res.Record) // best-effort; a logging failure must not hang the card
		}
		result <- res.Verdict
	}()
	via.Stream(ctx, 100*time.Millisecond, func(ctx *via.Ctx, _ time.Time) {
		select {
		case v := <-result:
			c.Verdict.Write(ctx, v)
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
