package packet_test

import (
	"testing"

	"github.com/joaomdsg/packets/internal/packet"
	"github.com/stretchr/testify/assert"
)

func TestLane_stringIsALowercaseMonoWordPerName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		lane packet.Lane
		want string
	}{
		{"unmeasured", packet.LaneUnmeasured, "unmeasured"},
		{"best-effort", packet.LaneBestEffort, "best-effort"},
		{"standard", packet.LaneStandard, "standard"},
		{"strict", packet.LaneStrict, "strict"},
		{"irreversible", packet.LaneIrreversible, "irreversible"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.lane.String())
		})
	}
}

func TestLane_stringFailsSafeOnAnOutOfRangeValue(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "unknown", packet.Lane(99).String())
}

// chainGraph is A -> B -> C -> D (A imports B, B imports C, C imports D):
// changing the leaf D ripples through every reverse-dependent up to A.
func chainGraph() packet.ImportGraph {
	return packet.ImportGraph{
		"a": {"b"},
		"b": {"c"},
		"c": {"d"},
		"d": {},
	}
}

// diamondGraph is A -> {B, C} -> D: two independent import paths converge on
// the same leaf, so D's reverse closure must not double-count A.
func diamondGraph() packet.ImportGraph {
	return packet.ImportGraph{
		"a": {"b", "c"},
		"b": {"d"},
		"c": {"d"},
		"d": {},
	}
}

func TestBlastRadius_changingALeafRipplesToEveryTransitiveImporter(t *testing.T) {
	t.Parallel()

	affected, total := packet.BlastRadius(chainGraph(), []string{"d"})

	assert.Equal(t, 4, affected, "d, c, b, a all sit on the reverse-dependency chain from d")
	assert.Equal(t, 4, total)
}

func TestBlastRadius_changingTheRootTouchesOnlyItself(t *testing.T) {
	t.Parallel()

	affected, total := packet.BlastRadius(chainGraph(), []string{"a"})

	assert.Equal(t, 1, affected, "nothing imports a, so no reverse-dependency ripples out from it")
	assert.Equal(t, 4, total)
}

func TestBlastRadius_diamondConvergenceDoesNotDoubleCountTheSharedAncestor(t *testing.T) {
	t.Parallel()

	affected, total := packet.BlastRadius(diamondGraph(), []string{"d"})

	assert.Equal(t, 4, affected, "a is reached via both b and c but must be counted once")
	assert.Equal(t, 4, total)
}

func TestBlastRadius_anIsolatedLeafWithNoImportersOrImportsAffectsOnlyItself(t *testing.T) {
	t.Parallel()

	graph := packet.ImportGraph{
		"a": {"b"},
		"b": {},
		"e": {}, // isolated: nothing imports it, it imports nothing
	}

	affected, total := packet.BlastRadius(graph, []string{"e"})

	assert.Equal(t, 1, affected)
	assert.Equal(t, 3, total)
}

func TestBlastRadius_changingEveryPackageAffectsTheWholeGraph(t *testing.T) {
	t.Parallel()

	graph := diamondGraph()

	affected, total := packet.BlastRadius(graph, []string{"a", "b", "c", "d"})

	assert.Equal(t, 4, affected)
	assert.Equal(t, 4, total)
}

func TestBlastRadius_multipleChangedPackagesUnionTheirClosuresWithoutDoubleCounting(t *testing.T) {
	t.Parallel()

	// a -> b -> c ; x -> y (disjoint component)
	graph := packet.ImportGraph{
		"a": {"b"},
		"b": {"c"},
		"c": {},
		"x": {"y"},
		"y": {},
	}

	affected, total := packet.BlastRadius(graph, []string{"c", "y"})

	assert.Equal(t, 5, affected, "c's closure {c,b,a} unions with y's closure {y,x}, five distinct packages")
	assert.Equal(t, 5, total)
}

func TestBlastRadius_emptyGraphMeasuresNothing(t *testing.T) {
	t.Parallel()

	affected, total := packet.BlastRadius(packet.ImportGraph{}, []string{"a"})

	assert.Equal(t, 1, affected, "a changed even though it names no known package")
	assert.Equal(t, 0, total)
}

func TestBlastRadius_noChangedPackagesAffectsNothing(t *testing.T) {
	t.Parallel()

	affected, total := packet.BlastRadius(chainGraph(), nil)

	assert.Equal(t, 0, affected)
	assert.Equal(t, 4, total)
}

func TestLaneFor_ratioAtOrBelowTenPercentIsBestEffort(t *testing.T) {
	t.Parallel()

	assert.Equal(t, packet.LaneBestEffort, packet.LaneFor(1, 10), "exact 10% boundary is inclusive")
	assert.Equal(t, packet.LaneBestEffort, packet.LaneFor(1, 100))
}

func TestLaneFor_ratioJustAboveTenPercentIsStandard(t *testing.T) {
	t.Parallel()

	assert.Equal(t, packet.LaneStandard, packet.LaneFor(11, 100), "11% crosses the best-effort ceiling")
}

func TestLaneFor_ratioAtFortyPercentIsStillStandard(t *testing.T) {
	t.Parallel()

	assert.Equal(t, packet.LaneStandard, packet.LaneFor(40, 100), "exact 40% boundary is inclusive")
}

func TestLaneFor_ratioJustAboveFortyPercentIsStrict(t *testing.T) {
	t.Parallel()

	assert.Equal(t, packet.LaneStrict, packet.LaneFor(41, 100))
}

func TestLaneFor_ratioOfOneHundredPercentIsStrict(t *testing.T) {
	t.Parallel()

	assert.Equal(t, packet.LaneStrict, packet.LaneFor(100, 100))
}

func TestLaneFor_zeroTotalPackagesIsUnmeasuredRegardlessOfAffected(t *testing.T) {
	t.Parallel()

	assert.Equal(t, packet.LaneUnmeasured, packet.LaneFor(0, 0))
	assert.Equal(t, packet.LaneUnmeasured, packet.LaneFor(1, 0), "an empty graph is unmeasurable even if BlastRadius reports a nonzero affected count")
}

func TestLaneFor_noChangedPackagesIsUnmeasuredEvenWithAKnownGraph(t *testing.T) {
	t.Parallel()

	assert.Equal(t, packet.LaneUnmeasured, packet.LaneFor(0, 100), "affected==0 means nothing Go-local changed to measure")
}

func TestLaneFor_neverProducesIrreversible(t *testing.T) {
	t.Parallel()

	// Coupling alone can never justify the irreversible lane: irreversibility is
	// about consequences (data/money/prod side-effects) no marker in this repo
	// declares yet — the same "unreachable until a real guardrail exists"
	// pattern as Packet.Deliverable. Swept across the whole ratio space.
	for total := 0; total <= 200; total += 7 {
		for affected := 0; affected <= total; affected += 3 {
			assert.NotEqual(t, packet.LaneIrreversible, packet.LaneFor(affected, total),
				"affected=%d total=%d", affected, total)
		}
	}
}

func TestBlastRadius_changedPackageThatIsOnlyAnImportValueStillFindsItsImporters(t *testing.T) {
	t.Parallel()

	// "z" is never a graph key (a sparse/partial graph) but "a" imports it — the
	// reverse closure must still find "a" by scanning import VALUES, not keys.
	graph := packet.ImportGraph{"a": {"z"}}

	affected, total := packet.BlastRadius(graph, []string{"z"})

	assert.Equal(t, 2, affected, "z itself plus a, which imports it")
	assert.Equal(t, 1, total, "total counts known packages (graph keys) only")
}
