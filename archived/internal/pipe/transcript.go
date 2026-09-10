package pipe

import "github.com/joaomdsg/packets/internal/catch"

// Transcript is the deterministic, verdict-relevant projection of a CycleResult:
// exactly the fields the host re-derives a catch from. It deliberately OMITS the
// CycleResult.Trace — those beats carry wall-clock timestamps that are
// non-deterministic and irrelevant to the verdict — so the same work serializes
// to byte-identical JSON. It is the structured form a sandboxed verifier (#6c)
// emits and the host re-derives the verdict from; the SAME RunCatchCycle runs
// in-process and in the cage, and Transcribe projects either result identically.
type Transcript struct {
	Outcome catch.Outcome   `json:"outcome"`
	Reason  Reason          `json:"reason"`
	Path    string          `json:"path"`
	Line    int             `json:"line"`
	Land    LandState       `json:"land"`
	Before  catch.LineState `json:"before"`
	After   catch.LineState `json:"after"`
}

// Transcribe projects a CycleResult onto its deterministic verdict Transcript,
// dropping the non-deterministic Trace.
func Transcribe(r CycleResult) Transcript {
	return Transcript{
		Outcome: r.Outcome,
		Reason:  r.Reason,
		Path:    r.Path,
		Line:    r.Line,
		Land:    r.Land,
		Before:  r.Before,
		After:   r.After,
	}
}
