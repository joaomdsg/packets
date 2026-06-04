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

	"github.com/joaomdsg/agntpr/internal/catch"
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
}

// ShouldRecord reports whether an outcome warrants a ledger entry: only a real
// mint (Catch) is recorded, so no-op churn, no-catch, no-oracle-signal, and
// partial-catch leave no trace (the farm-denial invariant).
func ShouldRecord(o catch.Outcome) bool {
	return o == catch.Catch
}

// kindSpend tags a debit line. A catch line carries NO kind field, so logs
// written before spends existed re-read byte-identically.
const kindSpend = "spend"

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
// A Log is single-writer: it holds one append handle and is NOT safe for
// concurrent Append, or for Append/Records/Close racing each other. Callers
// must serialize access. This matches the single-Lead prototype model (one
// owner mints catches); a multi-writer ledger would need an external lock or a
// mutex around the handle and is out of scope here.
type Log struct {
	path string
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
	if _, err := l.f.Write(append(line, '\n')); err != nil {
		return fmt.Errorf("ledger: append: %w", err)
	}
	return nil
}

// Records reads back every appended CATCH record in order. Spend (debit) lines
// are skipped, so the confirmed-catch count is never polluted by the sink.
func (l *Log) Records() ([]CatchRecord, error) {
	var out []CatchRecord
	err := l.scan(func(kind string, line []byte) error {
		if kind == kindSpend {
			return nil // a debit is not a catch
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

// Close releases the underlying file handle.
func (l *Log) Close() error {
	return l.f.Close()
}
