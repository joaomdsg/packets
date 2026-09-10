package app_test

import (
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/go-via/via/h"
	"github.com/go-via/via/vt"

	"github.com/joaomdsg/packets/internal/app"
	"github.com/joaomdsg/packets/internal/catch"
	"github.com/joaomdsg/packets/internal/ledger"
	"github.com/joaomdsg/packets/internal/reanchor"
)

// benchCapMirror duplicates prep_bench.go's unexported benchCap — the bench's
// visible-item cap — since app_test has no access to the production constant.
const benchCapMirror = 5

// boardRefreshIntervalMirror duplicates board.go's unexported boardRefreshInterval
// — the board's SSE polling tick — since app_test has no access to the production
// constant.
const boardRefreshIntervalMirror = time.Second

// defaultKey names the primary/default session — the app_test-visible equivalent
// of the unexported defaultSessionKey ("default"), pinned here so a rename on the
// production side is a one-line fix instead of a repo-wide string hunt.
const defaultKey = "default"

// anchorForCap is a fixed, stable test anchor — duplicated from cap_internal_
// test.go (which stays internal) since app_test has no access to it.
func anchorForCap() reanchor.Anchor {
	return reanchor.Anchor{Path: "adult.go", Start: 4, End: 4, LineHash: "x"}
}

// bodyOf returns the <body> portion of a rendered page, dropping the <head> —
// duplicated for the external test package (the original helper stays in the
// internal test package). See that copy's doc comment for why: the head's
// stylesheet class names would otherwise be matched by a whole-page
// structural-index assertion.
func bodyOf(html string) string {
	if i := strings.Index(html, "</head>"); i >= 0 {
		return html[i:]
	}
	return html
}

// renderHTML renders a standalone h node to a string — duplicated for the
// external test package (the original helper stays in the internal test package).
func renderHTML(t *testing.T, node h.H) string {
	t.Helper()
	if node == nil {
		return ""
	}
	var b strings.Builder
	require.NoError(t, node.Render(&b))
	return b.String()
}

// woTargetN builds the Nth of a family of distinct, deterministic dispatch
// targets — duplicated for the external test package (the original helper stays
// in the internal test package).
func woTargetN(i int) ledger.Target {
	s := strconv.Itoa(i)
	return ledger.Target{BaseRev: "wo-base-" + s, FixRev: "wo-fix-" + s, TipRev: "wo-fix-" + s, Path: "other.go", Line: 9 + i}
}

// sendTarget is the single-target counterpart to woTargetN — duplicated
// for the external test package (the original helper stays in the internal test
// package).
func sendTarget() ledger.Target {
	return ledger.Target{BaseRev: "wo-base", FixRev: "wo-fix", TipRev: "wo-fix", Path: "other.go", Line: 9}
}

// ownTargetOf replicates the production (unexported) supply.ownTargetOf's pure
// derivation — a session's own cycle expressed as the Target shape AppendSend
// compares against to refuse re-funding a card's own work. Duplicated because the
// original lives in production code (supply.go), never exported for callers.
func ownTargetOf(cfg app.LiveConfig) ledger.Target {
	return ledger.Target{
		BaseRev: cfg.BaseRev, FixRev: cfg.FixRev, TipRev: cfg.TipRev,
		Path: cfg.Anchor.Path, Line: cfg.Anchor.Start, LineHash: cfg.Anchor.LineHash,
	}
}

// drainFramesFor collects every SSE frame that arrives within d, then returns the
// concatenation — duplicated for the external test package (the original helper
// stays in the internal test package).
// Used to assert the ABSENCE of an expected-not-to-happen frame.
func drainFramesFor(frames <-chan string, d time.Duration) string {
	deadline := time.After(d)
	var b strings.Builder
	for {
		select {
		case f, ok := <-frames:
			if !ok {
				return b.String()
			}
			b.WriteString(f)
		case <-deadline:
			return b.String()
		}
	}
}

