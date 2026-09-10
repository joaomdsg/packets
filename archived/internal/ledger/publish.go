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

// PublishPacket emits a funded packet record on the canonical
// minted-workorder subject and returns its stream sequence.
func PublishPacket(ctx context.Context, f *fabric.Fabric, session, instance string, w PacketRecord) (uint64, error) {
	return publish(ctx, f, session, instance, kindPacket, w)
}

// PublishStatus emits a packet status transition on the canonical
// minted-wostatus subject and returns its stream sequence.
func PublishStatus(ctx context.Context, f *fabric.Fabric, session, instance string, s StatusRecord) (uint64, error) {
	return publish(ctx, f, session, instance, kindPacketStatus, s)
}

// PublishPacketVerdict emits a per-packet oracle-verdict record on the canonical
// minted-woverdict subject and returns its stream sequence. It targets StatusMinted
// like the packet/status lines (the send subtree), NOT a catch — it is
// diagnostic metadata, never an economic event.
func PublishPacketVerdict(ctx context.Context, f *fabric.Fabric, session, instance string, v PacketVerdictRecord) (uint64, error) {
	return publish(ctx, f, session, instance, kindPacketVerdict, v)
}

// PublishRefine emits a refined-packet record on the canonical minted-worefine
// subject and returns its stream sequence. Like the status/verdict lines it targets
// the send subtree, never a catch — sharpening is not an economic event.
func PublishRefine(ctx context.Context, f *fabric.Fabric, session, instance string, r RefinedPacketRecord) (uint64, error) {
	return publish(ctx, f, session, instance, kindPacketRefine, r)
}

// PublishAnnotation emits a durable annotation (or reply) on the canonical
// minted subtree, returning its stream sequence. Like the refine/status lines it
// is not an economic event — it never touches a catch subject.
func PublishAnnotation(ctx context.Context, f *fabric.Fabric, session, instance string, r AnnotationRecord) (uint64, error) {
	return publish(ctx, f, session, instance, kindAnnotation, r)
}

// DecodeAnnotation reverses PublishAnnotation for the fold.
func DecodeAnnotation(data []byte) (AnnotationRecord, error) {
	var r AnnotationRecord
	if err := json.Unmarshal(data, &r); err != nil {
		return AnnotationRecord{}, fmt.Errorf("ledger: decode annotation: %v", err)
	}
	return r, nil
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

// DecodePacket decodes a funded packet event payload from the bus.
func DecodePacket(data []byte) (PacketRecord, error) {
	var w PacketRecord
	if err := json.Unmarshal(data, &w); err != nil {
		return PacketRecord{}, fmt.Errorf("ledger: decode packet: %v", err)
	}
	return w, nil
}

// DecodeStatus decodes a packet status-transition event payload from the bus.
func DecodeStatus(data []byte) (StatusRecord, error) {
	var s StatusRecord
	if err := json.Unmarshal(data, &s); err != nil {
		return StatusRecord{}, fmt.Errorf("ledger: decode status: %v", err)
	}
	return s, nil
}

// DecodePacketVerdict decodes a per-packet oracle-verdict event payload from the bus.
func DecodePacketVerdict(data []byte) (PacketVerdictRecord, error) {
	var v PacketVerdictRecord
	if err := json.Unmarshal(data, &v); err != nil {
		return PacketVerdictRecord{}, fmt.Errorf("ledger: decode packet verdict: %v", err)
	}
	return v, nil
}

// DecodeRefine decodes a refined-packet event payload from the bus.
func DecodeRefine(data []byte) (RefinedPacketRecord, error) {
	var r RefinedPacketRecord
	if err := json.Unmarshal(data, &r); err != nil {
		return RefinedPacketRecord{}, fmt.Errorf("ledger: decode refine: %v", err)
	}
	return r, nil
}
