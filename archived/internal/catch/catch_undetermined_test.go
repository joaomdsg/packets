package catch_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/joaomdsg/packets/internal/catch"
)

// The phantom-catch regression: if the after-fix oracle run did NOT finish (its mutants
// timed out → Undetermined), the survivor-set is empty only because the run was
// incomplete, not because the line was constrained. Minting a Catch here would credit a
// fix for merely making the suite hang. Detect must fail closed to NoOracleSignal.
func TestDetect_refusesPhantomCatchWhenAfterRunIsIncomplete(t *testing.T) {
	t.Parallel()
	before := catch.LineState{Inventory: []string{"=="}, Survivors: []string{"=="}}
	after := catch.LineState{Inventory: []string{"=="}, Survivors: nil, Undetermined: []string{"=="}}
	assert.Equal(t, catch.NoOracleSignal, catch.Detect(before, after),
		"an incomplete after-run must never mint a Catch from an empty survivor-set")
}

// A genuinely complete after-run that killed every survivor still mints a Catch — the
// fail-closed guard must not over-suppress real catches.
func TestDetect_stillMintsCatchWhenAfterRunCompletedAndKilledAll(t *testing.T) {
	t.Parallel()
	before := catch.LineState{Inventory: []string{"=="}, Survivors: []string{"=="}}
	after := catch.LineState{Inventory: []string{"=="}, Survivors: nil, Undetermined: nil}
	assert.Equal(t, catch.Catch, catch.Detect(before, after),
		"a complete run that cleared the survivors is a real catch")
}

// Even when the after survivor-set merely shrank (a partial-looking transition), an
// incomplete run taints the comparison — we cannot trust a shrink we didn't fully
// observe. Fail closed rather than mint a PartialCatch.
func TestDetect_refusesPartialCatchWhenAfterRunIsIncomplete(t *testing.T) {
	t.Parallel()
	before := catch.LineState{Inventory: []string{"==", "&&"}, Survivors: []string{"==", "&&"}}
	after := catch.LineState{Inventory: []string{"==", "&&"}, Survivors: []string{"=="}, Undetermined: []string{"&&"}}
	assert.Equal(t, catch.NoOracleSignal, catch.Detect(before, after),
		"a shrink observed through an incomplete run is not a trustworthy partial catch")
}

// A catch must be minted only from two COMPLETE oracle runs. An incomplete BEFORE run
// understates the before survivor-set (a timed-out before-mutant is recorded as
// Undetermined, not Survived), so the before→after comparison runs over an untrustworthy
// baseline — fail closed to NoOracleSignal even when the after run is clean, rather than
// credit a catch we cannot stand behind.
func TestDetect_failsClosedWhenBeforeRunIsIncomplete(t *testing.T) {
	t.Parallel()
	before := catch.LineState{Inventory: []string{"==", "&&"}, Survivors: []string{"=="}, Undetermined: []string{"&&"}}
	after := catch.LineState{Inventory: []string{"==", "&&"}, Survivors: nil, Undetermined: nil}
	assert.Equal(t, catch.NoOracleSignal, catch.Detect(before, after),
		"an incomplete before-run is an untrustworthy baseline — no catch, even with a clean after")
}

// Both runs incomplete is the clearest no-signal case — fail closed, never a catch.
func TestDetect_failsClosedWhenBothRunsAreIncomplete(t *testing.T) {
	t.Parallel()
	before := catch.LineState{Inventory: []string{"==", "&&"}, Survivors: []string{"=="}, Undetermined: []string{"&&"}}
	after := catch.LineState{Inventory: []string{"==", "&&"}, Survivors: nil, Undetermined: []string{"=="}}
	assert.Equal(t, catch.NoOracleSignal, catch.Detect(before, after),
		"two incomplete runs give the oracle no trustworthy signal")
}

// The inventory-change precedence still wins over the incompleteness guard: a real
// operator-alphabet change is ill-typed regardless of run completeness, so it is NoCatch,
// never NoOracleSignal — proving the guard sits after the inventory check.
func TestDetect_inventoryMismatchWinsOverBeforeIncompleteness(t *testing.T) {
	t.Parallel()
	before := catch.LineState{Inventory: []string{"==", "&&"}, Survivors: []string{"=="}, Undetermined: []string{"&&"}}
	after := catch.LineState{Inventory: []string{"=="}, Survivors: nil, Undetermined: nil} // alphabet changed
	assert.Equal(t, catch.NoCatch, catch.Detect(before, after),
		"an operator-alphabet change is NoCatch even when the before run was incomplete")
}

// Both runs complete with a cleared survivor-set still mints a Catch — the either-side
// guard must not over-suppress when neither side is incomplete.
func TestDetect_mintsCatchWhenBothRunsComplete(t *testing.T) {
	t.Parallel()
	before := catch.LineState{Inventory: []string{"=="}, Survivors: []string{"=="}, Undetermined: nil}
	after := catch.LineState{Inventory: []string{"=="}, Survivors: nil, Undetermined: nil}
	assert.Equal(t, catch.Catch, catch.Detect(before, after),
		"two complete runs with a cleared survivor-set is a real catch")
}

// A complete partial transition (survivors shrank, nothing undetermined) still reports
// PartialCatch — the guard only fires on incompleteness.
func TestDetect_stillReportsPartialCatchWhenAfterRunIsComplete(t *testing.T) {
	t.Parallel()
	before := catch.LineState{Inventory: []string{"==", "&&"}, Survivors: []string{"==", "&&"}}
	after := catch.LineState{Inventory: []string{"==", "&&"}, Survivors: []string{"=="}, Undetermined: nil}
	assert.Equal(t, catch.PartialCatch, catch.Detect(before, after),
		"a complete shrink is still a partial catch")
}
