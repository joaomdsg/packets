package cage_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/packets/internal/cage"
	"github.com/joaomdsg/packets/internal/catch"
	"github.com/joaomdsg/packets/internal/ledger"
	"github.com/joaomdsg/packets/internal/pipe"
)

func anchorTarget() ledger.Target {
	return ledger.Target{
		BaseRev: "basesha", FixRev: "fixsha", TipRev: "fixsha",
		Path: "adult.go", Line: 4,
	}
}

// catchTranscript is an evidence-backed catch: a stable operator inventory whose
// survivor-set went from non-empty (before) to empty (after) on the anchored line.
func catchTranscript() pipe.Transcript {
	return pipe.Transcript{
		Outcome: catch.Catch,
		Reason:  pipe.ReasonNone,
		Path:    "adult.go",
		Line:    4,
		Land:    pipe.LandClean,
		Before:  catch.LineState{Inventory: []string{">=", "<"}, Survivors: []string{">="}},
		After:   catch.LineState{Inventory: []string{">=", "<"}, Survivors: nil},
	}
}

// A genuine catch — the cage's self-report agrees with the survivor-set evidence
// — mints a record. Crucially the record's revisions come from the TRUSTED
// target, never the transcript (which carries none), and the inventories come
// from the transcript the host re-derived over.
func TestDeriveCatch_mintsAGenuineEvidenceBackedCatch(t *testing.T) {
	t.Parallel()
	target := anchorTarget()

	rec, err := cage.DeriveCatch(catchTranscript(), target)
	require.NoError(t, err)
	require.NotNil(t, rec, "an evidence-backed catch must mint a record")

	assert.Equal(t, catch.Catch, rec.Outcome)
	assert.Equal(t, "adult.go", rec.Path)
	assert.Equal(t, 4, rec.Line)
	assert.Equal(t, target.BaseRev, rec.BeforeRev, "the before-rev must be the trusted target's, not anything the cage reported")
	assert.Equal(t, target.FixRev, rec.AfterRev, "the after-rev must be the trusted target's, not anything the cage reported")
	assert.Equal(t, []string{">=", "<"}, rec.BeforeInventory)
	assert.Equal(t, []string{">=", "<"}, rec.AfterInventory)
	assert.Equal(t, 2, rec.MutantsConsidered)
	assert.Equal(t, string(catch.Catch), rec.ReasonTag)
	assert.False(t, rec.SelfFlagged)
	assert.False(t, rec.WouldHaveShipped)
}

// Any non-catch verdict whose self-report agrees with the evidence mints
// nothing — and that is NOT an error, just an honest negative verdict. Only a
// fully-eliminated survivor-set is a recordable catch; everything else (no
// progress, partial progress, or no oracle signal at all) records nothing.
func TestDeriveCatch_anHonestNonCatchMintsNothing(t *testing.T) {
	t.Parallel()
	honest := []struct {
		name    string
		outcome catch.Outcome
		before  catch.LineState
		after   catch.LineState
	}{
		{"no progress", catch.NoCatch,
			catch.LineState{Inventory: []string{">=", "<"}, Survivors: []string{">="}},
			catch.LineState{Inventory: []string{">=", "<"}, Survivors: []string{">="}}},
		{"partial progress", catch.PartialCatch,
			catch.LineState{Inventory: []string{">=", "<"}, Survivors: []string{">=", "<"}},
			catch.LineState{Inventory: []string{">=", "<"}, Survivors: []string{">="}}},
		{"no oracle signal (empty before-inventory)", catch.NoOracleSignal,
			catch.LineState{Inventory: nil, Survivors: nil},
			catch.LineState{Inventory: nil, Survivors: nil}},
	}
	for _, h := range honest {
		t.Run(h.name, func(t *testing.T) {
			t.Parallel()
			tr := catchTranscript()
			tr.Outcome, tr.Before, tr.After = h.outcome, h.before, h.after
			rec, err := cage.DeriveCatch(tr, anchorTarget())
			require.NoError(t, err, "an honest non-catch is a negative verdict, not an error")
			assert.Nil(t, rec, "an honest non-catch mints nothing")
		})
	}
}

