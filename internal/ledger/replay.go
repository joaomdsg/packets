package ledger

import (
	"context"
	"fmt"
	"strings"

	"github.com/joaomdsg/packets/internal/fabric"
)

// Projection is the economy state folded from an ordered ledger event sequence,
// independent of where the bytes live. It exposes the SAME read projections as
// Log — the substrate-independent economy logic the migration moves unchanged —
// so a caller can fold from the stream exactly what the JSONL scan folds from the
// file. Its methods mirror Log's read methods minus the error: the events are
// already in memory, so there is no I/O to fail.
type Projection struct {
	catches []CatchRecord
	balance int
	orders  []WorkOrderRecord
	status  map[int]string
}

// Balance is credits (confirmed catches) minus debits (positive spends), folded
// identically to Log.Balance — a non-positive spend amount never debits, and a
// non-catch outcome never credits.
func (p Projection) Balance() int { return p.balance }

// Records is the catch-kind record stream, UNFILTERED by ShouldRecord (mirroring
// Log.Records): a forged non-catch line survives the projection, while never
// contributing to Balance or the confirmed stock.
func (p Projection) Records() []CatchRecord { return p.catches }

// WorkOrders is the funded work-order ledger in funding (id) order.
func (p Projection) WorkOrders() []WorkOrderRecord { return p.orders }

// DispatchStatusCounts tallies the orders by CURRENT status (last status line
// per id wins; an unknown status counts as queued), mirroring Log.
func (p Projection) DispatchStatusCounts() DispatchCounts {
	var c DispatchCounts
	for _, o := range p.orders {
		switch p.status[o.ID] {
		case "running":
			c.Running++
		case "done":
			c.Done++
		default:
			c.Queued++
		}
	}
	return c
}

// QueuedWorkOrders returns the orders whose current status is exactly queued, in
// funding order — the runner's input, mirroring Log.QueuedWorkOrders.
func (p Projection) QueuedWorkOrders() []WorkOrderRecord {
	var queued []WorkOrderRecord
	for _, o := range p.orders {
		if p.status[o.ID] == "queued" {
			queued = append(queued, o)
		}
	}
	return queued
}

// ReplayProjection folds the economy state for session+instance from the fabric:
// it replays every MINTED ledger event (the authoritative source-of-truth
// subtree, demuxed from any scratch fan-out), decodes each by its subject kind,
// and folds the SAME projections the JSONL scan produces. The events keep their
// authoritative global sequence order, so the order-dependent status fold
// (last-writer-wins per id) matches the append-order file scan.
func ReplayProjection(ctx context.Context, f *fabric.Fabric, session, instance string) (Projection, error) {
	filter := fabric.EventSubject(session, instance, fabric.StatusMinted, "*")
	events, err := f.ReplaySubject(ctx, filter)
	if err != nil {
		return Projection{}, err
	}
	return foldEvents(events)
}

// FleetProjection folds every session's economy from the fabric in one replay:
// it replays the cross-session minted subtree, groups events by their session
// token, and folds each group with the same canonical fold ReplayProjection
// uses — returning one Projection per session. This is the cross-process
// aggregator: the fleet board derives from the authoritative stream, not from
// any in-process registry, so it reflects sessions written by any producer.
func FleetProjection(ctx context.Context, f *fabric.Fabric) (map[string]Projection, error) {
	events, err := f.ReplaySubject(ctx, fabric.FleetMintedSubject())
	if err != nil {
		return nil, err
	}
	// The fleet filter only matches canonical subjects, so SessionOf always
	// yields a real session token here.
	bySession := map[string][]fabric.Event{}
	for _, e := range events {
		s := fabric.SessionOf(e.Subject)
		bySession[s] = append(bySession[s], e)
	}
	fleet := make(map[string]Projection, len(bySession))
	for s, evs := range bySession {
		p, err := foldEvents(evs)
		if err != nil {
			return nil, err
		}
		fleet[s] = p
	}
	return fleet, nil
}

func foldEvents(events []fabric.Event) (Projection, error) {
	p := Projection{status: map[int]string{}}
	for _, e := range events {
		switch kind := e.Subject[strings.LastIndex(e.Subject, ".")+1:]; kind {
		case subjectKindCatch:
			r, err := DecodeCatch(e.Data)
			if err != nil {
				return Projection{}, err
			}
			p.catches = append(p.catches, r)
			if ShouldRecord(r.Outcome) {
				p.balance++
			}
		case kindSpend:
			s, err := DecodeSpend(e.Data)
			if err != nil {
				return Projection{}, err
			}
			if s.Amount > 0 {
				p.balance -= s.Amount
			}
		case kindWorkOrder:
			w, err := DecodeWorkOrder(e.Data)
			if err != nil {
				return Projection{}, err
			}
			p.orders = append(p.orders, w)
			p.status[w.ID] = w.Status
		case kindWOStatus:
			st, err := DecodeStatus(e.Data)
			if err != nil {
				return Projection{}, err
			}
			p.status[st.ID] = st.Status
		default:
			return Projection{}, fmt.Errorf("ledger: replay encountered unknown event kind %q", kind)
		}
	}
	return p, nil
}
