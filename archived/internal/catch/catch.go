// Package catch is the confirmed-catch oracle: a pure two-revision differential
// layered over the mutation oracle (internal/mutation). It decides whether a
// fix turned a weakly-tested line into a constrained one, keyed by the line's
// OPERATOR INVENTORY per revision — never by individual mutant identity — so
// that the incoherent "the same mutant was killed" claim cannot be expressed
// across a fix that edits the line. It is the first economy primitive.
package catch

import (
	"sort"

	"github.com/joaomdsg/packets/internal/mutation"
)

// Outcome is the verdict of comparing a line's mutation survivor-set across two
// revisions.
type Outcome string

const (
	// Catch: a stable line's survivor-set went from non-empty to empty — the
	// fix genuinely constrained a previously-weak line.
	Catch Outcome = "catch"
	// NoCatch: no qualifying transition (already constrained, no progress, a
	// regression, or the line's operator alphabet changed so the comparison is
	// ill-typed).
	NoCatch Outcome = "no_catch"
	// NoOracleSignal: the pre-fix line had no mutable operators, so the oracle
	// says nothing about it — must never be read as "nothing caught".
	NoOracleSignal Outcome = "no_oracle_signal"
	// PartialCatch: the survivor-set strictly shrank but is not empty — better
	// constrained, not yet fully.
	PartialCatch Outcome = "partial_catch"
)

// LineState is a line's mutation state at one revision: the operator alphabet
// present on the line (Inventory) and the subset whose mutant survived
// (Survivors ⊆ Inventory). Both are treated as SETS; duplicate operator sites
// collapse to one entry (a deliberate v1 simplification that keys the
// denominator on the operator alphabet rather than unstable per-site identity).
type LineState struct {
	Inventory []string `json:"inventory"`
	Survivors []string `json:"survivors"`
	// Undetermined is the subset of operators whose mutant did NOT finish (the suite
	// timed out — mutation.Undetermined). Distinct from Survivors: a non-empty
	// Undetermined means the run was INCOMPLETE, so an empty Survivors cannot be read as
	// "the line was constrained". Detect fails closed on it, never minting a phantom
	// Catch from an oracle that merely hung.
	Undetermined []string `json:"undetermined"`
}

// Detect compares an anchored line's state before and after a fix and returns
// the catch outcome. The comparison is keyed on the operator inventory: if the
// fix changed the line's alphabet, the before/after survivor-sets live over
// different domains and no catch is minted.
func Detect(before, after LineState) Outcome {
	beforeInv := toSet(before.Inventory)
	if len(beforeInv) == 0 {
		return NoOracleSignal
	}
	if !setsEqual(beforeInv, toSet(after.Inventory)) {
		return NoCatch
	}
	beforeSurv := toSet(before.Survivors)
	afterSurv := toSet(after.Survivors)
	// Fail closed on an INCOMPLETE run on EITHER side: if any mutant timed out, that side's
	// survivor-set is partial — an incomplete after understates what the fix constrained,
	// an incomplete before understates the baseline it's compared against. A catch is
	// minted only from two COMPLETE runs; otherwise NoOracleSignal. Never mint a (phantom)
	// catch from "the oracle didn't finish".
	if len(before.Undetermined) > 0 || len(after.Undetermined) > 0 {
		return NoOracleSignal
	}
	if len(beforeSurv) == 0 {
		return NoCatch
	}
	if len(afterSurv) == 0 {
		return Catch
	}
	if isStrictSubset(afterSurv, beforeSurv) {
		return PartialCatch
	}
	return NoCatch
}

// LineStateAt derives a LineState for the given 1-based line of src: the
// Inventory is the operator alphabet GenerateMutants finds on that line, and
// Survivors are the operators whose mutant the run reported as Survived on that
// line. Undetermined (timed-out) findings are NOT confirmed survivors, so they are kept
// out of Survivors — but recorded separately in Undetermined so Detect can tell an
// incomplete run from a real catch (rather than silently dropping the signal).
func LineStateAt(src []byte, line int, res mutation.Result) (LineState, error) {
	mutants, err := mutation.GenerateMutants(src, []mutation.LineRange{{Start: line, End: line}})
	if err != nil {
		return LineState{}, err
	}
	var inv []string
	for _, m := range mutants {
		inv = append(inv, m.Original)
	}
	var surv, undet []string
	for _, f := range res.Findings {
		if f.Line != line {
			continue
		}
		switch f.Outcome {
		case mutation.Survived:
			surv = append(surv, f.Original)
		case mutation.Undetermined:
			undet = append(undet, f.Original)
		}
	}
	return LineState{Inventory: dedup(inv), Survivors: dedup(surv), Undetermined: dedup(undet)}, nil
}

func dedup(xs []string) []string {
	set := toSet(xs)
	out := make([]string, 0, len(set))
	for x := range set {
		out = append(out, x)
	}
	sort.Strings(out)
	return out
}

func toSet(xs []string) map[string]struct{} {
	set := make(map[string]struct{}, len(xs))
	for _, x := range xs {
		set[x] = struct{}{}
	}
	return set
}

func setsEqual(a, b map[string]struct{}) bool {
	if len(a) != len(b) {
		return false
	}
	for k := range a {
		if _, ok := b[k]; !ok {
			return false
		}
	}
	return true
}

func isStrictSubset(sub, super map[string]struct{}) bool {
	if len(sub) >= len(super) {
		return false
	}
	for k := range sub {
		if _, ok := super[k]; !ok {
			return false
		}
	}
	return true
}