// gitIn runs git in dir, failing the test on error — duplicated for the external
// test package (the original helper stays in the internal test package, blocked
// on resetBundleGuardsForTest).
func gitIn(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null")
	out, err := cmd.CombinedOutput()
	require.NoErrorf(t, err, "git %v: %s", args, out)
	return strings.TrimSpace(string(out))
}

// freshGitRepo is an empty host git store — duplicated for the external test
// package (the original helper stays in the internal test package).
func freshGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	gitIn(t, dir, "init", "-q")
	return dir
}

// gitPacket runs git in dir, failing the test on error — functionally identical to
// the shared runGit helper; kept as a separate name only where the migrated file's
// own call sites already read gitPacket(...), to keep each file's diff minimal.
func gitPacket(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "git %v\n%s", args, out)
	return strings.TrimSpace(string(out))
}

// initGitRepoForPacket seeds a minimal one-commit repo for order-fulfillment tests —
// duplicated for the external test package (the original helper stays in the
// internal test package).
func initGitRepoForPacket(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	gitPacket(t, dir, "init", "-q")
	gitPacket(t, dir, "config", "user.email", "t@t")
	gitPacket(t, dir, "config", "user.name", "t")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "base.txt"), []byte("base\n"), 0o644))
	gitPacket(t, dir, "add", "-A")
	gitPacket(t, dir, "commit", "-qm", "base")
	return dir
}

// statusOfPacket returns a dispatched order's current status ("" if unknown) —
// duplicated for the external test package (the original helper stays in the
// internal test package).
func statusOfPacket(t *testing.T, log *ledger.Log, id int) string {
	t.Helper()
	views, err := log.RecentSends(0)
	require.NoError(t, err)
	for _, v := range views {
		if v.ID == id {
			return v.Status
		}
	}
	return ""
}

// orderRecordFor returns the dispatched work-order with the given id (zero value
// when absent) — duplicated for the external test package (the original helper
// stays in the internal test package).
func orderRecordFor(t *testing.T, log *ledger.Log, id int) ledger.SendView {
	t.Helper()
	views, err := log.RecentSends(0)
	require.NoError(t, err)
	for _, v := range views {
		if v.ID == id {
			return v
		}
	}
	return ledger.SendView{}
}

// fundBandwidth gives a session's attention meter one earned interval (a fast-
// cleared block) — duplicated for the external test package (the original helper
// stays in the internal test package).
func fundBandwidth(t *testing.T, log *ledger.Log) {
	t.Helper()
	base := time.Unix(1_700_000_000, 0)
	require.NoError(t, log.AppendBlock("q1", base))
	require.NoError(t, log.AppendUnblock("q1", base.Add(30*time.Second))) // +3 bandwidth
}

// rowIndex, rowFor, requireBefore — duplicated for the external test package (the
// original helper stays in the internal test package, being the highest-risk
// fleet-wide enumerator). liveReg is a
// process-wide global shared by every test in this binary, so board assertions
// filter to their OWN keys rather than assume a clean fleet.
func rowIndex(rows []app.CardRow, key string) int {
	for i, r := range rows {
		if r.Key == key {
			return i
		}
	}
	return -1
}

func rowFor(t *testing.T, rows []app.CardRow, key string) app.CardRow {
	t.Helper()
	i := rowIndex(rows, key)
	require.GreaterOrEqual(t, i, 0, "the board must include a row for "+key)
	return rows[i]
}

func requireBefore(t *testing.T, rows []app.CardRow, earlier, later string) {
	t.Helper()
	ei, li := rowIndex(rows, earlier), rowIndex(rows, later)
	require.GreaterOrEqual(t, ei, 0, "the board must include a row for "+earlier)
	require.GreaterOrEqual(t, li, 0, "the board must include a row for "+later)
	require.Less(t, ei, li, earlier+" must sort before "+later)
}

