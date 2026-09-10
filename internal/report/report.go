// Package report computes the §16 metrics packets report prints: gate,
// termination, accretion, amend, and halt counts derived from a fabric's
// logs and registry.
package report

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/joaomdsg/packets/internal/journal"
	"github.com/joaomdsg/packets/internal/packet"
	"github.com/joaomdsg/packets/internal/registry"
)

// Report holds every metric packets report computes.
type Report struct {
	GateRejects       int
	GateWarns         int
	GateOverrides     int
	WarningCodeCounts map[string]int

	TerminateWithApproval    int
	TerminateWithoutApproval int

	AccretionByCause map[string]int
	// CIFailureRepeats maps a ci_failure accretion's path to the number of
	// distinct later packets that also failed a predicate referencing it.
	CIFailureRepeats map[string]int

	PacketCount        int
	AmendCount         int
	AmendMeanPerPacket float64
	AmendMeanTokens    float64
	AmendMeanMinutes   float64

	HaltsByReason map[string]int
	BudgetHalts   int
}

type packetLog struct {
	slug    string
	entries []journal.Entry
	pkt     *packet.Packet
}

// Compute reads fabDir's fabric-level log.jsonl, registry.jsonl, and
// every packets/<slug>/log.jsonl, and derives every §16 metric.
func Compute(fabDir string) (*Report, error) {
	fabricEntries, err := journal.ReadAll(filepath.Join(fabDir, "log.jsonl"))
	if err != nil {
		return nil, err
	}
	regEntries, err := registry.ReadAll(filepath.Join(fabDir, "registry.jsonl"))
	if err != nil {
		return nil, err
	}
	packets, err := loadPackets(fabDir)
	if err != nil {
		return nil, err
	}

	r := &Report{
		WarningCodeCounts: map[string]int{},
		AccretionByCause:  map[string]int{},
		CIFailureRepeats:  map[string]int{},
		HaltsByReason:     map[string]int{},
		PacketCount:       len(packets),
	}

	countGateAndHalts(r, fabricEntries)
	for _, p := range packets {
		countGateAndHalts(r, p.entries)
		countTerminations(r, p)
	}
	countAccretions(r, regEntries)
	countRepeatFailures(r, regEntries, packets)
	countAmends(r, packets)

	return r, nil
}

