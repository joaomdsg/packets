// Package journal appends events to a packet's log.jsonl.
package journal

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
)

// Event names, exact strings per the log format spec.
const (
	EventEmitReject           = "emit_reject"
	EventEmitWarn             = "emit_warn"
	EventEmitOverride         = "emit_override"
	EventEmit                 = "emit"
	EventSlugCollision        = "slug_collision"
	EventEnterNode            = "enter_node"
	EventContainerStart       = "container_start"
	EventContainerExit        = "container_exit"
	EventLocalPred            = "local_pred"
	EventSmellHalt            = "smell_halt"
	EventAgentHalt            = "agent_halt"
	EventBudgetHalt           = "budget_halt"
	EventConflictHalt         = "conflict_halt"
	EventPush                 = "push"
	EventPROpen               = "pr_open"
	EventCIPoll               = "ci_poll"
	EventCIPred               = "ci_pred"
	EventRebase               = "rebase"
	EventMerge                = "merge"
	EventResume               = "resume"
	EventExtend               = "extend"
	EventAmend                = "amend"
	EventKill                 = "kill"
	EventApprove              = "approve"
	EventProposal             = "proposal"
	EventAccrete              = "accrete"
	EventTerminate            = "terminate"
	EventGateLLMError         = "gate_llm_error"
	EventRegistryDetectFailed = "registry_detect_failed"
	EventError                = "error"

	// EventStatus, EventList, and EventProposalsListed are not in §10's
	// event vocabulary, which only names events for mutating seams. §7's
	// "every seam command logs" rule still applies to the three read-only
	// seams (status, list, proposals), so these extend the vocabulary
	// rather than reuse an unrelated name.
	EventStatus          = "status"
	EventList            = "list"
	EventProposalsListed = "proposals"
)

// Actor values.
const (
	ActorHuman   = "human"
	ActorHarness = "harness"
	ActorAgent   = "agent"
)

// Entry is one log.jsonl line. Every field is always present in the
// marshaled JSON; pointer fields marshal as null when not applicable.
type Entry struct {
	Timestamp string         `json:"ts"`
	Slug      string         `json:"slug"`
	Version   int            `json:"version"`
	Node      *string        `json:"node"`
	Event     string         `json:"event"`
	Reason    *string        `json:"reason"`
	Tokens    int            `json:"tokens"`
	Actor     string         `json:"actor"`
	Detail    map[string]any `json:"detail"`
}

// Append writes e as one JSON line to the log.jsonl at path, creating the
// file if it does not exist.
func Append(path string, e Entry) error {
	if e.Detail == nil {
		e.Detail = map[string]any{}
	}

	data, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("journal: marshal entry: %s", err)
	}

	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o600)
	if err != nil {
		return fmt.Errorf("journal: open %s: %s", path, err)
	}
	defer f.Close()

	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("journal: write %s: %s", path, err)
	}
	return nil
}

// ReadAll parses every line of the log.jsonl at path, in order. A missing
// file is not an error: it reads as no entries, which is the normal state
// for a fabric-level log before any Stage A rejection has ever happened.
func ReadAll(path string) ([]Entry, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("journal: open %s: %s", path, err)
	}
	defer f.Close()

	var entries []Entry
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var e Entry
		if err := json.Unmarshal(line, &e); err != nil {
			return nil, fmt.Errorf("journal: parse %s: %s", path, err)
		}
		entries = append(entries, e)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("journal: read %s: %s", path, err)
	}
	return entries, nil
}

// ReadAllTolerant is ReadAll but skips lines that fail to parse instead of
// failing outright, returning how many were skipped. A torn last line is
// exactly what a crash mid-append leaves behind; callers that need to
// report metrics from a fabric's full history (packets report) cannot let
// one such line make every line unreadable.
func ReadAllTolerant(path string) ([]Entry, int, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, 0, nil
		}
		return nil, 0, fmt.Errorf("journal: open %s: %s", path, err)
	}
	defer f.Close()

	var entries []Entry
	var skipped int
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var e Entry
		if err := json.Unmarshal(line, &e); err != nil {
			skipped++
			continue
		}
		entries = append(entries, e)
	}
	if err := scanner.Err(); err != nil {
		return nil, skipped, fmt.Errorf("journal: read %s: %s", path, err)
	}
	return entries, skipped, nil
}
