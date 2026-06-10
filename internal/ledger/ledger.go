// Package ledger is the append-only event log of confirmed catches (the data
// substrate under the trust ledger). It is DATA-ONLY: it captures
// at mint time the facts a catch can never be reconstructed from later (the
// survivor-set inventories, the self-flag and would-have-shipped bits, the
// reason), and stores NO weight or price — pricing is a separate, later concern.
package ledger

import (
	"context"
	"fmt"
	"sync"

	"github.com/joaomdsg/packets/internal/catch"
	"github.com/joaomdsg/packets/internal/fabric"
)

// CatchRecord is one confirmed-catch event, carrying the mint-time facts that
// cannot be recovered after the fact.
type CatchRecord struct {
	Outcome         catch.Outcome `json:"outcome"`
	Path            string        `json:"path"`
	Line            int           `json:"line"`
	BeforeRev       string        `json:"before_rev"`
	AfterRev        string        `json:"after_rev"`
	BeforeInventory []string      `json:"before_inventory"`
	AfterInventory  []string      `json:"after_inventory"`
	// MutantsConsidered is the size of the anchored line's operator inventory at
	// the after revision — the deduped operator alphabet that is the catch's
	// per-line denominator, NOT a whole-run mutant count.
	MutantsConsidered int    `json:"mutants_considered"`
	ReasonTag         string `json:"reason_tag"`
	SelfFlagged       bool   `json:"self_flagged"`
	WouldHaveShipped  bool   `json:"would_have_shipped"`
	// Producer names which producer minted this catch — the connect-cycle
	// ("connect") or a dispatched work-order ("wo:<id>"). It is NOT part of the
	// catch identity (a re-mint of the same identity is deduped regardless of
	// producer); it is provenance, so a catch from a dispatched run reads as
	// reinvestment, byte-distinguishable from a connect mint, and the field demuxes
	// the two producers on replay.
	Producer string `json:"producer,omitempty"`
}

// ShouldRecord reports whether an outcome warrants a ledger entry: only a real
// mint (Catch) is recorded, so no-op churn, no-catch, no-oracle-signal, and
// partial-catch leave no trace (the farm-denial invariant).
func ShouldRecord(o catch.Outcome) bool {
	return o == catch.Catch
}

// NewCatchRecord is the SINGLE construction site for a CatchRecord: every mint —
// the in-process review cycle and the sandboxed cage verifier alike — builds its
// record here, so there is exactly one place that decides a catch's recorded
// shape (the single-minter invariant). It returns nil when the outcome is not
// recordable (ShouldRecord), so callers can assign its result unconditionally.
// MutantsConsidered is the after-revision inventory size (the catch's per-line
// denominator) and ReasonTag is fixed to the catch tag. Producer provenance is
// stamped later, by whichever consumer appends the record.
func NewCatchRecord(outcome catch.Outcome, path string, line int, beforeRev, afterRev string, beforeInv, afterInv []string, selfFlagged, wouldHaveShipped bool) *CatchRecord {
	if !ShouldRecord(outcome) {
		return nil
	}
	return &CatchRecord{
		Outcome:           outcome,
		Path:              path,
		Line:              line,
		BeforeRev:         beforeRev,
		AfterRev:          afterRev,
		BeforeInventory:   beforeInv,
		AfterInventory:    afterInv,
		MutantsConsidered: len(afterInv),
		ReasonTag:         string(catch.Catch),
		SelfFlagged:       selfFlagged,
		WouldHaveShipped:  wouldHaveShipped,
	}
}

// kindSpend tags a debit line; kindWorkOrder tags a funded work-order line. A
// catch line carries NO kind field, so logs written before spends/work-orders
// existed re-read byte-identically.
const (
	kindSpend     = "spend"
	kindWorkOrder = "workorder"
	kindWOStatus  = "wostatus"
	kindWOVerdict = "woverdict"
)

