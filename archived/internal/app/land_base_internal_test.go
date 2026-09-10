package app

import (
	"testing"

	"github.com/joaomdsg/packets/internal/ledger"
)

func order(base string) ledger.PacketRecord {
	return ledger.PacketRecord{Target: ledger.Target{BaseRev: base}}
}

// A legacy anchored session carries an explicit configured base — the land must squash
// against THAT, ignoring the order history, exactly as it always has.
func TestLandBaseRev_explicitConfigBaseWins(t *testing.T) {
	got := landBaseRev("cfgbase", []ledger.PacketRecord{order("o1"), order("o2")})
	if got != "cfgbase" {
		t.Fatalf("an explicit config base must win, got %q", got)
	}
}

// A prompt-first session has NO configured base (board.go zeroes it) — landing it used
// to pass "" to commit-tree and fail "not a valid object name". The base must instead be
// the session's ORIGIN: the earliest order's recorded base (the repo HEAD before any
// harness commit), so the squash collapses all the work onto where it actually started.
func TestLandBaseRev_promptFirstUsesEarliestPacketOrigin(t *testing.T) {
	got := landBaseRev("", []ledger.PacketRecord{order("origin"), order("mid"), order("tip")})
	if got != "origin" {
		t.Fatalf("a prompt-first land must squash onto the earliest order's origin, got %q", got)
	}
}

// Defensive: an empty base on an early order must not strand the land — fall through to
// the first order that actually recorded an origin.
func TestLandBaseRev_skipsEmptyEarlyBases(t *testing.T) {
	got := landBaseRev("", []ledger.PacketRecord{order(""), order("realorigin"), order("tip")})
	if got != "realorigin" {
		t.Fatalf("must skip empty bases and use the first real origin, got %q", got)
	}
}

// Orders that ALL recorded an empty base give nothing to squash onto — yield "" so the
// caller errors honestly rather than fabricating a commit with no parent.
func TestLandBaseRev_allEmptyBasesYieldEmpty(t *testing.T) {
	if got := landBaseRev("", []ledger.PacketRecord{order(""), order("")}); got != "" {
		t.Fatalf("all-empty bases must yield empty, got %q", got)
	}
}

// With no configured base AND no orders to derive one from, there is no honest base —
// return "" so the caller surfaces a real error rather than fabricating a bogus commit.
func TestLandBaseRev_noConfigNoPacketsYieldsEmpty(t *testing.T) {
	if got := landBaseRev("", nil); got != "" {
		t.Fatalf("no derivable base must yield empty, got %q", got)
	}
}
