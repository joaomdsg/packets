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
	Outcome           catch.Outcome `json:"outcome"`
	Path              string        `json:"path"`
	Line              int           `json:"line"`
	BeforeRev         string        `json:"before_rev"`
	AfterRev          string        `json:"after_rev"`
	BeforeInventory   []string      `json:"before_inventory"`
	AfterInventory    []string      `json:"after_inventory"`
	MutantsConsidered int           `json:"mutants_considered"`
	ReasonTag         string        `json:"reason_tag"`
	SelfFlagged       bool          `json:"self_flagged"`
	WouldHaveShipped  bool          `json:"would_have_shipped"`
}

// ShouldRecord reports whether an outcome warrants a ledger entry: only a real
// mint (Catch) is recorded, so no-op churn, no-catch, no-oracle-signal, and
// partial-catch leave no trace (the farm-denial invariant).
func ShouldRecord(o catch.Outcome) bool {
	return o == catch.Catch
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

// Records reads back every appended record in order.
func (l *Log) Records() ([]CatchRecord, error) {
	f, err := os.Open(l.path)
	if err != nil {
		return nil, fmt.Errorf("ledger: read %s: %w", l.path, err)
	}
	defer f.Close()

	var out []CatchRecord
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if len(sc.Bytes()) == 0 {
			continue
		}
		var r CatchRecord
		if err := json.Unmarshal(sc.Bytes(), &r); err != nil {
			return nil, fmt.Errorf("ledger: decode record: %w", err)
		}
		out = append(out, r)
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("ledger: scan %s: %w", l.path, err)
	}
	return out, nil
}

// Close releases the underlying file handle.
func (l *Log) Close() error {
	return l.f.Close()
}