// Target is the work a funded order will run: the rev/anchor triple a dispatched
// catch cycle executes. It is bound at funding time so the order is self-describing
// (the runner needs no other state) and so a dispatch can be refused when it would
// re-run the card's OWN already-caught cycle (a guaranteed loss).
type Target struct {
	BaseRev  string `json:"base_rev"`
	FixRev   string `json:"fix_rev"`
	TipRev   string `json:"tip_rev"`
	Path     string `json:"path"`
	Line     int    `json:"line"`
	LineHash string `json:"line_hash,omitempty"`
}

// DispatchCounts is the work-order tally split by current status — the watchable
// shape the Lead sees move queued→running→done as a dispatched order runs.
type DispatchCounts struct {
	Queued  int
	Running int
	Done    int
}

// inProcessProducer is the producer tag every work-order carries: the single
// in-process writer. Carrying it explicitly lets a future cross-process producer
// demux producers on replay without a schema migration.
const inProcessProducer = "in-process"

// WorkOrderRecord is the consequence a Spend funds: one unit of dispatched work,
// queued (it is not executed here). It shares the append-only stream and is
// distinguished by Kind=="workorder". It is paired
// with a debit (a spend line) in one atomic write, so a balance can never fund
// more orders than it held (conservation: debits == orders, per account).
type WorkOrderRecord struct {
	Kind     string `json:"kind"`
	ID       int    `json:"id"`
	Producer string `json:"producer"`
	Status   string `json:"status"`
	Reason   string `json:"reason,omitempty"`
	Target   Target `json:"target"`
}

// StatusRecord is one appended status transition for a work-order id. Status is
// NEVER mutated in place — each transition is a new line, so the log stays
// append-only and an order's current status replays as the last status line for
// its id (defaulting to the order's funded Status when none has been appended).
type StatusRecord struct {
	Kind   string `json:"kind"`
	ID     int    `json:"id"`
	Status string `json:"status"`
}

// WorkOrderVerdictRecord persists the oracle's honest verdict for one run of a
// work-order, keyed by id and distinguished by Kind=="woverdict". It is DIAGNOSTIC
// metadata — the WHY behind a caught/missed order (no-catch, no-oracle-signal,
// lost-via-rename, tested, …) — and is NEVER an economic event: it shares the
// append-only stream with the work-order/status lines but mints no balance and is
// not a confirmed catch (the two-scores invariant). An order's current verdict
// replays as the last verdict line for its id (last-writer-wins, like status).
type WorkOrderVerdictRecord struct {
	Kind    string `json:"kind"`
	ID      int    `json:"id"`
	Verdict string `json:"verdict"`
}

// SpendRecord is a debit against the confirmed-catch balance — the economy's
// SINK, the first non-minting record kind. It shares the append-only stream with
// CatchRecord and is distinguished by Kind=="spend". A spend can never mint
// credit: AppendSpend refuses any amount the current balance cannot cover.
type SpendRecord struct {
	Kind   string `json:"kind"`
	Amount int    `json:"amount"`
	Reason string `json:"reason,omitempty"`
}

// Log is one session's append-only economy log, backed by the JetStream fabric
// (the single authoritative substrate) and scoped to a session+instance subject
// namespace. Two Logs on the same fabric with distinct session tokens are
// ISOLATED economies: a Log only ever publishes and replays its own
// session.<session>.events.<instance> subtree, so a mint or spend on one session
// can never touch another — isolation enforced by the subject token.
//
// A Log serializes its writers under mu: Append, AppendSpend, and AppendDispatch
// take the lock across the replay-then-publish step, so the read-then-write
// balance/dedup check is atomic — no TOCTOU letting two spenders both see
// "enough" and overshoot below zero, and no two writers both seeing a catch
// "absent" and both minting it. The live server drives two writers at once (the
// catch cycle's Append and the Lead's AppendSpend on an action goroutine); the
// fabric publish waits for the stream ack before the lock is released, so the
// next writer replays the just-committed event. The projecting reads (Records,
// Balance, …) replay the committed stream and see whatever events were acked
// at replay time.
type Log struct {
	f          *fabric.Fabric
	session    string
	instance   string
	mu         sync.Mutex
	ownsFabric bool
}

