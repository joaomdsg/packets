package packet_test

import (
	"testing"

	"github.com/joaomdsg/packets/internal/packet"
	"github.com/stretchr/testify/assert"
)

func TestLifecycle_stringMatchesTheDesignStateGrammar(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		state packet.Lifecycle
		want  string
	}{
		{"composing", packet.Composing, "composing"},
		{"in-flight", packet.InFlight, "in-flight"},
		{"verified", packet.Verified, "verified"},
		{"held", packet.Held, "held"},
		{"delivered", packet.Delivered, "delivered"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.state.String())
		})
	}
}

func TestLifecycle_stringFailsSafeOnAnOutOfRangeValue(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "unknown", packet.Lifecycle(99).String())
}

func TestHoldKind_stringIsEmptyWhenNotHeld(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		kind packet.HoldKind
		want string
	}{
		{"no hold renders as empty, not a placeholder word", packet.HoldNone, ""},
		{"advisory", packet.HoldAdvisory, "advisory"},
		{"blocking", packet.HoldBlocking, "blocking"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.kind.String())
		})
	}
}
