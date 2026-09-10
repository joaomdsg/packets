package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/joaomdsg/packets/internal/fabric"
	"github.com/joaomdsg/packets/internal/translate"
)

// ActivityEvent is the JSON payload of one turn's live agent activity on the
// bus — the thinking/editing/tool beats the surface renders as a run happens.
type ActivityEvent struct {
	Events []translate.UIEvent
}

// PublishActivity emits a live agent's activity batch on the scratch/activity
// subject for session+instance and returns its stream sequence.
//
// Activity rides the SCRATCH status, never minted: it is non-authoritative
// diagnostic the surface watches, and must never be replayed into
// source-of-truth state (the economy firewall — only minted revisions feed the
// catch economy; activity is watchable-but-unscored).
//
// An empty batch is refused: it carries no information and would be bus noise
// (and a needless scratch refold for every live viewer), mirroring how
// PublishRevision refuses an unminted turn.
//
// session and instance are host-minted subject tokens; the caller's contract
// (non-empty, no '.'/space/wildcard) matches PublishRevision's.
func PublishActivity(ctx context.Context, f *fabric.Fabric, session, instance string, events []translate.UIEvent) (uint64, error) {
	if len(events) == 0 {
		return 0, fmt.Errorf("orchestrator: refusing to publish an empty activity batch")
	}
	data, err := json.Marshal(ActivityEvent{Events: events})
	if err != nil {
		return 0, fmt.Errorf("orchestrator: encode activity: %v", err)
	}
	return f.Publish(ctx, fabric.EventSubject(session, instance, fabric.StatusScratch, "activity"), data)
}

// DecodeActivity decodes an activity event payload from the bus.
func DecodeActivity(data []byte) ([]translate.UIEvent, error) {
	var a ActivityEvent
	if err := json.Unmarshal(data, &a); err != nil {
		return nil, fmt.Errorf("orchestrator: decode activity: %v", err)
	}
	return a.Events, nil
}