// defaultBootCfg is the standard anchored default-session config the majority of
// migrated tests boot against when the default session's own state is untested —
// only the KEYED session under test carries meaningful config. A `var`, not a
// function, so every caller shares one literal instead of allocating a fresh copy.
var defaultBootCfg = app.LiveConfig{
	RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
	TestCmd: []string{"true"},
}

// bootDefaultServer stands up the default session via app.NewServer, filling in a
// scratch LedgerPath when the caller leaves it empty (pure plumbing — every other
// field is the caller's own test data, left exactly as they set it). Returns the
// httptest server and the default session's log.
func bootDefaultServer(t *testing.T, cfg app.LiveConfig) (*httptest.Server, *ledger.Log) {
	t.Helper()
	if cfg.LedgerPath == "" {
		cfg.LedgerPath = filepath.Join(t.TempDir(), "default.jsonl")
	}
	viaApp, log, err := app.NewServer(cfg)
	require.NoError(t, err)
	server := httptest.NewServer(viaApp)
	t.Cleanup(server.Close)
	t.Cleanup(func() { _ = log.Close() })
	return server, log
}

// addFundedSession registers a SECOND keyed session on an already-booted server
// (bootDefaultServer must run first — app.AddSession needs the server's fabric
// already started) and returns its log so the caller can fund/seed it. Replaces
// the internal registerSession(key, cfg, log) idiom, whose standalone fabric.Start
// this collapses onto the one shared fabric bootDefaultServer already started —
// behaviorally equivalent for test purposes (BoardRows/dispatch folding is
// per-session regardless of which physical JetStream store backs it), and closer
// to production topology besides (one server, one fabric, many session keys).
func addFundedSession(t *testing.T, key string, cfg app.LiveConfig) *ledger.Log {
	t.Helper()
	log, err := app.AddSession(key, cfg)
	require.NoError(t, err)
	return log
}

// boardSession funds a session with seedCatches confirmed catches and an optional
// dispatch backlog, returning its log — duplicated/rewritten for the external test
// package from the internal boardSession helper (which stays internal as part of
// the highest-risk board batch). Requires bootDefaultServer to have run first in
// the same test.
func boardSession(t *testing.T, key string, seedCatches int, backlog []ledger.Target) *ledger.Log {
	t.Helper()
	log := addFundedSession(t, key, app.LiveConfig{BaseRev: "own-b-" + key, FixRev: "own-f", Anchor: anchorForCap(), SendBacklog: backlog})
	for i := 0; i < seedCatches; i++ {
		require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Line: 100 + i, ReasonTag: "catch"}))
	}
	return log
}

// fundedAuthoringServer stands up the default session with a spendable balance
// (one confirmed catch) and earned bandwidth (one fast-cleared block) — every
// act-now affordance (spend, bench, authoring, place-order) renders. Rewritten
// from the internal fundedAuthoringServer helper: the keyed session now
// rides bootDefaultServer's own fabric via addFundedSession instead of a
// standalone one, mirroring the boardSession rewrite above.
func fundedAuthoringServer(t *testing.T, key string) (*ledger.Log, *httptest.Server) {
	t.Helper()
	server, _ := bootDefaultServer(t, app.LiveConfig{})
	log := addFundedSession(t, key, app.LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"},
	})
	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "seed.go", Line: 1, ReasonTag: "catch"}))
	fundBandwidth(t, log)
	return log, server
}

// actNowCardBody is fundedAuthoringServer's original single-purpose shape (a
// dispatch backlog instead of a bare seed catch) — rewritten from the internal
// actNowCardBody helper the same way.
func actNowCardBody(t *testing.T) string {
	t.Helper()
	server, _ := bootDefaultServer(t, app.LiveConfig{})
	log := addFundedSession(t, "actnow", app.LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"},
		SendBacklog: []ledger.Target{
			{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "deck.go", Line: 9},
		},
	})
	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "seed.go", Line: 1, ReasonTag: "catch"}))
	fundBandwidth(t, log)
	return bodyOf(vt.NewClient(t, server, "/?key=actnow").HTML())
}
