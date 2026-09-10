package packet_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/joaomdsg/packets/internal/packet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The whole point of an adversarial probe (concepts.md: "actively tries to
// break packets; seeds probes to keep everyone honest") is that TODAY's real
// gate must catch a KNOWN-bad revision — this is a build-time regression
// check on the gate's own integrity, not a fixture sanity check.
func TestRunAdversarialProbe_theRealBuildVetGateCatchesASeededKnownBadRevision(t *testing.T) {
	t.Parallel()

	report, err := packet.RunAdversarialProbe(context.Background())
	require.NoError(t, err)

	assert.True(t, report.Caught, "a healthy gate must catch the seeded known-bad revision, never let it pass")
	assert.Equal(t, packet.GateFailed, report.Gate.Status)
	assert.Contains(t, report.Gate.Detail, "go build",
		"the detail must name the real command that failed — proves RunBuildVetGate genuinely ran against a broken module, not a stubbed result")
}

// A probe that can't even materialize its own fixture (a scratch-dir
// failure) must surface that honestly as an error — never silently report a
// fabricated Caught/escaped verdict about a gate that never ran. NOT
// parallel (t.Setenv).
func TestRunAdversarialProbe_surfacesAScratchDirFailureAsAnErrorNotAFabricatedVerdict(t *testing.T) {
	t.Setenv("TMPDIR", filepath.Join(t.TempDir(), "does-not-exist"))

	_, err := packet.RunAdversarialProbe(context.Background())

	assert.Error(t, err, "a probe that never ran must report an error, not a verdict")
}

func TestProbeReport_stringNamesEscapedProbesDistinctlyFromCaughtOnes(t *testing.T) {
	t.Parallel()

	caught := packet.ProbeReport{Name: "known-bad-build", Gate: packet.Gate{Status: packet.GateFailed, Detail: "go build: syntax error"}, Caught: true}
	assert.Contains(t, caught.String(), "caught")
	assert.NotContains(t, caught.String(), "ESCAPED")

	escaped := packet.ProbeReport{Name: "known-bad-build", Gate: packet.Gate{Status: packet.GatePassed, Detail: "ok"}, Caught: false}
	assert.Contains(t, escaped.String(), "ESCAPED")
	assert.NotContains(t, escaped.String(), "caught")
}
