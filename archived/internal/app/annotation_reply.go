package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-via/via"

	"github.com/joaomdsg/packets/internal/assist"
	"github.com/joaomdsg/packets/internal/ledger"
)

// findAnnotation returns the annotation with the given id, or ok=false when
// none matches — a reply must anchor to a parent that really exists, never a
// fabricated one.
func findAnnotation(anns []ledger.AnnotationRecord, id string) (ledger.AnnotationRecord, bool) {
	for _, a := range anns {
		if a.ID == id {
			return a, true
		}
	}
	return ledger.AnnotationRecord{}, false
}

// ReplyToAnnotation posts a reply to an existing annotation: it persists a
// durable reply (ParentID set, inheriting the parent's file/line anchor) and
// re-triggers the harness with the reply as the turn — the human's answer both
// joins the thread and tells the agent what to change. Like AddAdjustment, the
// reply is persisted BEFORE the send, so it stays on the log even when the
// re-trigger is refused for budget. An empty reply, a missing parent, or a
// treeless session is a silent no-op.
func (c *ReviewCard) ReplyToAnnotation(ctx *via.Ctx) {
	key := c.Key
	if key == "" {
		key = defaultSessionKey
	}
	cfg, log := readLiveState(key)
	if log == nil {
		return
	}
	parentID := strings.TrimSpace(c.ReplyParent.Read(ctx))
	text := strings.TrimSpace(c.ReplyText.Read(ctx))
	if parentID == "" || text == "" {
		return // need both a target and words
	}
	anns, _ := log.Annotations()
	parent, ok := findAnnotation(anns, parentID)
	if !ok {
		return // never reply to a phantom
	}

	// Persist the reply before the re-trigger, inheriting the parent's anchor so
	// the thread stays anchored to one place. The id is sequential over existing
	// annotations, giving this reply its own stable handle.
	_ = log.AppendAnnotation(ledger.AnnotationRecord{
		ID:        fmt.Sprintf("ann-%d", len(anns)+1),
		ParentID:  parentID,
		File:      parent.File,
		StartLine: parent.StartLine,
		EndLine:   parent.EndLine,
		Author:    "lead",
		Body:      text,
		AtUnixMs:  time.Now().UnixMilli(),
	})

	head, ok := repoHead(cfg.RepoDir)
	if !ok {
		return // no resolvable tree to route the turn against
	}
	codeLine := readSourceLine(cfg.RepoDir, parent.File, parent.StartLine)
	tgt := ledger.Target{BaseRev: head, Prompt: assist.ReviewTurnPrompt(parent.File, parent.StartLine, codeLine, text)}
	if err := log.AppendLiveSend("liveorder", tgt, ownTargetOf(cfg)); err != nil {
		return
	}
	go drainQueuedPackets(key)
}
