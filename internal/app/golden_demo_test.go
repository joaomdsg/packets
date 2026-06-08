package app_test

import (
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-via/via"
	"github.com/go-via/via/vt"

	"github.com/joaomdsg/packets/internal/app"
	"github.com/joaomdsg/packets/internal/ledger"
	"github.com/joaomdsg/packets/internal/reanchor"
)

// The golden fixture: two adjacent under-tested >= comparisons whose strengthened
// test (the fix) kills both boundary mutants, so the connect cycle catches at the
// anchor (line 4) and the from-catch candidates at lines 5 and 6 catch too (real
// compounds), while the candidate at line 7 (the closing brace, no mutable
// operator) is a real honest miss.
const goldenGo = "package adult\n\nfunc bothOK(a, b int) bool {\n\tokA := a >= 18\n\tokB := b >= 18\n\treturn okA && okB\n}\n"
const goldenWeak = "package adult\n\nimport \"testing\"\n\nfunc TestBoth(t *testing.T) {\n\tif !bothOK(25, 25) {\n\t\tt.Fatal(\"25,25\")\n\t}\n}\n"
const goldenStrong = "package adult\n\nimport \"testing\"\n\nfunc TestBoth(t *testing.T) {\n\tif !bothOK(25, 25) {\n\t\tt.Fatal(\"25,25\")\n\t}\n\tif bothOK(17, 25) {\n\t\tt.Fatal(\"17 a\")\n\t}\n\tif bothOK(25, 17) {\n\t\tt.Fatal(\"17 b\")\n\t}\n\tif !bothOK(18, 18) {\n\t\tt.Fatal(\"18,18\")\n\t}\n}\n"

func goldenRow(t *testing.T) app.CardRow {
	t.Helper()
	for _, r := range app.BoardRows() {
		if r.Key == "default" {
			return r
		}
	}
	t.Fatal("no default board row")
	return app.CardRow{}
}

func TestGoldenReplay_realLoopShowsAWinTwoCompoundsAndAnHonestMissAndReplaysConsistently(t *testing.T) {
	// A worked end-to-end DEMONSTRATION of the proven loop against the REAL oracle
	// (no resolveCycle seam): a real catch (the connect WIN), real from-catch
	// compounds (a spend's distinct work mints back), and a real honest MISS (a
	// neighbor line with no mutable operator), all replaying purely from the JSONL.
	// It asserts a LOSS and replay-consistency — not merely that the pipe runs —
	// directly answering worked-example-happy-path != validation.
	dir := initRepo(t)
	write(t, dir, "go.mod", "module adultapp\n\ngo 1.23\n")
	write(t, dir, "adult.go", goldenGo)
	write(t, dir, "adult_test.go", goldenWeak)
	base := commitAll(t, dir, "base: two under-tested comparisons")
	write(t, dir, "adult_test.go", goldenStrong)
	fix := commitAll(t, dir, "fix: strengthen both boundaries")

	logPath := filepath.Join(t.TempDir(), "catches.jsonl")
	var server *httptest.Server
	_, log, err := app.NewServer(app.LiveConfig{
		RepoDir: dir, BaseRev: base, FixRev: fix, TipRev: fix,
		Anchor:  reanchor.Anchor{Path: "adult.go", Start: 4, End: 4, LineHash: reanchor.HashLines("\tokA := a >= 18")},
		TestCmd: goTestCmd, LedgerPath: logPath,
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })

	tc := vt.NewClient(t, server, "/")
	frames, cancel := tc.SSE()
	defer cancel()
	vt.AwaitFrame(t, frames, 60*time.Second, `data-balance="1"`) // the connect WIN minted a catch

	win := goldenRow(t)
	require.Equal(t, 1, win.Confirmed, "the connect cycle caught the anchor line — a real WIN")
	require.Equal(t, 0, win.Reinvested, "nothing reinvested yet — the win is a connect mint, not a bet")

	// Three spends draw the from-catch supply: lines 5 and 6 catch (real compounds),
	// line 7 (the closing brace) is a real honest miss.
	for i := 1; i <= 3; i++ {
		require.Equal(t, 200, tc.Action((&app.LiveCard{}).Spend).Fire())
		done := i
		require.Eventually(t, func() bool { return goldenRow(t).Done >= done }, 60*time.Second, 200*time.Millisecond, "the dispatched candidate ran to done")
	}

	r := goldenRow(t)
	assert.Equal(t, 3, r.Confirmed, "1 connect win + 2 reinvested compounds")
	assert.Equal(t, 2, r.Reinvested, "two from-catch candidates minted back — the spend-to-earn loop paid off twice")
	assert.Equal(t, 1, r.Confirmed-r.Reinvested, "exactly one catch is connect-minted (the win); the rest are reinvestment")
	assert.Equal(t, 3, r.Done, "three dispatched bets resolved")
	assert.Equal(t, 1, r.Misses, "one bet was a real MISS — a neighbor line with no mutable operator; the loss is honest, not hidden")

	// Replay consistency: re-open the SAME JSONL into a fresh ledger and re-derive
	// the projections — identical, proving the economy is a pure replay of logged
	// events, holding no in-memory state.
	recs, err := log.Records()
	require.NoError(t, err)
	stock := ledger.ConfirmedCatches(recs)
	assert.Equal(t, r.Confirmed, stock.Count, "Confirmed replays identically from the persisted log")
	assert.Equal(t, r.Reinvested, stock.Reinvested, "Reinvested replays identically")
	counts, err := log.DispatchStatusCounts()
	require.NoError(t, err)
	assert.Equal(t, r.Done, counts.Done, "Done replays identically")

	// The headline standing renders on the board as an honest count ratio.
	board := vt.NewClient(t, server, "/board").HTML()
	assert.Contains(t, board, "hit-rate 2/3", "the board shows the earned standing: 2 hits of 3 resolved bets")
}