func loadPackets(fabDir string) ([]packetLog, error) {
	dirEntries, err := os.ReadDir(filepath.Join(fabDir, "packets"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var packets []packetLog
	for _, d := range dirEntries {
		if !d.IsDir() {
			continue
		}
		packetDir := filepath.Join(fabDir, "packets", d.Name())
		entries, err := journal.ReadAll(filepath.Join(packetDir, "log.jsonl"))
		if err != nil {
			return nil, err
		}
		// A packet.yaml that fails to load (e.g. mid-emit crash) still
		// contributes its log to every other metric; only the
		// termination-split needs its terminal list.
		pkt, _ := packet.Load(filepath.Join(packetDir, "packet.yaml"))
		packets = append(packets, packetLog{slug: d.Name(), entries: entries, pkt: pkt})
	}
	return packets, nil
}

func countGateAndHalts(r *Report, entries []journal.Entry) {
	for _, e := range entries {
		switch e.Event {
		case journal.EventEmitReject:
			r.GateRejects++
		case journal.EventEmitWarn:
			r.GateWarns++
			addCodes(r, e)
		case journal.EventEmitOverride:
			r.GateOverrides++
			addCodes(r, e)
		case journal.EventBudgetHalt:
			r.BudgetHalts++
			r.HaltsByReason[haltReason(e)]++
		case journal.EventSmellHalt, journal.EventAgentHalt, journal.EventConflictHalt:
			r.HaltsByReason[haltReason(e)]++
		}
	}
}

// haltReason prefers the entry's reason (e.g. "smell:test_modified") over
// its bare event name, since that's the more specific §16 grouping key.
func haltReason(e journal.Entry) string {
	if e.Reason != nil && *e.Reason != "" {
		return *e.Reason
	}
	return e.Event
}

func addCodes(r *Report, e journal.Entry) {
	raw, ok := e.Detail["codes"]
	if !ok {
		return
	}
	codes, ok := raw.([]any)
	if !ok {
		return
	}
	for _, c := range codes {
		if s, ok := c.(string); ok {
			r.WarningCodeCounts[s]++
		}
	}
}

const humanApproval = "human_approval"

func countTerminations(r *Report, p packetLog) {
	hasApproval := false
	if p.pkt != nil {
		for _, t := range p.pkt.Terminal {
			if t == humanApproval {
				hasApproval = true
				break
			}
		}
	}
	for _, e := range p.entries {
		if e.Event != journal.EventTerminate {
			continue
		}
		if hasApproval {
			r.TerminateWithApproval++
		} else {
			r.TerminateWithoutApproval++
		}
	}
}

func countAccretions(r *Report, entries []registry.Entry) {
	for _, e := range entries {
		r.AccretionByCause[e.Cause]++
	}
}

// countRepeatFailures counts, for each ci_failure accretion, the distinct
// later packets whose local_pred/ci_pred logged a non-zero exit for a
// command referencing the same path.
func countRepeatFailures(r *Report, regEntries []registry.Entry, packets []packetLog) {
	for _, acc := range regEntries {
		if acc.Cause != registry.CauseCIFailure {
			continue
		}
		accTime, err := time.Parse(time.RFC3339, acc.Timestamp)
		if err != nil {
			continue
		}
		seen := map[string]bool{}
		for _, p := range packets {
			if p.slug == acc.Packet {
				continue
			}
			for _, e := range p.entries {
				if e.Event != journal.EventLocalPred && e.Event != journal.EventCIPred {
					continue
				}
				if !predFailed(e) || !cmdReferencesPath(e, acc.Path) {
					continue
				}
				ts, err := time.Parse(time.RFC3339, e.Timestamp)
				if err != nil || !ts.After(accTime) {
					continue
				}
				seen[p.slug] = true
			}
		}
		r.CIFailureRepeats[acc.Path] += len(seen)
	}
}

func predFailed(e journal.Entry) bool {
	exit, ok := e.Detail["exit"].(float64)
	return ok && exit != 0
}

func cmdReferencesPath(e journal.Entry, path string) bool {
	cmd, ok := e.Detail["cmd"].(string)
	return ok && strings.Contains(cmd, path)
}

// countAmends computes the mean amends per packet and the mean
// tokens/minutes spent between each amend and the next terminate.
func countAmends(r *Report, packets []packetLog) {
	var totalTokens, totalMinutes float64
	var episodes int
	for _, p := range packets {
		for i, e := range p.entries {
			if e.Event != journal.EventAmend {
				continue
			}
			r.AmendCount++
			amendTime, err := time.Parse(time.RFC3339, e.Timestamp)
			if err != nil {
				continue
			}
			for j := i + 1; j < len(p.entries); j++ {
				if p.entries[j].Event != journal.EventTerminate {
					continue
				}
				termTime, err := time.Parse(time.RFC3339, p.entries[j].Timestamp)
				if err != nil {
					break
				}
				var tokens int
				for k := i + 1; k <= j; k++ {
					tokens += p.entries[k].Tokens
				}
				totalTokens += float64(tokens)
				totalMinutes += termTime.Sub(amendTime).Minutes()
				episodes++
				break
			}
		}
	}
	if r.PacketCount > 0 {
		r.AmendMeanPerPacket = float64(r.AmendCount) / float64(r.PacketCount)
	}
	if episodes > 0 {
		r.AmendMeanTokens = totalTokens / float64(episodes)
		r.AmendMeanMinutes = totalMinutes / float64(episodes)
	}
}
