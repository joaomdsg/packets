package report_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/joaomdsg/packets/internal/journal"
	"github.com/joaomdsg/packets/internal/packet"
	"github.com/joaomdsg/packets/internal/registry"
	"github.com/joaomdsg/packets/internal/report"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompute_countsGateEventsAcrossFabricAndPacketLogs(t *testing.T) {
	t.Parallel()
	fabDir := t.TempDir()

	require.NoError(t, journal.Append(filepath.Join(fabDir, "log.jsonl"), journal.Entry{
		Event: journal.EventEmitReject, Actor: journal.ActorHuman,
	}))

	packetDir := filepath.Join(fabDir, "packets", "fix-x")
	require.NoError(t, os.MkdirAll(packetDir, 0o700))
	codes := []any{"AMBIGUOUS_GOAL", "LOOSE_CONSTRAINTS"}
	require.NoError(t, journal.Append(filepath.Join(packetDir, "log.jsonl"), journal.Entry{
		Event: journal.EventEmitWarn, Actor: journal.ActorHuman, Detail: map[string]any{"codes": codes},
	}))
	require.NoError(t, journal.Append(filepath.Join(packetDir, "log.jsonl"), journal.Entry{
		Event: journal.EventEmitOverride, Actor: journal.ActorHuman, Detail: map[string]any{"codes": []any{"AMBIGUOUS_GOAL"}},
	}))
	require.NoError(t, packet.Save(filepath.Join(packetDir, "packet.yaml"), &packet.Packet{
		Goal: "fix x", Terminal: []string{"true"}, Budget: packet.Budget{Retries: 3, Minutes: 30},
	}))

	r, err := report.Compute(fabDir)

	require.NoError(t, err)
	assert.Equal(t, 1, r.GateRejects)
	assert.Equal(t, 1, r.GateWarns)
	assert.Equal(t, 1, r.GateOverrides)
	assert.Equal(t, 2, r.WarningCodeCounts["AMBIGUOUS_GOAL"])
	assert.Equal(t, 1, r.WarningCodeCounts["LOOSE_CONSTRAINTS"])
}

func TestCompute_splitsTerminationsByHumanApprovalPresence(t *testing.T) {
	t.Parallel()
	fabDir := t.TempDir()

	approvedDir := filepath.Join(fabDir, "packets", "needs-approval")
	require.NoError(t, os.MkdirAll(approvedDir, 0o700))
	require.NoError(t, packet.Save(filepath.Join(approvedDir, "packet.yaml"), &packet.Packet{
		Goal: "a", Terminal: []string{"human_approval"}, Budget: packet.Budget{Retries: 1, Minutes: 1},
	}))
	require.NoError(t, journal.Append(filepath.Join(approvedDir, "log.jsonl"), journal.Entry{Event: journal.EventTerminate}))

	autoDir := filepath.Join(fabDir, "packets", "auto")
	require.NoError(t, os.MkdirAll(autoDir, 0o700))
	require.NoError(t, packet.Save(filepath.Join(autoDir, "packet.yaml"), &packet.Packet{
		Goal: "b", Terminal: []string{"true"}, Budget: packet.Budget{Retries: 1, Minutes: 1},
	}))
	require.NoError(t, journal.Append(filepath.Join(autoDir, "log.jsonl"), journal.Entry{Event: journal.EventTerminate}))

	r, err := report.Compute(fabDir)

	require.NoError(t, err)
	assert.Equal(t, 1, r.TerminateWithApproval)
	assert.Equal(t, 1, r.TerminateWithoutApproval)
}

func TestCompute_countsAccretionsByCauseAndHaltsByReason(t *testing.T) {
	t.Parallel()
	fabDir := t.TempDir()

	require.NoError(t, registry.Append(filepath.Join(fabDir, "registry.jsonl"), registry.Entry{
		Cause: registry.CauseGateSuggested, Path: "test/a.spec.js",
	}))
	require.NoError(t, registry.Append(filepath.Join(fabDir, "registry.jsonl"), registry.Entry{
		Cause: registry.CauseCIFailure, Path: "test/b.spec.js",
	}))

	packetDir := filepath.Join(fabDir, "packets", "fix-x")
	require.NoError(t, os.MkdirAll(packetDir, 0o700))
	reason := "smell:max_files"
	require.NoError(t, journal.Append(filepath.Join(packetDir, "log.jsonl"), journal.Entry{
		Event: journal.EventSmellHalt, Reason: &reason,
	}))
	require.NoError(t, journal.Append(filepath.Join(packetDir, "log.jsonl"), journal.Entry{
		Event: journal.EventBudgetHalt, Reason: strPtr("budget"),
	}))

	r, err := report.Compute(fabDir)

	require.NoError(t, err)
	assert.Equal(t, 1, r.AccretionByCause[registry.CauseGateSuggested])
	assert.Equal(t, 1, r.AccretionByCause[registry.CauseCIFailure])
	assert.Equal(t, 1, r.HaltsByReason["smell:max_files"])
	assert.Equal(t, 1, r.HaltsByReason["budget"])
	assert.Equal(t, 1, r.BudgetHalts)
}

