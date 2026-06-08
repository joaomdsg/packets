package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/joaomdsg/packets/internal/diff"
	"github.com/joaomdsg/packets/internal/fabric"
)

// RevisionEvent is the JSON payload of a minted revision event on the bus — the
// changeset a consumer rebuilds review state from.
type RevisionEvent struct {
	SHA     string
	Added   int
	Deleted int
	Diff    diff.Diff
}

// PublishRevision emits a minted revision event on the canonical
// minted-revision subject for session+instance and returns its stream sequence.
//
// Only a genuinely minted outcome may reach the source-of-truth subject: an
// unminted turn (secret-blocked or no-edit) publishes nothing and returns an
// error, so a consumer can never rebuild state from a revision that never
// landed.
//
// session and instance are host-minted subject tokens: they must be non-empty
// and contain no '.', space, or NATS wildcard ('*'/'>'), since they are
// interpolated into the dotted subject. A token with those characters would
// corrupt the subject's token structure (extra tokens, or a wildcard published
// as a literal). This is the caller's contract, not validated here.
func PublishRevision(ctx context.Context, f *fabric.Fabric, session, instance string, out TurnOutcome) (uint64, error) {
	if !out.Minted {
		return 0, fmt.Errorf("orchestrator: refusing to publish revision for an unminted turn")
	}
	data, err := json.Marshal(RevisionEvent{SHA: out.SHA, Added: out.Added, Deleted: out.Deleted, Diff: out.Diff})
	if err != nil {
		return 0, fmt.Errorf("orchestrator: encode revision: %v", err)
	}
	return f.Publish(ctx, fabric.EventSubject(session, instance, fabric.StatusMinted, "revision"), data)
}

// DecodeRevision decodes a revision event payload from the bus.
func DecodeRevision(data []byte) (RevisionEvent, error) {
	var rev RevisionEvent
	if err := json.Unmarshal(data, &rev); err != nil {
		return RevisionEvent{}, fmt.Errorf("orchestrator: decode revision: %v", err)
	}
	return rev, nil
}
