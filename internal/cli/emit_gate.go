package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/joaomdsg/packets/internal/gate"
	"github.com/joaomdsg/packets/internal/journal"
)

// gateOutcome is what runEmit needs from Stage B to finish (or abandon)
// creating the packet.
type gateOutcome struct {
	terminal               []string
	gateSuggestedTerminals []string
	entries                []journal.Entry // buffered; persisted only if the packet is actually created
	proceed                bool
}

// repoListing returns the top-level entries of repoPath, one per line, for
// the gate prompt's REPO_FILES section.
func repoListing(repoPath string) (string, error) {
	entries, err := os.ReadDir(repoPath)
	if err != nil {
		return "", fmt.Errorf("list %s: %s", repoPath, err)
	}
	names := make([]string, len(entries))
	for i, e := range entries {
		names[i] = e.Name()
	}
	sort.Strings(names)
	return strings.Join(names, "\n"), nil
}

// runGate executes Stage B: the LLM review, then the interactive warning,
// suggestion, and override prompts. in/out drive the [y/N] prompts so this
// is testable without a TTY.
func runGate(llm gate.LLM, packetYAML string, terminal []string, repoFiles string, in io.Reader, out io.Writer) gateOutcome {
	reader := bufio.NewReader(in)
	outcome := gateOutcome{terminal: terminal}

	result, err := gate.Run(llm, packetYAML, repoFiles)
	if err != nil {
		// Never block emit on LLM failure: proceed with zero warnings.
		reason := err.Error()
		outcome.entries = append(outcome.entries, journal.Entry{
			Event:  journal.EventGateLLMError,
			Reason: &reason,
			Actor:  journal.ActorHuman,
		})
		outcome.proceed = true
		return outcome
	}

	for _, w := range result.Warnings {
		fmt.Fprintf(out, "[%s] %s\n", w.Code, w.Detail)
	}

	for _, suggestion := range result.SuggestedTerminal {
		fmt.Fprintf(out, "Suggested terminal: %s\n", suggestion)
		if promptYesNo(out, reader, "Add to terminal? [y/N]") {
			outcome.terminal = append(outcome.terminal, suggestion)
			outcome.gateSuggestedTerminals = append(outcome.gateSuggestedTerminals, suggestion)
		}
	}

	if len(result.Warnings) == 0 {
		outcome.proceed = true
		return outcome
	}

	codes := make([]string, len(result.Warnings))
	for i, w := range result.Warnings {
		codes[i] = w.Code
	}
	outcome.entries = append(outcome.entries, journal.Entry{
		Event:  journal.EventEmitWarn,
		Actor:  journal.ActorHuman,
		Detail: map[string]any{"codes": codes},
	})

	if !promptYesNo(out, reader, "Emit anyway? [y/N]") {
		outcome.proceed = false
		return outcome
	}

	outcome.entries = append(outcome.entries, journal.Entry{
		Event:  journal.EventEmitOverride,
		Actor:  journal.ActorHuman,
		Detail: map[string]any{"codes": codes},
	})
	outcome.proceed = true
	return outcome
}

func promptYesNo(out io.Writer, reader *bufio.Reader, prompt string) bool {
	fmt.Fprintf(out, "%s ", prompt)
	line, _ := reader.ReadString('\n')
	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "y" || answer == "yes"
}