func TestCompute_countsLaterPacketsFailingTheSameCIFailureAccretionPath(t *testing.T) {
	t.Parallel()
	fabDir := t.TempDir()

	require.NoError(t, registry.Append(filepath.Join(fabDir, "registry.jsonl"), registry.Entry{
		Timestamp: "2026-01-01T00:00:00Z", Packet: "origin", Path: "test/flaky.spec.js", Cause: registry.CauseCIFailure,
	}))

	laterDir := filepath.Join(fabDir, "packets", "later")
	require.NoError(t, os.MkdirAll(laterDir, 0o700))
	require.NoError(t, journal.Append(filepath.Join(laterDir, "log.jsonl"), journal.Entry{
		Timestamp: "2026-01-02T00:00:00Z", Event: journal.EventCIPred,
		Detail: map[string]any{"cmd": "npm test test/flaky.spec.js", "exit": float64(1), "kind": "ci"},
	}))

	earlierDir := filepath.Join(fabDir, "packets", "earlier")
	require.NoError(t, os.MkdirAll(earlierDir, 0o700))
	require.NoError(t, journal.Append(filepath.Join(earlierDir, "log.jsonl"), journal.Entry{
		Timestamp: "2025-12-31T00:00:00Z", Event: journal.EventCIPred,
		Detail: map[string]any{"cmd": "npm test test/flaky.spec.js", "exit": float64(1), "kind": "ci"},
	}))

	r, err := report.Compute(fabDir)

	require.NoError(t, err)
	assert.Equal(t, 1, r.CIFailureRepeats["test/flaky.spec.js"], "only the later packet's failure counts")
}

func TestCompute_computesMeanAmendsAndAmendToTerminateGap(t *testing.T) {
	t.Parallel()
	fabDir := t.TempDir()

	packetDir := filepath.Join(fabDir, "packets", "fix-x")
	require.NoError(t, os.MkdirAll(packetDir, 0o700))
	require.NoError(t, journal.Append(filepath.Join(packetDir, "log.jsonl"), journal.Entry{
		Timestamp: "2026-01-01T00:00:00Z", Event: journal.EventAmend,
	}))
	require.NoError(t, journal.Append(filepath.Join(packetDir, "log.jsonl"), journal.Entry{
		Timestamp: "2026-01-01T00:10:00Z", Event: journal.EventContainerExit, Tokens: 500,
	}))
	require.NoError(t, journal.Append(filepath.Join(packetDir, "log.jsonl"), journal.Entry{
		Timestamp: "2026-01-01T01:00:00Z", Event: journal.EventTerminate,
	}))

	otherDir := filepath.Join(fabDir, "packets", "no-amend")
	require.NoError(t, os.MkdirAll(otherDir, 0o700))

	r, err := report.Compute(fabDir)

	require.NoError(t, err)
	assert.Equal(t, 1, r.AmendCount)
	assert.InDelta(t, 0.5, r.AmendMeanPerPacket, 0.001)
	assert.InDelta(t, 500, r.AmendMeanTokens, 0.001)
	assert.InDelta(t, 60, r.AmendMeanMinutes, 0.001)
}

func TestCompute_skipsUnparseableLinesInsteadOfFailing(t *testing.T) {
	t.Parallel()
	fabDir := t.TempDir()

	fabricLogPath := filepath.Join(fabDir, "log.jsonl")
	require.NoError(t, journal.Append(fabricLogPath, journal.Entry{Event: journal.EventEmitReject}))
	appendRawLine(t, fabricLogPath, "not json")

	registryPath := filepath.Join(fabDir, "registry.jsonl")
	require.NoError(t, registry.Append(registryPath, registry.Entry{Cause: registry.CauseProposal}))
	appendRawLine(t, registryPath, "not json")

	packetDir := filepath.Join(fabDir, "packets", "fix-x")
	require.NoError(t, os.MkdirAll(packetDir, 0o700))
	packetLogPath := filepath.Join(packetDir, "log.jsonl")
	require.NoError(t, journal.Append(packetLogPath, journal.Entry{Event: journal.EventTerminate}))
	appendRawLine(t, packetLogPath, "not json")

	r, err := report.Compute(fabDir)

	require.NoError(t, err)
	assert.Equal(t, 1, r.GateRejects)
	assert.Equal(t, 1, r.AccretionByCause[registry.CauseProposal])
	assert.Equal(t, 1, r.TerminateWithoutApproval)
	require.Len(t, r.Skipped, 3)
	paths := map[string]int{}
	for _, s := range r.Skipped {
		paths[s.Path] = s.Skipped
	}
	assert.Equal(t, 1, paths[fabricLogPath])
	assert.Equal(t, 1, paths[registryPath])
	assert.Equal(t, 1, paths[packetLogPath])
}

// appendRawLine writes a line that is not valid JSON, simulating a torn
// last line left behind by a crash mid-append.
func appendRawLine(t *testing.T, path, line string) {
	t.Helper()
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o600)
	require.NoError(t, err)
	defer f.Close()
	_, err = f.WriteString(line + "\n")
	require.NoError(t, err)
}

func strPtr(s string) *string { return &s }
