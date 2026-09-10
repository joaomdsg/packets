// Package packet's lane mechanic derives a packet's QoS class from MEASURED
// blast radius — never self-reported, never vibes. The
// pure core (ImportGraph, BlastRadius, LaneFor) is host-side arithmetic over
// a `go list` import graph; the exec seam (LoadImportGraph, ChangedPackages,
// Measure) is the only part that shells out.
//
// Threshold rationale (the WHY, since a number nobody can explain is a number
// nobody should trust): coupling buys scrutiny. A change whose reverse-
// dependency closure touches ≤10% of the module's packages is narrow enough
// that a mistake stays local — best-effort is proportionate. Above that but
// at or under 40%, the blast radius reaches enough of the module that the
// standard gate set earns its cost. Past 40% the change is coupled to most of
// the codebase — strict is the only proportionate response. These are the
// same three bands the design's lane vocabulary names (best-effort/standard/
// strict); the cut points are a deliberate, documented choice, not a
// discovered constant, and may move as the gauntlet's gates mature.
package packet

// Lane is a packet's QoS class, derived from measured coupling (never a
// state — lane chips are neutral pills, not state-grammar colors).
type Lane int

// The lanes, in ascending order of scrutiny bought by blast radius.
// LaneIrreversible is UNREACHABLE from LaneFor today: irreversibility is
// about consequences (data/money/prod side-effects), which no marker in this
// repo declares yet — the same honesty pattern as Packet.Deliverable. LaneFor
// only ever emits the measured, reachable lanes; it never manufactures
// LaneIrreversible.
const (
	LaneUnmeasured Lane = iota
	LaneBestEffort
	LaneStandard
	LaneStrict
	LaneIrreversible
)

// String renders the lowercase, hyphenated mono-voice name used across the UI.
func (l Lane) String() string {
	switch l {
	case LaneUnmeasured:
		return "unmeasured"
	case LaneBestEffort:
		return "best-effort"
	case LaneStandard:
		return "standard"
	case LaneStrict:
		return "strict"
	case LaneIrreversible:
		return "irreversible"
	default:
		return "unknown"
	}
}

// ImportGraph maps a module-local package's import path to the import paths
// it imports — also module-local only (see LoadImportGraph).
type ImportGraph map[string][]string

// BlastRadius measures the reverse-dependency closure of changed: affected is
// the count of changed packages plus every package that transitively imports
// one of them (deduplicated); total is the size of the known graph. A changed
// package absent from graph (an unknown/non-local package) still counts
// itself into affected — it did change — but never inflates total, which
// counts only packages the graph actually knows about.
func BlastRadius(graph ImportGraph, changed []string) (affected, total int) {
	total = len(graph)
	if len(changed) == 0 {
		return 0, total
	}

	// importedBy is the reverse graph: importedBy[x] holds every package that
	// imports x. Built from import VALUES, not just keys, so a package that is
	// only ever imported (never itself a graph key — a sparse/partial graph)
	// still has its importers found.
	importedBy := make(map[string][]string, len(graph))
	for pkg, imports := range graph {
		for _, imp := range imports {
			importedBy[imp] = append(importedBy[imp], pkg)
		}
	}

	reached := make(map[string]bool)
	var queue []string
	for _, c := range changed {
		if !reached[c] {
			reached[c] = true
			queue = append(queue, c)
		}
	}
	for len(queue) > 0 {
		pkg := queue[0]
		queue = queue[1:]
		for _, importer := range importedBy[pkg] {
			if !reached[importer] {
				reached[importer] = true
				queue = append(queue, importer)
			}
		}
	}

	return len(reached), total
}

// LaneFor derives a Lane from measured blast radius. total==0 (nothing known
// about the module) or affected==0 (no changed package was measured — e.g. a
// non-Go-only change) is honestly unmeasured, never defaulted up to a
// permissive lane. See the package doc for the ratio bands' rationale.
func LaneFor(affected, total int) Lane {
	if total == 0 || affected == 0 {
		return LaneUnmeasured
	}
	r := float64(affected) / float64(total)
	switch {
	case r <= 0.10:
		return LaneBestEffort
	case r <= 0.40:
		return LaneStandard
	default:
		return LaneStrict
	}
}
