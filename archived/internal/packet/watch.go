// Package packet's watch mechanic is the STANDING inspection mode: a fixed
// set of pre-defined triggers, each evaluated against
// real packet facts, each carrying a precision score computed from real
// human fired-vs-useful history — never a fabricated score, and never a
// human-authored condition nobody has verified. A trigger that proves
// mostly noise loses the right to interrupt (IsNoisy).
package packet

// WatchKind names one of the three canonical STANDING triggers
// (design/guidelines/concepts.md's "watches/capture triggers, each carrying
// a precision score"). These are pre-defined, not an
// author-your-own DSL — that keeps the mechanic honest and bounded: every
// watch's predicate is a fixed, auditable rule over facts this package
// already computes, never a free-form condition nobody has verified.
type WatchKind int

const (
	// WatchStrictLane fires when a packet's measured blast radius reaches
	// the strict lane — the highest scrutiny tier LaneFor produces today.
	WatchStrictLane WatchKind = iota
	// WatchGateFailure fires when the gauntlet is not forwardable — a hard
	// gate failure (see Gauntlet.Forwardable).
	WatchGateFailure
	// WatchBlockingHold fires when a packet is held with HoldBlocking.
	WatchBlockingHold
)

// String renders the lowercase, hyphenated mono-voice name used across the
// UI. An unrecognized kind renders "unknown" rather than a numeric — fails
// safe/visibly rather than printing a meaningless digit.
func (k WatchKind) String() string {
	switch k {
	case WatchStrictLane:
		return "strict-lane"
	case WatchGateFailure:
		return "gate-failure"
	case WatchBlockingHold:
		return "blocking-hold"
	default:
		return "unknown"
	}
}

// WatchFire is one recorded occurrence of a watch's predicate matching a
// packet. Useful is nil until a human marks it — Precision only ever scores
// MARKED fires, never an unmarked one, so a watch with no human judgment yet
// reports "no history yet" rather than a fabricated score.
type WatchFire struct {
	Kind     WatchKind
	PacketID int
	AtUnixMs int64
	Useful   *bool
}

// EvaluateWatch reports whether kind's predicate matches p. An unrecognized
// kind NEVER fires — fail closed, not open, even when p's own facts would
// satisfy a different, real kind's predicate.
func EvaluateWatch(kind WatchKind, p Packet) bool {
	switch kind {
	case WatchStrictLane:
		return p.Lane == LaneStrict
	case WatchGateFailure:
		return !p.Gauntlet.Forwardable()
	case WatchBlockingHold:
		return p.Hold == HoldBlocking
	default:
		return false
	}
}

// Precision folds fires into kind's precision score: the fraction of this
// kind's MARKED fires (Useful != nil) that were marked useful. An unmarked
// fire never enters the sample — only human judgment counts. ok is false
// when the sample is empty (zero marked fires of this kind), the honest "no
// history yet" case a caller must render literally, never as a synthetic
// 0% or 100%.
func Precision(fires []WatchFire, kind WatchKind) (score float64, sampled int, ok bool) {
	var usefulCount int
	for _, f := range fires {
		if f.Kind != kind || f.Useful == nil {
			continue
		}
		sampled++
		if *f.Useful {
			usefulCount++
		}
	}
	if sampled == 0 {
		return 0, 0, false
	}
	return float64(usefulCount) / float64(sampled), sampled, true
}

// IsNoisy reports whether a watch has lost the right to interrupt ("noisy
// triggers lose the right to interrupt"). A trigger needs
// a real sample before judgment (sampled >= 5 — a handful of marks is
// enough to distrust it, but a single unlucky mark isn't) AND a majority of
// its marked fires must have been useless (score < 0.5) — losing the
// interrupt right is a majority-noise bar, not a single-miss bar.
func IsNoisy(score float64, sampled int) bool {
	return sampled >= 5 && score < 0.5
}
