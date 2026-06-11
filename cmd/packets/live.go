package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/joaomdsg/packets/internal/ledger"
)

// promptDelim matches the prompt key — at the spec start or after a comma, with
// optional surrounding whitespace — so the free-text after it (which may contain
// commas and '=') is captured whole rather than comma-split.
var promptDelim = regexp.MustCompile(`(^|,)\s*prompt\s*=`)

// liveFlag collects repeatable -live specs (one prompt-bearing live work-order
// target each), mirroring backlogFlag.
type liveFlag struct{ specs []string }

func (l *liveFlag) String() string { return strings.Join(l.specs, " ") }

func (l *liveFlag) Set(v string) error {
	l.specs = append(l.specs, v)
	return nil
}

// parseLiveSpec parses a "file=F,line=N,base=SHA[,tip=SHA],prompt=<task>" spec into
// a PROMPT-BEARING live work-order Target — the Lead's task plus the PRE-SPECIFIED
// anchor (file/line) the catch is checked against (the R70 anti-farming firewall:
// the trusted Lead names the target, never the agent's own diff). There is no
// FixRev — a real Claude Code harness PRODUCES the fix at run time, so tip defaults
// to base and runLiveOrder diffs against the live HEAD. The LineHash is left unset
// (the re-anchor identity is computed at wiring time via git, like parseBacklogSpec).
//
// prompt= is the trailing free-text (everything after it is the task, so a
// natural-language prompt may contain commas and '='); it must therefore come LAST,
// or a key placed after it is swallowed and its missing anchor key fails the parse.
func parseLiveSpec(spec string) (ledger.Target, error) {
	loc := promptDelim.FindStringIndex(spec)
	if loc == nil {
		return ledger.Target{}, fmt.Errorf("live %q: missing prompt", spec)
	}
	prompt := strings.TrimSpace(spec[loc[1]:])
	if prompt == "" {
		return ledger.Target{}, fmt.Errorf("live %q: empty prompt", spec)
	}

	head := strings.TrimRight(strings.TrimSpace(spec[:loc[0]]), ", ")
	kv := map[string]string{}
	for _, pair := range strings.Split(head, ",") {
		k, v, ok := strings.Cut(pair, "=")
		if !ok {
			return ledger.Target{}, fmt.Errorf("live %q: %q is not key=value", spec, pair)
		}
		kv[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}
	for _, req := range []string{"base", "file", "line"} {
		if kv[req] == "" {
			return ledger.Target{}, fmt.Errorf("live %q: missing %s", spec, req)
		}
	}
	line, err := strconv.Atoi(kv["line"])
	if err != nil || line < 1 {
		return ledger.Target{}, fmt.Errorf("live %q: line must be a positive integer", spec)
	}
	tip := kv["tip"]
	if tip == "" {
		tip = kv["base"]
	}
	return ledger.Target{
		BaseRev: kv["base"],
		TipRev:  tip,
		Path:    kv["file"],
		Line:    line,
		Prompt:  prompt,
	}, nil
}
