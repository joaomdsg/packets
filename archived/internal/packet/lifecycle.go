package packet

// Lifecycle is a packet's state in the design concept model's state grammar
// (composing → in-flight → verified|held → delivered).
type Lifecycle int

// The lifecycle states, in the order a packet naturally progresses through
// them. Delivered exists as a value but is UNREACHABLE from Fold today — see
// Packet.Deliverable.
const (
	Composing Lifecycle = iota
	InFlight
	Verified
	Held
	Delivered
)

// String renders the lowercase, hyphenated form used across the UI's mono
// voice ("in-flight", not "InFlight").
func (l Lifecycle) String() string {
	switch l {
	case Composing:
		return "composing"
	case InFlight:
		return "in-flight"
	case Verified:
		return "verified"
	case Held:
		return "held"
	case Delivered:
		return "delivered"
	default:
		return "unknown"
	}
}

// HoldKind distinguishes why a Held packet is held: sampled attention
// (HoldAdvisory) vs. a required stop (HoldBlocking).
type HoldKind int

// The hold kinds. HoldNone is the zero value, so a Packet that was never held
// carries it for free.
const (
	HoldNone HoldKind = iota
	HoldAdvisory
	HoldBlocking
)

// String renders "" for HoldNone (a non-held packet has nothing to say about
// holds), "advisory", or "blocking".
func (k HoldKind) String() string {
	switch k {
	case HoldAdvisory:
		return "advisory"
	case HoldBlocking:
		return "blocking"
	default:
		return ""
	}
}
