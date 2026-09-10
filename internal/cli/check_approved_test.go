package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/joaomdsg/packets/internal/cli"
	"github.com/joaomdsg/packets/internal/packet"
	"github.com/joaomdsg/packets/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckApproved_exitsZeroIffApprovedIsTrue(t *testing.T) {
	tests := []struct {
		name     string
		approved bool
		wantErr  bool
	}{
		{"approved", true, false},
		{"not approved", false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// t.Setenv (inside runFixture) forbids t.Parallel.
			pkt := &packet.Packet{Goal: "fix x", Terminal: []string{"human_approval"}, Budget: packet.Budget{Retries: 3, Minutes: 30}}
			st := &state.State{Slug: "fix-x", Version: 1, State: state.AwaitingApproval, Approved: tt.approved}
			fabricSlug := runFixture(t, "fix-x", st, pkt)

			out := &bytes.Buffer{}
			root := cli.NewRoot()
			root.SetOut(out)
			root.SetArgs([]string{"check-approved", "--fabric", fabricSlug, "fix-x"})

			err := root.Execute()

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Empty(t, out.String(), "check-approved must stay quiet on stdout")
		})
	}
}

func TestCheckApproved_writesNoLogLine(t *testing.T) {
	// t.Setenv (inside runFixture) forbids t.Parallel.
	pkt := &packet.Packet{Goal: "fix x", Terminal: []string{"human_approval"}, Budget: packet.Budget{Retries: 3, Minutes: 30}}
	st := &state.State{Slug: "fix-x", Version: 1, State: state.AwaitingApproval, Approved: true}
	fabricSlug := runFixture(t, "fix-x", st, pkt)

	root := cli.NewRoot()
	root.SetArgs([]string{"check-approved", "--fabric", fabricSlug, "fix-x"})
	require.NoError(t, root.Execute())

	_, err := os.Stat(filepath.Join(packetDirFor(t, fabricSlug, "fix-x"), "log.jsonl"))
	assert.True(t, os.IsNotExist(err), "check-approved must not write a log line")
}
