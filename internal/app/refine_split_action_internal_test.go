package app

import (
	"context"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-via/via"
	"github.com/go-via/via/vt"

	"github.com/joaomdsg/packets/internal/ledger"
	"github.com/joaomdsg/packets/internal/pipe"
	"github.com/joaomdsg/packets/internal/reanchor"
)

func stubResolveCycleNoMint(t *testing.T) {
	t.Helper()
	restore := resolveCycle
	t.Cleanup(func() { resolveCycle = restore })
	resolveCycle = func(_ context.Context, _, _, _, _ string, _ reanchor.Anchor, _ []string, _, _ bool, _ chan<- pipe.TraceEvent) (Resolution, error) {
		return Resolution{}, nil
	}
}

func TestLiveCard_splitChosenSplitsTheTargetIntoItsChangedRegions(t *testing.T) {
	// The split path, end to end: the Lead asks to split a broad fundable target;
	// the system harvests its changed regions from the real diff and records a split
	// refinement into them (brick 2 then folds it into the fundable set). NOT parallel.
	stubResolveCycleNoMint(t)
	dir, base, fix := splitRepo(t)
	logPath := filepath.Join(t.TempDir(), "c.jsonl")
	var server *httptest.Server
	_, log, err := NewServer(LiveConfig{
		RepoDir: dir, BaseRev: "ownb", FixRev: "ownf", TipRev: "ownf", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: logPath,
		DispatchBacklog: []ledger.Target{{BaseRev: base, FixRev: fix, TipRev: fix, Path: "pay.go", Line: 10}},
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })

	tc := vt.NewClient(t, server, "/")
	require.Equal(t, 200, tc.Action((&LiveCard{}).SplitChosen).WithSignal("refinetarget", "pay.go:10").Fire())

	refs, err := log.Refinements()
	require.NoError(t, err)
	require.Len(t, refs, 1, "asking to split records exactly one split refinement")
	assert.Equal(t, "split", refs[0].Refine)
	require.GreaterOrEqual(t, len(refs[0].Splits), 2, "the split carries the changed regions harvested from the diff")
	for _, s := range refs[0].Splits {
		assert.Equal(t, "pay.go", s.Path, "each sub-target stays in the parent's file")
		assert.NotEqual(t, 10, s.Line, "no sub-target is the parent line itself")
	}
}

func TestLiveCard_splitChosenIsANoOpWhenThereIsNothingToSplitInto(t *testing.T) {
	// A target whose file the diff never touched has no changed regions to split
	// into: asking to split records nothing, never a fabricated or empty split. NOT
	// parallel.
	stubResolveCycleNoMint(t)
	dir, base, fix := splitRepo(t)
	logPath := filepath.Join(t.TempDir(), "c.jsonl")
	var server *httptest.Server
	_, log, err := NewServer(LiveConfig{
		RepoDir: dir, BaseRev: "ownb", FixRev: "ownf", TipRev: "ownf", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: logPath,
		DispatchBacklog: []ledger.Target{{BaseRev: base, FixRev: fix, TipRev: fix, Path: "untouched.go", Line: 3}},
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })

	tc := vt.NewClient(t, server, "/")
	require.Equal(t, 200, tc.Action((&LiveCard{}).SplitChosen).WithSignal("refinetarget", "untouched.go:3").Fire())

	refs, err := log.Refinements()
	require.NoError(t, err)
	require.Empty(t, refs, "nothing to split into → nothing recorded")
}

func TestLiveCard_benchCardOffersASplitFromTheDiff(t *testing.T) {
	// The sharpen body offers a split affordance wired to SplitChosen, so the Lead
	// can break a broad target into its changed regions. NOT parallel.
	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	_, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
		DispatchBacklog: []ledger.Target{{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "pay.go", Line: 88}},
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })

	body := bodyOf(vt.NewClient(t, server, "/").HTML())
	require.Contains(t, body, "bench__split", "the sharpen body has a split affordance")
	require.Contains(t, body, "SplitChosen", "the split affordance is wired to the SplitChosen action")
}
