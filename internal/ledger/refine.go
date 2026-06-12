package ledger

import (
	"context"
	"fmt"
)

// kindWORefine tags a refined-work-order line: a Lead's dead-air sharpening of a
// backlog target (split / acceptance-criteria / accepted-convention). It shares
// the append-only stream and is distinguished by Kind=="worefine".
const kindWORefine = "worefine"

// RefinedOrderRecord is a Lead's sharpening of a backlog target during dead-air
// (VISION §12.1/§13.4): a split into sub-targets, attached acceptance criteria, or
// an accepted convention. It is NEVER an economic event — it mints no balance and
// funds no work-order on its own. The backlog projection folds it on read: a split
// replaces its parent Target with Splits; criteria/convention annotate the target.
type RefinedOrderRecord struct {
	Kind     string   `json:"kind"`
	RefineID int      `json:"refine_id"`
	Target   Target   `json:"target"`
	Refine   string   `json:"refine"`
	Splits   []Target `json:"splits,omitempty"`
	Criteria []string `json:"criteria,omitempty"`
	Note     string   `json:"note,omitempty"`
}

// AppendRefine records a refinement as a NEW append-only line — never mutating the
// target or any order, so the log stays a pure append-only substrate and the
// sharpening replays back for the backlog fold. It stamps Kind so a caller need not
// know the wire discriminator.
func (l *Log) AppendRefine(r RefinedOrderRecord) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	r.Kind = kindWORefine
	if _, err := PublishRefine(context.Background(), l.f, l.session, l.instance, r); err != nil {
		return fmt.Errorf("ledger: append refine: %w", err)
	}
	return nil
}

// Refinements is the refined-work-order ledger in append order — the sharpening
// facts the backlog projection folds on read.
func (l *Log) Refinements() ([]RefinedOrderRecord, error) {
	p, err := l.project()
	if err != nil {
		return nil, err
	}
	return p.Refinements(), nil
}
