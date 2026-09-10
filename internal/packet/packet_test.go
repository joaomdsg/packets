package packet_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/joaomdsg/packets/internal/packet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func causedBy(s string) *string { return &s }

func validPacket() *packet.Packet {
	return &packet.Packet{
		Goal:        "Fix the login timeout",
		Terminal:    []string{"true"},
		Constraints: []string{"true"},
		Budget:      packet.Budget{Retries: 5, Minutes: 60},
	}
}

func TestSaveLoad_roundTripsAllFields(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "packet.yaml")
	want := validPacket()
	want.Context = "some context"
	want.CausedBy = causedBy("other-packet")

	require.NoError(t, packet.Save(path, want))
	got, err := packet.Load(path)

	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestValidate_acceptsAWellFormedPacket(t *testing.T) {
	t.Parallel()

	failures := packet.Validate(validPacket(), "")

	assert.Empty(t, failures)
}

func TestValidate_collectsEveryFailingRule(t *testing.T) {
	t.Parallel()
	p := &packet.Packet{
		Goal:        "",
		Terminal:    nil,
		Constraints: []string{"", "if this is not"},
		Budget:      packet.Budget{Retries: 0, Minutes: 0},
	}

	failures := packet.Validate(p, "")

	assert.Contains(t, failures, "goal: must not be empty")
	assert.Contains(t, failures, "terminal: must contain at least one entry")
	assert.Contains(t, failures, "constraints[0]: must not be empty")
	assert.Contains(t, failures, `constraints[1]: does not parse as a shell command: "if this is not"`)
	assert.Contains(t, failures, "budget.retries: must be >= 1")
	assert.Contains(t, failures, "budget.minutes: must be >= 1")
}

func TestValidate_allowsHumanApprovalAsTerminalEntry(t *testing.T) {
	t.Parallel()
	p := validPacket()
	p.Terminal = []string{"human_approval"}

	failures := packet.Validate(p, "")

	assert.Empty(t, failures)
}

func TestValidate_rejectsCausedByPacketThatDoesNotExist(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	p := validPacket()
	p.CausedBy = causedBy("missing-packet")

	failures := packet.Validate(p, dir)

	assert.Contains(t, failures, `caused_by: packet "missing-packet" not found in `+dir)
}

func TestValidate_acceptsCausedByPacketThatExists(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dir, "other-packet"), 0o700))
	p := validPacket()
	p.CausedBy = causedBy("other-packet")

	failures := packet.Validate(p, dir)

	assert.Empty(t, failures)
}

func TestTemplate_isValidYAML(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "packet.yaml")

	require.NoError(t, os.WriteFile(path, []byte(packet.Template), 0o600))
	_, err := packet.Load(path)

	require.NoError(t, err)
}
