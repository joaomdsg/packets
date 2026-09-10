package ledger

import (
	"context"
	"fmt"
)

// kindAnnotation tags a human-authored code annotation (or a reply to one): a
// durable, anchored comment on the reviewed diff. It shares the append-only
// stream and is distinguished by Kind=="annotation". Unlike the old in-memory
// adjustment anchors, an annotation survives a restart because it lives on the
// log; the read model folds annotations by file:line and nests replies under
// their parent.
const kindAnnotation = "annotation"

// AnnotationRecord is one durable annotation or reply. A top-level annotation has
// an empty ParentID and anchors to a file (StartLine 0) or a line/line-range
// (StartLine..EndLine, EndLine==0 meaning a single line). A reply sets ParentID
// to the id of the annotation it answers, so the read model can nest a threaded
// conversation. It is NEVER an economic event — it mints no balance and funds no
// packet; a reply that re-triggers the agent does so through a separate live
// send, not by this record's fold.
type AnnotationRecord struct {
	Kind      string `json:"kind"`
	ID        string `json:"id"`
	ParentID  string `json:"parent_id,omitempty"`
	File      string `json:"file"`
	StartLine int    `json:"start_line,omitempty"`
	EndLine   int    `json:"end_line,omitempty"`
	Author    string `json:"author"`
	Body      string `json:"body"`
	AtUnixMs  int64  `json:"at_unix_ms,omitempty"`
}

// AppendAnnotation records an annotation as a NEW append-only line — never
// mutating any prior record, so the log stays a pure append-only substrate and
// the annotation (or reply) replays back for the read model. It stamps Kind so a
// caller need not know the wire discriminator.
func (l *Log) AppendAnnotation(r AnnotationRecord) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	r.Kind = kindAnnotation
	if _, err := PublishAnnotation(context.Background(), l.f, l.session, l.instance, r); err != nil {
		return fmt.Errorf("ledger: append annotation: %w", err)
	}
	return nil
}

// Annotations is the annotation ledger in append order — the durable comments and
// replies the Inspector's read model folds into threads on read.
func (l *Log) Annotations() ([]AnnotationRecord, error) {
	p, err := l.project()
	if err != nil {
		return nil, err
	}
	return p.Annotations(), nil
}
