package ledger

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/joaomdsg/packets/internal/fabric"
)

// subjectKindClaim is the claim-subtree token for a unit of work submitted for
// verification — distinct from the minted-catch kind, so a claim and a catch can
// never be confused on the bus.
const subjectKindClaim = "work"

// ClaimRecord is an untrusted producer's work-submission: the revs and anchored
// line (a Target) the host must VERIFY before it mints anything. It deliberately
// carries NO test command — the host fixes what it runs, so a producer cannot
// choose the command executed on its behalf — and it is published on the claim
// subtree, never the minted subtree, so a claim credits nothing until a host-run
// oracle confirms it.
type ClaimRecord struct {
	Target Target `json:"target"`
}

// PublishClaim emits a producer's work-submission on the claim subtree for
// session+instance and returns its stream sequence. It targets StatusClaim, not
// StatusMinted, so it never enters the economy projection — the host consumes it,
// verifies it, and only then mints through the authoritative catch path.
func PublishClaim(ctx context.Context, f *fabric.Fabric, session, instance string, c ClaimRecord) (uint64, error) {
	data, err := json.Marshal(c)
	if err != nil {
		return 0, fmt.Errorf("ledger: encode claim: %v", err)
	}
	return f.Publish(ctx, fabric.EventSubject(session, instance, fabric.StatusClaim, subjectKindClaim), data)
}

// DecodeClaim decodes a producer work-submission payload from the bus.
func DecodeClaim(data []byte) (ClaimRecord, error) {
	var c ClaimRecord
	if err := json.Unmarshal(data, &c); err != nil {
		return ClaimRecord{}, fmt.Errorf("ledger: decode claim: %v", err)
	}
	return c, nil
}
