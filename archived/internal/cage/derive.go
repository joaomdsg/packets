package cage

import (
	"fmt"

	"github.com/joaomdsg/packets/internal/catch"
	"github.com/joaomdsg/packets/internal/ledger"
	"github.com/joaomdsg/packets/internal/pipe"
)

// DeriveCatch re-derives the host's verdict from a sandboxed verifier's
// transcript, trusting ONLY the survivor-set evidence (Before/After) — never the
// cage's self-reported Outcome nor any process exit code. It returns the catch to
// mint (nil on an honest non-catch), or an error when the transcript cannot be
// trusted: this is the lie-green trap.
//
// The transcript carries no revisions, so the record's BeforeRev/AfterRev come
// from the TRUSTED target — a cage cannot forge which revs were compared. The
// record is built through ledger.NewCatchRecord, the one mint construction site.
func DeriveCatch(t pipe.Transcript, target ledger.Target) (*ledger.CatchRecord, error) {
	if t.Path == "" || t.Line < 1 {
		return nil, fmt.Errorf("cage: incomplete transcript: path=%q line=%d", t.Path, t.Line)
	}
	if t.Path != target.Path || t.Line != target.Line {
		return nil, fmt.Errorf("cage: transcript anchor %s:%d does not match the target anchor %s:%d", t.Path, t.Line, target.Path, target.Line)
	}

	// Well-formedness gate: the evidence comes from an untrusted (possibly buggy or
	// hostile) cage. catch.LineState declares the invariant Survivors/Undetermined ⊆
	// Inventory, but Detect computes set relations over whatever is reported — a phantom
	// out-of-alphabet operator could flip the verdict (e.g. forge a PartialCatch), and the
	// self-report would agree with that forged computation. Refuse malformed evidence
	// wholesale rather than trust the verdict it implies.
	if op, ok := escapesInventory(t.Before); ok {
		return nil, fmt.Errorf("cage: malformed transcript — before-state operator %q escapes its inventory (Survivors/Undetermined ⊆ Inventory) — refused", op)
	}
	if op, ok := escapesInventory(t.After); ok {
		return nil, fmt.Errorf("cage: malformed transcript — after-state operator %q escapes its inventory (Survivors/Undetermined ⊆ Inventory) — refused", op)
	}

	// The verdict is whatever the evidence says, recomputed here — the cage's
	// self-report is only cross-checked against it. Any disagreement is a refusal.
	derived := catch.Detect(t.Before, t.After)
	if derived != t.Outcome {
		return nil, fmt.Errorf("cage: self-reported outcome %q disagrees with the evidence (%q) — refused", t.Outcome, derived)
	}

	return ledger.NewCatchRecord(derived, t.Path, t.Line, target.BaseRev, target.FixRev, t.Before.Inventory, t.After.Inventory, false, false), nil
}

// escapesInventory returns the first operator in ls.Survivors or ls.Undetermined that is
// not present in ls.Inventory — a violation of the declared Survivors/Undetermined ⊆
// Inventory invariant, which marks the LineState as malformed (buggy/hostile cage). An
// empty Survivors/Undetermined is trivially well-formed (the loop never runs).
func escapesInventory(ls catch.LineState) (string, bool) {
	inv := make(map[string]bool, len(ls.Inventory))
	for _, op := range ls.Inventory {
		inv[op] = true
	}
	for _, op := range ls.Survivors {
		if !inv[op] {
			return op, true
		}
	}
	for _, op := range ls.Undetermined {
		if !inv[op] {
			return op, true
		}
	}
	return "", false
}
