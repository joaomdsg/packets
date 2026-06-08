// Package ledger is the append-only event log of confirmed catches (the data
// substrate under DESIGN-COUNCIL's Trust Ledger). It is DATA-ONLY: it captures
// at mint time the facts a catch can never be reconstructed from later (the
// survivor-set inventories, the self-flag and would-have-shipped bits, the
// reason), and stores NO weight or price — pricing is a separate, later concern.
package ledger

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/joaomdsg/packets/internal/catch"
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
	// the two real producers on replay (the DESIGN §13.3 P0, now load-bearing).
	Producer string `json:"producer,omitempty"`
}

// ShouldRecord reports whether an outcome warrants a ledger entry: only a real
// mint (Catch) is recorded, so no-op churn, no-catch, no-oracle-signal, and
// partial-catch leave no trace (the farm-denial invariant).
func ShouldRecord(o catch.Outcome) bool {
	return o == catch.Catch
}

// kindSpend tags a debit line; kindWorkOrder tags a funded work-order line. A
// catch line carries NO kind field, so logs written before spends/work-orders
// existed re-read byte-identically.
const (
	kindSpend     = "spend"
	kindWorkOrder = "workorder"
	kindWOStatus  = "wostatus"
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

// inProcessProducer is the producer tag every work-order carries this round —
// the single in-process writer. It is pre-paid onto the line now (DESIGN §13.3
// P0): once a real cross-process producer exists, the field is already there to
// demux producers on replay, and the monotonic seq reconciliation can be added
// without a schema migration.
const inProcessProducer = "in-process"

// WorkOrderRecord is the consequence a Spend funds: one unit of dispatched work,
// queued (this round it does NOT run — executing it is a later slice). It shares
// the append-only JSONL and is distinguished by Kind=="workorder". It is paired
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

// SpendRecord is a debit against the confirmed-catch balance — the economy's
// SINK, the first non-minting record kind. It shares the append-only JSONL with
// CatchRecord and is distinguished by Kind=="spend". A spend can never mint
// credit: AppendSpend refuses any amount the current balance cannot cover.
type SpendRecord struct {
	Kind   string `json:"kind"`
	Amount int    `json:"amount"`
	Reason string `json:"reason,omitempty"`
}

// Log is an append-only JSONL log of CatchRecords backed by a file.
//
// A Log serializes all mutation paths under mu: Append, AppendSpend, and Close
// take the write lock, so concurrent writers never tear a line, and
// AppendSpend's read-then-write balance check is atomic (no TOCTOU letting two
// spenders both see "enough" and overshoot below zero). This matters because
// the live server now drives two writers at once — the catch cycle's Append
// (a mint) and the Lead's AppendSpend (a debit, on an action goroutine). The
// projecting reads (Records, Balance) open their own read handle via scan and
// see whatever full lines were committed at scan time.
type Log struct {
	path string
	mu   sync.Mutex
	f    *os.File
}

// Open opens (creating if needed) the append-only log at path.
func Open(path string) (*Log, error) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("ledger: open %s: %w", path, err)
	}
	return &Log{path: path, f: f}, nil
}

