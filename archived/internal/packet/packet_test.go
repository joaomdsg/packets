package packet_test

import (
	"fmt"
	"testing"

	"github.com/joaomdsg/packets/internal/ledger"
	"github.com/joaomdsg/packets/internal/packet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFold_derivesNameAsASlugOfTheFirstThreeWordsOfThePrompt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		prompt string
		id     int
		want   string
	}{
		{"three-plus words keeps only the first three", "Fix the rate limiter now", 1, "fix-the-rate"},
		{"exactly three words", "Fix the limiter", 1, "fix-the-limiter"},
		{"two words uses both", "Fix limiter", 1, "fix-limiter"},
		{"one word uses it alone", "Fix", 1, "fix"},
		{"empty prompt falls back to the order id", "", 7, "pkt-7"},
		{"whitespace-only prompt falls back to the order id", "   ", 9, "pkt-9"},
		{"non letter/digit runes are dropped from each word", "Fix! the-rate #limiter", 1, "fix-therate-limiter"},
		{"punctuation-only words clean to empty and are skipped, not fabricated", "--- the limiter", 1, "the-limiter"},
		{"all three candidate words clean to empty falls back to the order id", "--- *** !!!", 3, "pkt-3"},
		{"digits are kept", "Bump v2 config", 1, "bump-v2-config"},
		{"an all-digit word is kept as-is", "123 abc def", 1, "123-abc-def"},
		{"multi-byte unicode letters are kept and lowercased", "Café Latte Fix", 1, "café-latte-fix"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			views := []ledger.SendView{{
				ID:     tt.id,
				Target: ledger.Target{Prompt: tt.prompt},
				Status: "queued",
			}}

			got := packet.Fold(views, packet.Addr{}, zeroQuestions)

			require.Len(t, got, 1)
			assert.Equal(t, tt.want, got[0].Name)
		})
	}
}

func TestFold_copiesPacketFactsThroughUnchanged(t *testing.T) {
	t.Parallel()

	addr := packet.Addr{Owner: "acme", Name: "widgets"}
	views := []ledger.SendView{{
		ID:      42,
		Target:  ledger.Target{Prompt: "fix the thing", BaseRev: "base123", FixRev: "fix456"},
		Status:  "done",
		Caught:  true,
		Verdict: "tested",
	}}

	got := packet.Fold(views, addr, constQuestions(5))

	require.Len(t, got, 1)
	p := got[0]
	assert.Equal(t, 42, p.ID)
	assert.Equal(t, addr, p.Addr)
	assert.Equal(t, "fix the thing", p.Intent)
	assert.Equal(t, "base123", p.BaseRev)
	assert.Equal(t, "fix456", p.FixRev)
	assert.True(t, p.Caught)
	assert.Equal(t, "tested", p.Verdict)
	assert.Equal(t, 5, p.OpenQuestions)
}

// None of these statuses is a real ACK signal — delivered must stay
// unreachable for all of them, regardless of caught/questions, since none
// carries the host-issued deploy evidence a real ACK requires.
func TestPacket_isNeverDeliverableFromNonACKStatuses(t *testing.T) {
	t.Parallel()

	statuses := []string{"queued", "running", "done", "failed", "cancelled"}
	caughtValues := []bool{true, false}
	questionCounts := []int{0, 1, 3}

	for _, status := range statuses {
		for _, caught := range caughtValues {
			for _, questions := range questionCounts {
				name := fmt.Sprintf("status=%s caught=%v questions=%d", status, caught, questions)
				t.Run(name, func(t *testing.T) {
					t.Parallel()

					views := []ledger.SendView{{
						ID:     1,
						Target: ledger.Target{Prompt: "irrelevant prompt here"},
						Status: status,
						Caught: caught,
					}}
					got := packet.Fold(views, packet.Addr{}, constQuestions(questions))

					require.Len(t, got, 1)
					assert.NotEqual(t, packet.Delivered, got[0].State,
						"delivered must never be reachable from a non-ACK status")
					assert.False(t, got[0].Deliverable(),
						"Deliverable() stays false without real ACK evidence")
				})
			}
		}
	}
}

// "deployed" is the one status a real host-issued ACK command (the
// `packets deployed` command) produces — the only path that reaches Delivered.
func TestFold_deployedStatusReachesDeliveredAndIsDeliverable(t *testing.T) {
	t.Parallel()

	views := []ledger.SendView{{
		ID:     1,
		Target: ledger.Target{Prompt: "ship the thing"},
		Status: "deployed",
		Caught: true,
	}}
	got := packet.Fold(views, packet.Addr{}, zeroQuestions)

	require.Len(t, got, 1)
	assert.Equal(t, packet.Delivered, got[0].State)
	assert.Equal(t, packet.HoldNone, got[0].Hold, "a delivered packet is not held")
	assert.True(t, got[0].Deliverable(), "State==Delivered is exactly what Deliverable() reports")
}

// "regressed" (`packets regressed`) routes a packet back to a blocking hold —
// a deploy that broke in prod demands the same attention as any other hard
// stop, never a silent Delivered.
func TestFold_regressedStatusRoutesBackToHeldBlocking(t *testing.T) {
	t.Parallel()

	views := []ledger.SendView{{
		ID:     1,
		Target: ledger.Target{Prompt: "ship the thing"},
		Status: "regressed",
		Caught: true,
	}}
	got := packet.Fold(views, packet.Addr{}, zeroQuestions)

	require.Len(t, got, 1)
	assert.Equal(t, packet.Held, got[0].State)
	assert.Equal(t, packet.HoldBlocking, got[0].Hold)
	assert.Equal(t, "deployment regression", got[0].HoldReason,
		"a real 'regressed' case, not the generic unknown-state catch-all (which would say \"unknown state · regressed\")")
	assert.False(t, got[0].Deliverable())
}

func zeroQuestions(int) int { return 0 }

func constQuestions(n int) func(int) int {
	return func(int) int { return n }
}
