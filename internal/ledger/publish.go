package ledger

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/joaomdsg/packets/internal/fabric"
)

// subjectKindCatch is the subject-taxonomy token for a confirmed-catch event. A
// catch line carries no "kind" field in the JSONL (so legacy logs re-read
// identically), so its bus token is named here rather than reused from a record
// constant. The other kinds reuse the JSONL discriminators (kindSpend etc.), so
// a payload's subject token and its on-disk kind agree.
const subjectKindCatch = "catch"

// PublishCatch emits a confirmed-catch record on the canonical minted-catch
// subject for session+instance and returns its stream sequence. It is a thin
// substrate primitive — the catch-only farm-denial gate lives in Log.Append, not
// here — so a caller drives it only with records that already passed that gate.
//
// session and instance are host-minted subject tokens: non-empty and free of
// '.', space, or NATS wildcard, since they are interpolated into the dotted
// subject. The caller owns that contract; it is not validated here.
func PublishCatch(ctx context.Context, f *fabric.Fabric, session, instance string, r CatchRecord) (uint64, error) {
	return publish(ctx, f, session, instance, subjectKindCatch, r)
}

// PublishSpend emits a debit record on the canonical minted-spend subject and
// returns its stream sequence.
func PublishSpend(ctx context.Context, f *fabric.Fabric, session, instance string, s SpendRecord) (uint64, error) {
	return publish(ctx, f, session, instance, kindSpend, s)
}

// PublishWorkOrder emits a funded work-order record on the canonical
// minted-workorder subject and returns its stream sequence.
func PublishWorkOrder(ctx context.Context, f *fabric.Fabric, session, instance string, w WorkOrderRecord) (uint64, error) {
	return publish(ctx, f, session, instance, kindWorkOrder, w)
}

// PublishStatus emits a work-order status transition on the canonical
// minted-wostatus subject and returns its stream sequence.
func PublishStatus(ctx context.Context, f *fabric.Fabric, session, instance string, s StatusRecord) (uint64, error) {
	return publish(ctx, f, session, instance, kindWOStatus, s)
}

// PublishWorkOrderVerdict emits a per-order oracle-verdict record on the canonical
// minted-woverdict subject and returns its stream sequence. It targets StatusMinted
// like the work-order/status lines (the dispatch subtree), NOT a catch — it is
// diagnostic metadata, never an economic event.
func PublishWorkOrderVerdict(ctx context.Context, f *fabric.Fabric, session, instance string, v WorkOrderVerdictRecord) (uint64, error) {
	return publish(ctx, f, session, instance, kindWOVerdict, v)
}

func publish(ctx context.Context, f *fabric.Fabric, session, instance, kind string, rec any) (uint64, error) {
	data, err := json.Marshal(rec)
	if err != nil {
		return 0, fmt.Errorf("ledger: encode %s: %v", kind, err)
	}
	return f.Publish(ctx, fabric.EventSubject(session, instance, fabric.StatusMinted, kind), data)
}

// DecodeCatch decodes a confirmed-catch event payload from the bus.
func DecodeCatch(data []byte) (CatchRecord, error) {
	var r CatchRecord
	if err := json.Unmarshal(data, &r); err != nil {
		return CatchRecord{}, fmt.Errorf("ledger: decode catch: %v", err)
	}
	return r, nil
}

// DecodeSpend decodes a debit event payload from the bus.
func DecodeSpend(data []byte) (SpendRecord, error) {
	var s SpendRecord
	if err := json.Unmarshal(data, &s); err != nil {
		return SpendRecord{}, fmt.Errorf("ledger: decode spend: %v", err)
	}
	return s, nil
}

// DecodeWorkOrder decodes a funded work-order event payload from the bus.
func DecodeWorkOrder(data []byte) (WorkOrderRecord, error) {
	var w WorkOrderRecord
	if err := json.Unmarshal(data, &w); err != nil {
		return WorkOrderRecord{}, fmt.Errorf("ledger: decode work-order: %v", err)
	}
	return w, nil
}

// DecodeStatus decodes a work-order status-transition event payload from the bus.
func DecodeStatus(data []byte) (StatusRecord, error) {
	var s StatusRecord
	if err := json.Unmarshal(data, &s); err != nil {
		return StatusRecord{}, fmt.Errorf("ledger: decode status: %v", err)
	}
	return s, nil
}

// DecodeWorkOrderVerdict decodes a per-order oracle-verdict event payload from the bus.
func DecodeWorkOrderVerdict(data []byte) (WorkOrderVerdictRecord, error) {
	var v WorkOrderVerdictRecord
	if err := json.Unmarshal(data, &v); err != nil {
		return WorkOrderVerdictRecord{}, fmt.Errorf("ledger: decode work-order verdict: %v", err)
	}
	return v, nil
}
