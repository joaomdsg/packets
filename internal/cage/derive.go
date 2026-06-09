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

	// The verdict is whatever the evidence says, recomputed here — the cage's
	// self-report is only cross-checked against it. Any disagreement is a refusal.
	derived := catch.Detect(t.Before, t.After)
	if derived != t.Outcome {
		return nil, fmt.Errorf("cage: self-reported outcome %q disagrees with the evidence (%q) — refused", t.Outcome, derived)
	}

	return ledger.NewCatchRecord(derived, t.Path, t.Line, target.BaseRev, target.FixRev, t.Before.Inventory, t.After.Inventory, false, false), nil
}
