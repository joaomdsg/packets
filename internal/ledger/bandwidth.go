package ledger

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/joaomdsg/packets/internal/fabric"
)

// kindBlock tags a "the Lead's input is needed" event; kindUnblock tags the
// matching "the Lead provided it" event. Together they are the attention economy's
// source: a cleared block (a block with a matching unblock) earns bandwidth, the
// scarce resource that funds dispatching autonomous work.
const (
	kindBlock        = "block"
	kindUnblock      = "unblock"
	kindBWSpend      = "bwspend"
	liveDispatchCost = 1 // bandwidth a UI-authored live order costs to dispatch
)

// BandwidthSpendRecord is a debit against the earned attention bandwidth — the
// meter's SINK, mirroring SpendRecord for the catch balance. A live order authored
// from the UI is funded here, not from a catch, so the two meters stay distinct.
type BandwidthSpendRecord struct {
	Kind   string `json:"kind"`
	Amount int    `json:"amount"`
	Reason string `json:"reason,omitempty"`
}

// BlockRecord marks the moment a producer needed the Lead's input (a raised
// question, an order awaiting review) for the work identified by ID, stamped with
// the wall-clock time so the matching unblock's latency is a logged fact, never an
// inference.
type BlockRecord struct {
	Kind     string `json:"kind"`
	ID       string `json:"id"`
	AtUnixMs int64  `json:"at_unix_ms"`
}

// UnblockRecord marks the moment the Lead cleared the block identified by ID
// (answered the question, reviewed the order). Its latency from the block is the
// awarded bandwidth's grounding.
type UnblockRecord struct {
	Kind     string `json:"kind"`
	ID       string `json:"id"`
	AtUnixMs int64  `json:"at_unix_ms"`
}

// Bandwidth award tiers. The award redeems against ONE logged block→unblock pair
// and folds both axes: a throughput base (you cleared a block at all) plus a
// latency bonus (how fast). A faster clear is worth more attention bandwidth.
const (
	bandwidthFastWindow = 2 * time.Minute  // a clear within this earns the full bonus
	bandwidthMedWindow  = 15 * time.Minute // within this earns the partial bonus
	bandwidthBase       = 1                // the throughput base: any clear earns this
	bandwidthFastBonus  = 2                // a fast clear earns base + this
	bandwidthMedBonus   = 1                // a medium clear earns base + this
)

// bandwidthAward is the credit one cleared block pays, given its clear latency. A
// negative latency (clock skew between the two stamps) floors at the slow tier, so
// a skewed pair can never pay more than the throughput base.
func bandwidthAward(latency time.Duration) int {
	switch {
	case latency < 0:
		return bandwidthBase // a skewed (negative) interval never pays more than the base
	case latency <= bandwidthFastWindow:
		return bandwidthBase + bandwidthFastBonus
	case latency <= bandwidthMedWindow:
		return bandwidthBase + bandwidthMedBonus
	default:
		return bandwidthBase
	}
}

// PublishBlock emits a block event on the canonical minted subtree.
func PublishBlock(ctx context.Context, f *fabric.Fabric, session, instance string, b BlockRecord) (uint64, error) {
	return publish(ctx, f, session, instance, kindBlock, b)
}

// PublishUnblock emits an unblock event on the canonical minted subtree.
func PublishUnblock(ctx context.Context, f *fabric.Fabric, session, instance string, u UnblockRecord) (uint64, error) {
	return publish(ctx, f, session, instance, kindUnblock, u)
}

// PublishBandwidthSpend emits a bandwidth debit on the canonical minted subtree.
func PublishBandwidthSpend(ctx context.Context, f *fabric.Fabric, session, instance string, s BandwidthSpendRecord) (uint64, error) {
	return publish(ctx, f, session, instance, kindBWSpend, s)
}

// DecodeBandwidthSpend decodes a bandwidth debit event payload from the bus.
func DecodeBandwidthSpend(data []byte) (BandwidthSpendRecord, error) {
	var s BandwidthSpendRecord
	if err := json.Unmarshal(data, &s); err != nil {
		return BandwidthSpendRecord{}, fmt.Errorf("ledger: decode bandwidth spend: %v", err)
	}
	return s, nil
}

// DecodeBlock decodes a block event payload from the bus.
func DecodeBlock(data []byte) (BlockRecord, error) {
	var b BlockRecord
	if err := json.Unmarshal(data, &b); err != nil {
		return BlockRecord{}, fmt.Errorf("ledger: decode block: %v", err)
	}
	return b, nil
}

