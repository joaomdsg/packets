package app

import (
	"context"

	"github.com/joaomdsg/packets/internal/ledger"
)

// InProcVerifier bridges the claim seam to the real catch oracle, in-process: it
// runs the SAME Resolve (and thus pipe.RunCatchCycle) the live review loop uses,
// over repoDir at the claim's revisions, and returns the resulting CatchRecord
// (nil when there is no catch). It is the verifier the host runs over a repo it
// already holds — the reference the #6c equivalence lock compares the sandboxed
// verdict against, and the trusted-use bridge for ledger.ConsumeClaims.
//
// It deliberately runs the oracle IN-PROCESS, so it executes the work's tests in
// the host process. That is safe only for TRUSTED work; it must NOT be wired as
// the consumer of UNTRUSTED claims — those are verified only inside the #6c
// sandbox. testCmd is the host-fixed suite command (never agent-supplied).
func InProcVerifier(repoDir string, testCmd []string) ledger.Verifier {
	return func(c ledger.ClaimRecord) (*ledger.CatchRecord, error) {
		t := c.Target
		res, err := Resolve(context.Background(), repoDir, t.BaseRev, t.FixRev, t.TipRev, anchorFromTarget(t), testCmd, false, false)
		if err != nil {
			return nil, err
		}
		return res.Record, nil
	}
}
