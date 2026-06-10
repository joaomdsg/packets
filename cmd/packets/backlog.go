package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/joaomdsg/packets/internal/ledger"
)

// backlogFlag collects repeatable -backlog specs (one fundable work-order target
// each), mirroring sessionFlag.
type backlogFlag struct{ specs []string }

func (b *backlogFlag) String() string { return strings.Join(b.specs, " ") }

func (b *backlogFlag) Set(v string) error {
	b.specs = append(b.specs, v)
	return nil
}

// parseBacklogSpec parses a "base=SHA,fix=SHA,file=F,line=N[,tip=SHA]" spec into a
// fundable work-order Target. tip defaults to fix (clean integration by
// construction), mirroring parseSessionSpec. The LineHash is left unset — the
// re-anchor identity is computed at wiring time via git, not in this pure parser.
func parseBacklogSpec(spec string) (ledger.Target, error) {
	kv := map[string]string{}
	for _, pair := range strings.Split(spec, ",") {
		k, v, ok := strings.Cut(pair, "=")
		if !ok {
			return ledger.Target{}, fmt.Errorf("backlog %q: %q is not key=value", spec, pair)
		}
		kv[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}
	for _, req := range []string{"base", "fix", "file", "line"} {
		if kv[req] == "" {
			return ledger.Target{}, fmt.Errorf("backlog %q: missing %s", spec, req)
		}
	}
	line, err := strconv.Atoi(kv["line"])
	if err != nil || line < 1 {
		return ledger.Target{}, fmt.Errorf("backlog %q: line must be a positive integer", spec)
	}
	tip := kv["tip"]
	if tip == "" {
		tip = kv["fix"]
	}
	return ledger.Target{
		BaseRev: kv["base"],
		FixRev:  kv["fix"],
		TipRev:  tip,
		Path:    kv["file"],
		Line:    line,
	}, nil
}