// DecodeUnblock decodes an unblock event payload from the bus.
func DecodeUnblock(data []byte) (UnblockRecord, error) {
	var u UnblockRecord
	if err := json.Unmarshal(data, &u); err != nil {
		return UnblockRecord{}, fmt.Errorf("ledger: decode unblock: %v", err)
	}
	return u, nil
}

// AppendBlock records that the work identified by id needs the Lead's input, at
// time at — the start of an attention-latency interval the matching AppendUnblock
// closes. Unlike a spend it gates on nothing: a block is a fact, not a debit.
func (l *Log) AppendBlock(id string, at time.Time) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	_, err := PublishBlock(context.Background(), l.f, l.session, l.instance,
		BlockRecord{Kind: kindBlock, ID: id, AtUnixMs: at.UnixMilli()})
	if err != nil {
		return fmt.Errorf("ledger: append block: %w", err)
	}
	return nil
}

// AppendUnblock records that the Lead cleared id at time at. The award is computed
// at projection time from the logged block/unblock latency, so this only logs the
// fact — it mints no bandwidth itself.
func (l *Log) AppendUnblock(id string, at time.Time) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	_, err := PublishUnblock(context.Background(), l.f, l.session, l.instance,
		UnblockRecord{Kind: kindUnblock, ID: id, AtUnixMs: at.UnixMilli()})
	if err != nil {
		return fmt.Errorf("ledger: append unblock: %w", err)
	}
	return nil
}

// AppendBandwidthSpend debits amount from the earned bandwidth. It refuses any
// amount the current bandwidth cannot cover (no overdraft, mirroring AppendSpend),
// under the lock so the read-then-write is atomic.
func (l *Log) AppendBandwidthSpend(amount int, reason string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	p, err := l.project()
	if err != nil {
		return err
	}
	if amount > 0 && p.Bandwidth() < amount {
		return fmt.Errorf("ledger: cannot spend %d bandwidth — only %d earned", amount, p.Bandwidth())
	}
	if _, err := PublishBandwidthSpend(context.Background(), l.f, l.session, l.instance,
		BandwidthSpendRecord{Kind: kindBWSpend, Amount: amount, Reason: reason}); err != nil {
		return fmt.Errorf("ledger: append bandwidth spend: %w", err)
	}
	return nil
}

// AppendLiveDispatch funds a UI-authored LIVE order from the attention-bandwidth
// meter (not a catch) and queues it in one write under the lock: the bandwidth
// debit and the work-order are two publishes no other writer interleaves, so the
// meter can never fund more orders than it held. It refuses its own target (the
// distinct-work guard) and any dispatch the bandwidth cannot cover.
func (l *Log) AppendLiveDispatch(reason string, target, own Target) error {
	if target == own {
		return fmt.Errorf("ledger: refusing to dispatch the card's own work — fund DISTINCT work")
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	p, err := l.project()
	if err != nil {
		return err
	}
	if p.Bandwidth() < liveDispatchCost {
		return fmt.Errorf("ledger: cannot dispatch — only %d attention bandwidth, need %d", p.Bandwidth(), liveDispatchCost)
	}
	ctx := context.Background()
	if _, err := PublishBandwidthSpend(ctx, l.f, l.session, l.instance,
		BandwidthSpendRecord{Kind: kindBWSpend, Amount: liveDispatchCost, Reason: reason}); err != nil {
		return fmt.Errorf("ledger: append live dispatch debit: %w", err)
	}
	if _, err := PublishWorkOrder(ctx, l.f, l.session, l.instance, WorkOrderRecord{
		Kind:     kindWorkOrder,
		ID:       len(p.WorkOrders()) + 1,
		Producer: inProcessProducer,
		Status:   "queued",
		Reason:   reason,
		Target:   target,
	}); err != nil {
		return fmt.Errorf("ledger: append live work-order: %w", err)
	}
	return nil
}

// Bandwidth folds the session's earned attention bandwidth — the sum of awards
// across every cleared block (a block with a matching unblock). An open block
// (no unblock) earns nothing.
func (l *Log) Bandwidth() (int, error) {
	p, err := l.project()
	if err != nil {
		return 0, err
	}
	return p.Bandwidth(), nil
}
