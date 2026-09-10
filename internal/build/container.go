package build

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/joaomdsg/packets/internal/fabric"
	"github.com/joaomdsg/packets/internal/journal"
	"github.com/joaomdsg/packets/internal/packet"
	"github.com/joaomdsg/packets/internal/state"
)

// apiErrorHaltReason is distinct from "budget" so `packets extend --retries`
// (which only applies when halt_reason is "budget") can't be used on a
// packet that halted for an unrelated API failure.
const apiErrorHaltReason = "agent:api_error"

// apiErrorHaltMD renders halt.md for an API-level container failure,
// mirroring the shape an agent-written halt.md has: reason on the first
// line, human-readable detail after.
func apiErrorHaltMD(usage claudeUsage) string {
	return fmt.Sprintf("%s\n\n%s\n\napi_error_status: %d\nterminal_reason: %s\n",
		apiErrorHaltReason, usage.Result, usage.APIErrorStatus, usage.TerminalReason)
}

// runContainer is §12 steps 3, 4, and 4b: spawn the container, account for
// its tokens/minutes, harvest any proposals, and check for an
// agent-written halt or a budget timeout. A non-nil *state.State return
// means the packet halted and Run should return immediately; a nil
// *state.State with a nil error means the caller should continue to step 5.
func runContainer(
	deps Deps, fab *fabric.Config, pkt *packet.Packet, st *state.State,
	packetDir, runDir, signalsDir string,
	log logFunc, save func() error, halt func(reason, event string) (*state.State, error),
) (*state.State, error) {
	timeoutMinutes := float64(pkt.Budget.Minutes) - st.MinutesUsed
	timeoutSeconds := int(timeoutMinutes * 60)
	if timeoutSeconds < 0 {
		timeoutSeconds = 0
	}
	args, err := containerArgs(fab, deps.Getenv, deps.Exists, st.Slug, st.Attempt, timeoutSeconds, runDir)
	if err != nil {
		return nil, err
	}

	start := deps.Clock.Now()
	output, exitCode, err := deps.Docker.Run(args)
	elapsed := deps.Clock.Now().Sub(start)
	if err != nil {
		return nil, fmt.Errorf("build: docker run: %s", err)
	}
	if err := os.WriteFile(filepath.Join(runDir, "container.log"), []byte(output), 0o600); err != nil {
		return nil, fmt.Errorf("build: write container.log: %s", err)
	}

	usage, blob, parseErr := parseClaudeUsage(output)
	if parseErr == nil {
		if err := os.WriteFile(filepath.Join(runDir, "claude-output.json"), []byte(blob), 0o600); err != nil {
			return nil, fmt.Errorf("build: write claude-output.json: %s", err)
		}
	}
	tokens := usage.Usage.InputTokens + usage.Usage.OutputTokens
	st.TokensUsed += tokens
	st.MinutesUsed += elapsed.Minutes()
	if err := save(); err != nil {
		return nil, err
	}
	if err := log(journal.EventContainerExit, "", tokens, map[string]any{
		"exit_code":      exitCode,
		"total_cost_usd": usage.TotalCostUSD,
	}); err != nil {
		return nil, err
	}

	// A container that fails at the API level (e.g. credit exhaustion) can
	// still exit 0 with subtype "success" — is_error is the only signal
	// worth trusting. Halt now, before proposals/smell/commit: nothing the
	// agent "did" here reflects on the packet, so it must not be judged on
	// the merits or run through the smell recheck.
	if usage.IsError {
		haltMD := apiErrorHaltMD(usage)
		if err := os.WriteFile(filepath.Join(packetDir, "halt.md"), []byte(haltMD), 0o600); err != nil {
			return nil, fmt.Errorf("build: write halt.md: %s", err)
		}
		st2, err := halt(apiErrorHaltReason, journal.EventAgentHalt)
		return st2, err
	}

	copiedProposals, err := harvestProposals(signalsDir, packetDir)
	if err != nil {
		return nil, err
	}
	for _, name := range copiedProposals {
		if err := log(journal.EventProposal, "", 0, map[string]any{"file": name}); err != nil {
			return nil, err
		}
	}

	// Step 4: agent-requested (or hook-raised) halt.
	haltMD, haltExists, err := readHaltMD(signalsDir)
	if err != nil {
		return nil, err
	}
	if haltExists {
		if err := os.WriteFile(filepath.Join(packetDir, "halt.md"), []byte(haltMD), 0o600); err != nil {
			return nil, fmt.Errorf("build: copy halt.md: %s", err)
		}
		reason := haltReason(haltMD)
		event := journal.EventAgentHalt
		if strings.HasPrefix(reason, "smell:") {
			event = journal.EventSmellHalt
		}
		st2, err := halt(reason, event)
		return st2, err
	}

	// Step 4b: container timeout.
	if exitCode == dockerTimeoutExitCode {
		st2, err := halt("budget", journal.EventBudgetHalt)
		return st2, err
	}

	return nil, nil
}