// The lie-green trap, BOTH directions: the host trusts the recomputed evidence,
// never the self-reported Outcome, so ANY disagreement between the two is
// refused and mints nothing — a cage over-claiming a catch its evidence does not
// support, AND a transcript whose self-report otherwise contradicts its evidence.
func TestDeriveCatch_refusesAnyDisagreementBetweenSelfReportAndEvidence(t *testing.T) {
	t.Parallel()
	lies := []struct {
		name    string
		outcome catch.Outcome
		after   catch.LineState
	}{
		{"claims catch but survivors only shrank (partial)", catch.Catch, catch.LineState{Inventory: []string{">=", "<"}, Survivors: []string{">="}}},
		{"claims catch but survivors unchanged (no progress)", catch.Catch, catch.LineState{Inventory: []string{">=", "<"}, Survivors: []string{">=", "<"}}},
		{"claims catch but inventory changed (ill-typed)", catch.Catch, catch.LineState{Inventory: []string{"!="}, Survivors: nil}},
		{"evidence is a catch but self-report under-claims no_catch", catch.NoCatch, catch.LineState{Inventory: []string{">=", "<"}, Survivors: nil}},
		{"evidence is a catch but self-report mislabels partial", catch.PartialCatch, catch.LineState{Inventory: []string{">=", "<"}, Survivors: nil}},
	}
	for _, l := range lies {
		t.Run(l.name, func(t *testing.T) {
			t.Parallel()
			tr := catchTranscript() // Before is the catch evidence; After/Outcome overridden per case
			tr.Outcome, tr.After = l.outcome, l.after
			rec, err := cage.DeriveCatch(tr, anchorTarget())
			require.Error(t, err, "a self-report that disagrees with the evidence must be refused")
			assert.Nil(t, rec, "a refused claim mints nothing")
		})
	}
}

// An incomplete transcript (no anchored path, or a non-positive line) cannot be
// trusted to describe a verdict and is refused.
func TestDeriveCatch_refusesAnIncompleteTranscript(t *testing.T) {
	t.Parallel()
	bad := []struct {
		name   string
		mutate func(*pipe.Transcript)
	}{
		{"empty path", func(tr *pipe.Transcript) { tr.Path = "" }},
		{"non-positive line", func(tr *pipe.Transcript) { tr.Line = 0 }},
	}
	for _, b := range bad {
		t.Run(b.name, func(t *testing.T) {
			t.Parallel()
			tr := catchTranscript()
			b.mutate(&tr)
			// keep the target matching the original anchor so the failure is the
			// incompleteness, not the anchor-mismatch check
			rec, err := cage.DeriveCatch(tr, anchorTarget())
			require.Error(t, err)
			assert.Nil(t, rec)
		})
	}
}

// A transcript for a DIFFERENT anchor than the one the host asked the cage to
// verify means the cage verified the wrong thing — refuse it rather than mint a
// catch attributed to the wrong line.
func TestDeriveCatch_refusesATranscriptForADifferentAnchor(t *testing.T) {
	t.Parallel()
	mismatched := []struct {
		name   string
		target ledger.Target
	}{
		{"different path", ledger.Target{BaseRev: "basesha", FixRev: "fixsha", Path: "other.go", Line: 4}},
		{"different line", ledger.Target{BaseRev: "basesha", FixRev: "fixsha", Path: "adult.go", Line: 9}},
	}
	for _, m := range mismatched {
		t.Run(m.name, func(t *testing.T) {
			t.Parallel()
			rec, err := cage.DeriveCatch(catchTranscript(), m.target)
			require.Error(t, err, "a transcript whose anchor differs from the target must be refused")
			assert.Nil(t, rec)
		})
	}
}