// Bind binds a Log to an already-running fabric under the session+instance
// subject namespace. The Log does NOT own the fabric: its Close is a no-op, and
// the fabric's lifecycle belongs to whoever started it. Use this for sessions
// sharing one server's fabric.
//
// session and instance are subject tokens: they must be non-empty and contain no
// '.', space, or NATS wildcard ('*'/'>'), since they are interpolated into the
// dotted subject. A token with those characters would corrupt the subject's
// token structure. This is the caller's contract, not validated here.
func Bind(f *fabric.Fabric, session, instance string) *Log {
	return &Log{f: f, session: session, instance: instance}
}

// BindOwning is Bind for the Log that OWNS the fabric's lifecycle: its Close
// shuts the embedded server down. A server stands up one fabric and gives its
// primary Log ownership, so the existing "close the returned log" teardown
// contract tears the whole substrate down once, after the non-owning session
// Logs' no-op closes.
func BindOwning(f *fabric.Fabric, session, instance string) *Log {
	return &Log{f: f, session: session, instance: instance, ownsFabric: true}
}

// project folds this session's committed economy state from the stream — the one
// read path behind every projecting reader, so they all observe the same fold.
func (l *Log) project() (Projection, error) {
	return ReplayProjection(context.Background(), l.f, l.session, l.instance)
}

// Append writes r as exactly one JSON line. It refuses any record that is not a
// confirmed catch, so the log can hold nothing but real mints regardless of a
// miswired caller.
func (l *Log) Append(r CatchRecord) error {
	if !ShouldRecord(r.Outcome) {
		return fmt.Errorf("ledger: refusing to record a non-catch outcome %q", r.Outcome)
	}
	// Hold the lock across the dedup replay AND the publish so the check-then-write
	// is one atomic step (no two writers both seeing "absent" and both minting).
	// The publish waits for the stream ack, so the next writer's replay sees it.
	l.mu.Lock()
	defer l.mu.Unlock()
	p, err := l.project()
	if err != nil {
		return err
	}
	key := identityKey(r)
	for _, e := range p.Records() {
		if identityKey(e) == key {
			// A catch is identified by (BeforeRev, AfterRev, Path, Line, ReasonTag).
			// Re-running the same work reproduces the same identity; minting it twice
			// is the farm. Refuse the duplicate — projected purely from the committed
			// stream, so the gate survives a restart (a restart cannot reopen the farm).
			return fmt.Errorf("ledger: refusing a duplicate catch identity %q — a re-run mints nothing", key)
		}
	}
	if _, err := PublishCatch(context.Background(), l.f, l.session, l.instance, r); err != nil {
		return fmt.Errorf("ledger: append: %w", err)
	}
	return nil
}

// identityKey is a catch's identity: the tuple that makes two catches the SAME
// catch — the same anchored line, the same before→after revisions, the same
// reason. It is the dedup key the farm-denial gate (Append) keys on, and the
// provenance a re-run is measured against (re-run the SAME identity ⇒ no mint).
func identityKey(r CatchRecord) string {
	return fmt.Sprintf("%s\x00%s\x00%s\x00%d\x00%s", r.BeforeRev, r.AfterRev, r.Path, r.Line, r.ReasonTag)
}

// Records reads back every appended CATCH record in order. Spend (debit) lines
// are skipped, so the confirmed-catch count is never polluted by the sink.
func (l *Log) Records() ([]CatchRecord, error) {
	p, err := l.project()
	if err != nil {
		return nil, err
	}
	return p.Records(), nil
}

// Balance is the economy's held quantity: confirmed catches (credits) minus the
// sum of spends (debits), projected purely from the log — no in-memory counter,
// so it replays identically from the persisted stream alone.
func (l *Log) Balance() (int, error) {
	p, err := l.project()
	if err != nil {
		return 0, err
	}
	return p.Balance(), nil
}