// Append writes r as exactly one JSON line. It refuses any record that is not a
// confirmed catch, so the log can hold nothing but real mints regardless of a
// miswired caller.
func (l *Log) Append(r CatchRecord) error {
	if !ShouldRecord(r.Outcome) {
		return fmt.Errorf("ledger: refusing to record a non-catch outcome %q", r.Outcome)
	}
	line, err := json.Marshal(r)
	if err != nil {
		return fmt.Errorf("ledger: marshal record: %w", err)
	}
	// Hold the lock across the dedup scan AND the write so the check-then-write is
	// one atomic step (no two writers both seeing "absent" and both minting).
	l.mu.Lock()
	defer l.mu.Unlock()
	existing, err := l.Records()
	if err != nil {
		return err
	}
	key := identityKey(r)
	for _, e := range existing {
		if identityKey(e) == key {
			// A catch is identified by (BeforeRev, AfterRev, Path, Line, ReasonTag).
			// Re-running the same work reproduces the same identity; minting it twice
			// is the farm. Refuse the duplicate — projected purely from the persisted
			// log, so the gate survives a reopen (a restart cannot reopen the farm).
			return fmt.Errorf("ledger: refusing a duplicate catch identity %q — a re-run mints nothing", key)
		}
	}
	if _, err := l.f.Write(append(line, '\n')); err != nil {
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
	var out []CatchRecord
	err := l.scan(func(kind string, line []byte) error {
		if kind == kindSpend || kind == kindWorkOrder || kind == kindWOStatus {
			return nil // a debit, a work-order, or a status transition is not a catch
		}
		var r CatchRecord
		if err := json.Unmarshal(line, &r); err != nil {
			return fmt.Errorf("ledger: decode record: %w", err)
		}
		out = append(out, r)
		return nil
	})
	return out, err
}

// Balance is the economy's held quantity: confirmed catches (credits) minus the
// sum of spends (debits), projected purely from the log — no in-memory counter,
// so it replays identically from the persisted JSONL alone.
func (l *Log) Balance() (int, error) {
	balance := 0
	err := l.scan(func(kind string, line []byte) error {
		if kind == kindWorkOrder || kind == kindWOStatus {
			return nil // a work-order / its status transition is not a credit; the paired debit drains the balance
		}
		if kind == kindSpend {
			var s SpendRecord
			if err := json.Unmarshal(line, &s); err != nil {
				return fmt.Errorf("ledger: decode spend: %w", err)
			}
			// A spend can never mint credit: AppendSpend rejects amount<=0, but the
			// JSONL is the authoritative replay substrate, so a hand-edited
			// non-positive amount must contribute nothing rather than ADD to balance.
			if s.Amount > 0 {
				balance -= s.Amount
			}
			return nil
		}
		var r CatchRecord
		if err := json.Unmarshal(line, &r); err != nil {
			return fmt.Errorf("ledger: decode record: %w", err)
		}
		if ShouldRecord(r.Outcome) {
			balance++
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return balance, nil
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
	balance, err := l.Balance()
	if err != nil {
		return err
	}
	if amount > balance {
		return fmt.Errorf("ledger: spend of %d exceeds balance %d", amount, balance)
	}
	line, err := json.Marshal(SpendRecord{Kind: kindSpend, Amount: amount, Reason: reason})
	if err != nil {
		return fmt.Errorf("ledger: marshal spend: %w", err)
	}
	if _, err := l.f.Write(append(line, '\n')); err != nil {
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
// counter. The order is queued — this round it does not run.
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
	balance, err := l.Balance()
	if err != nil {
		return err
	}
	if balance < 1 {
		return fmt.Errorf("ledger: cannot dispatch with balance %d — nothing to fund", balance)
	}
	orders, err := l.WorkOrders()
	if err != nil {
		return err
	}
	spend, err := json.Marshal(SpendRecord{Kind: kindSpend, Amount: 1, Reason: reason})
	if err != nil {
		return fmt.Errorf("ledger: marshal dispatch debit: %w", err)
	}
	order, err := json.Marshal(WorkOrderRecord{
		Kind:     kindWorkOrder,
		ID:       len(orders) + 1,
		Producer: inProcessProducer,
		Status:   "queued",
		Reason:   reason,
		Target:   target,
	})
	if err != nil {
		return fmt.Errorf("ledger: marshal work-order: %w", err)
	}
	// One Write call for both lines: the debit and its work-order commit together
	// or not at all — they can never half-land.
	buf := append(spend, '\n')
	buf = append(buf, order...)
	buf = append(buf, '\n')
	if _, err := l.f.Write(buf); err != nil {
		return fmt.Errorf("ledger: append dispatch: %w", err)
	}
	return nil
}

// WorkOrders reads back every funded work-order in order, a pure projection of
// the persisted log (catch and spend lines are skipped). The monotonic id and
// producer/status fields are read straight from disk, so they replay identically.
func (l *Log) WorkOrders() ([]WorkOrderRecord, error) {
	var out []WorkOrderRecord
	err := l.scan(func(kind string, line []byte) error {
		if kind != kindWorkOrder {
			return nil
		}
		var w WorkOrderRecord
		if err := json.Unmarshal(line, &w); err != nil {
			return fmt.Errorf("ledger: decode work-order: %w", err)
		}
		out = append(out, w)
		return nil
	})
	return out, err
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
	line, err := json.Marshal(StatusRecord{Kind: kindWOStatus, ID: id, Status: status})
	if err != nil {
		return fmt.Errorf("ledger: marshal status: %w", err)
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if _, err := l.f.Write(append(line, '\n')); err != nil {
		return fmt.Errorf("ledger: append status: %w", err)
	}
	return nil
}

// DispatchStatusCounts is the work-order tally split by CURRENT status — the
// watchable shape the Lead sees move queued→running→done. Each order starts at
// its funded Status ("queued") and advances to the last status line appended for
// its id (last-writer-wins per id), so every order is counted in exactly one
// bucket. A pure projection of the persisted log.
func (l *Log) DispatchStatusCounts() (DispatchCounts, error) {
	orders, status, err := l.ordersWithStatus()
	if err != nil {
		return DispatchCounts{}, err
	}
	var c DispatchCounts
	for _, o := range orders {
		switch status[o.ID] {
		case "running":
			c.Running++
		case "done":
			c.Done++
		default:
			c.Queued++
		}
	}
	return c, nil
}

// QueuedWorkOrders returns the funded orders whose CURRENT status is queued, in
// funding (id) order — the runner's input: the work waiting to be executed.
func (l *Log) QueuedWorkOrders() ([]WorkOrderRecord, error) {
	orders, status, err := l.ordersWithStatus()
	if err != nil {
		return nil, err
	}
	var queued []WorkOrderRecord
	for _, o := range orders {
		if status[o.ID] == "queued" {
			queued = append(queued, o)
		}
	}
	return queued, nil
}

// ordersWithStatus projects the funded orders (in id order) plus each order's
// CURRENT status (its funded Status, advanced by the last status line for its id).
// The shared read behind DispatchStatusCounts and QueuedWorkOrders.
func (l *Log) ordersWithStatus() ([]WorkOrderRecord, map[int]string, error) {
	var orders []WorkOrderRecord
	status := map[int]string{}
	err := l.scan(func(kind string, line []byte) error {
		switch kind {
		case kindWorkOrder:
			var w WorkOrderRecord
			if err := json.Unmarshal(line, &w); err != nil {
				return fmt.Errorf("ledger: decode work-order: %w", err)
			}
			orders = append(orders, w)
			status[w.ID] = w.Status
		case kindWOStatus:
			var s StatusRecord
			if err := json.Unmarshal(line, &s); err != nil {
				return fmt.Errorf("ledger: decode status: %w", err)
			}
			status[s.ID] = s.Status // last status line for an id wins
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return orders, status, nil
}

// scan reads each non-empty JSONL line in order, probing its "kind" discriminator
// (absent → a catch) and handing the raw line to fn. One place owns the file read.
func (l *Log) scan(fn func(kind string, line []byte) error) error {
	f, err := os.Open(l.path)
	if err != nil {
		return fmt.Errorf("ledger: read %s: %w", l.path, err)
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if len(sc.Bytes()) == 0 {
			continue
		}
		var probe struct {
			Kind string `json:"kind"`
		}
		if err := json.Unmarshal(sc.Bytes(), &probe); err != nil {
			return fmt.Errorf("ledger: decode record: %w", err)
		}
		// Copy the line: the scanner reuses its buffer across iterations.
		line := append([]byte(nil), sc.Bytes()...)
		if err := fn(probe.Kind, line); err != nil {
			return err
		}
	}
	if err := sc.Err(); err != nil {
		return fmt.Errorf("ledger: scan %s: %w", l.path, err)
	}
	return nil
}

// Close releases the underlying file handle. It takes the write lock so it
// cannot close the handle out from under an in-flight Append/AppendSpend.
func (l *Log) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.f.Close()
}