// AppendSpend records a debit of amount against the balance — the sink that lets
// the stock drain. It refuses a non-positive amount and any amount the current
// balance cannot cover (you cannot spend what you did not catch), writing NOTHING
// on refusal. It does NOT route through Append: the catch-only farm-denial gate
// stays intact and debits travel this guarded path alone.
func (l *Log) AppendSpend(amount int, reason string) error {
	if amount <= 0 {
		return fmt.Errorf("ledger: spend amount must be positive, got %d", amount)
	}
	// Hold the write lock across the balance check AND the write: the read and
	// the debit must be one atomic step, or two concurrent spenders both read
	// "enough" before either writes and the balance overshoots below zero.
	l.mu.Lock()
	defer l.mu.Unlock()
	p, err := l.project()
	if err != nil {
		return err
	}
	if amount > p.Balance() {
		return fmt.Errorf("ledger: spend of %d exceeds balance %d", amount, p.Balance())
	}
	if _, err := PublishSpend(context.Background(), l.f, l.session, l.instance, SpendRecord{Kind: kindSpend, Amount: amount, Reason: reason}); err != nil {
		return fmt.Errorf("ledger: append spend: %w", err)
	}
	return nil
}

// AppendDispatch funds exactly one work-order against the balance — the
// consequence a Spend buys. It refuses if the balance cannot cover one unit
// (you cannot dispatch what you did not catch), writing NOTHING on refusal.
// On success it writes the debit (a spend of 1) AND the paired work-order line
// as a SINGLE write under the one lock, so the two lines never tear apart and a
// balance can never fund more orders than it held: one debit ⇒ one order,
// conserved. The work-order id is monotonic, derived from the persisted log
// (count of existing work-orders + 1) so it survives a reopen with no in-memory
// counter. The order is queued — it is not executed here.
//
// target is the distinct work the order will run; own is the card's OWN caught
// cycle. A dispatch whose target equals own is refused (writing nothing): it
// would re-run already-caught work, reproducing an identity the dedup gate would
// mint nothing for — a guaranteed loss, so it is rejected up front (the
// distinct-work requirement; the identity dedup in Append is the backstop).
func (l *Log) AppendDispatch(reason string, target, own Target) error {
	if target == own {
		return fmt.Errorf("ledger: refusing to dispatch the card's own caught work — fund DISTINCT work")
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	p, err := l.project()
	if err != nil {
		return err
	}
	if p.Balance() < 1 {
		return fmt.Errorf("ledger: cannot dispatch with balance %d — nothing to fund", p.Balance())
	}
	ctx := context.Background()
	// The debit and its work-order are two publishes under the one lock: no other
	// writer interleaves, so a balance can never fund more orders than it held (one
	// debit ⇒ one order, conserved at rest). They are NOT crash-atomic the way the
	// single file write was — a crash between the two would drop the order, never
	// double-mint — which is acceptable until the durability/hibernation gate is
	// built. The id is monotonic, derived from the committed order count + 1, so it
	// survives a restart with no in-memory counter.
	if _, err := PublishSpend(ctx, l.f, l.session, l.instance, SpendRecord{Kind: kindSpend, Amount: 1, Reason: reason}); err != nil {
		return fmt.Errorf("ledger: append dispatch debit: %w", err)
	}
	if _, err := PublishWorkOrder(ctx, l.f, l.session, l.instance, WorkOrderRecord{
		Kind:     kindWorkOrder,
		ID:       len(p.WorkOrders()) + 1,
		Producer: inProcessProducer,
		Status:   "queued",
		Reason:   reason,
		Target:   target,
	}); err != nil {
		return fmt.Errorf("ledger: append work-order: %w", err)
	}
	return nil
}

// DispatchView is one funded work-order's round-trip, made legible: its id, the
// target it runs, its current status (queued→running→done), whether its run minted
// a catch (Caught) or not (a missed bet), and the oracle's honest Verdict for that
// run (the WHY: no-catch, no-oracle-signal, lost-via-rename, tested, … — empty when
// none persisted). Honest per-order outcome — never a fabricated rank. Caught keys
// on the order's own "wo:<id>" mint provenance, so an unrelated connect-cycle catch
// never falsely credits it.
type DispatchView struct {
	ID      int
	Target  Target
	Status  string
	Caught  bool
	Verdict string
	// Questions is how many open review questions (surviving mutants) the filled
	// order left — the order's reviewable test-debt. NOT projected from the ledger
	// (the findings are off-ledger diagnostic state); the app layer fills it from the
	// per-order findings cache before rendering. Zero when none / not yet filled.
	Questions int
}

// RecentDispatches projects this log's funded work-orders into the most-recent n
// DispatchViews, NEWEST FIRST (n<=0 returns all) — the data behind the board's
// "watch a funded order round-trip" surface. A pure projection of the persisted
// log: status is the order's last status line (default "queued"), Caught is
// whether a catch with Producer "wo:<id>" was minted.
func (l *Log) RecentDispatches(n int) ([]DispatchView, error) {
	p, err := l.project()
	if err != nil {
		return nil, err
	}
	return p.RecentDispatches(n), nil
}

// WorkOrders reads back every funded work-order in order, a pure projection of
// the persisted log (catch and spend lines are skipped). The monotonic id and
// producer/status fields are read straight from the stream, so they replay identically.
func (l *Log) WorkOrders() ([]WorkOrderRecord, error) {
	p, err := l.project()
	if err != nil {
		return nil, err
	}
	return p.WorkOrders(), nil
}

// PendingDispatches counts the funded work-orders projected purely from the log
// — the total dispatched-work tally (every funded order, regardless of status).
func (l *Log) PendingDispatches() (int, error) {
	orders, err := l.WorkOrders()
	if err != nil {
		return 0, err
	}
	return len(orders), nil
}

// AppendStatus records a work-order's status transition as a NEW append-only line
// keyed by id — never mutating the order, so the log stays a pure append-only
// substrate and an order's current status replays as its last status line.
func (l *Log) AppendStatus(id int, status string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if _, err := PublishStatus(context.Background(), l.f, l.session, l.instance, StatusRecord{Kind: kindWOStatus, ID: id, Status: status}); err != nil {
		return fmt.Errorf("ledger: append status: %w", err)
	}
	return nil
}

// AppendWorkOrderVerdict records the oracle's verdict for one run of a work-order
// as a NEW append-only line keyed by id — never mutating the order or its status,
// so the log stays a pure append-only substrate and an order's current verdict
// replays as its last verdict line. Diagnostic only: it mints no balance and is not
// a confirmed catch.
func (l *Log) AppendWorkOrderVerdict(id int, verdict string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if _, err := PublishWorkOrderVerdict(context.Background(), l.f, l.session, l.instance, WorkOrderVerdictRecord{Kind: kindWOVerdict, ID: id, Verdict: verdict}); err != nil {
		return fmt.Errorf("ledger: append work-order verdict: %w", err)
	}
	return nil
}

// DispatchStatusCounts is the work-order tally split by CURRENT status — the
// watchable shape the Lead sees move queued→running→done. Each order starts at
// its funded Status ("queued") and advances to the last status line appended for
// its id (last-writer-wins per id), so every order is counted in exactly one
// bucket. A pure projection of the persisted log.
func (l *Log) DispatchStatusCounts() (DispatchCounts, error) {
	p, err := l.project()
	if err != nil {
		return DispatchCounts{}, err
	}
	return p.DispatchStatusCounts(), nil
}

// QueuedWorkOrders returns the funded orders whose CURRENT status is queued, in
// funding (id) order — the runner's input: the work waiting to be executed.
func (l *Log) QueuedWorkOrders() ([]WorkOrderRecord, error) {
	p, err := l.project()
	if err != nil {
		return nil, err
	}
	return p.QueuedWorkOrders(), nil
}

// Close releases the Log. A Log bound with Bind does not own the fabric, so its
// Close is a no-op; a Log bound with BindOwning owns the embedded server and
// shuts it down. It takes the write lock so it cannot tear the fabric down out
// from under an in-flight writer.
func (l *Log) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.ownsFabric {
		return l.f.Close()
	}
	return nil
}
