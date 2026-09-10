package packet

// LearningThreshold is the minimum number of settled packets a repo needs
// before it's honestly called "converged" (mirrors watch.go's IsNoisy
// sample-size bar of 5: enough real judgment to trust, not a single lucky
// packet). Below it, the repo stays "learning" — never a fabricated verdict.
const LearningThreshold = 5

// SettledCount counts packets whose lifecycle carries real judgment —
// Verified, Held, or Delivered — mirroring internal/app's console settled
// rail exactly (composing/in-flight packets have produced no judgment yet).
func SettledCount(packets []Packet) int {
	var n int
	for _, p := range packets {
		if p.State == Verified || p.State == Held || p.State == Delivered {
			n++
		}
	}
	return n
}

// Converged reports whether a repo has accumulated enough real settled
// history to call its learning period over (a repo new to Packets
// starts in "learning," never in a fabricated converged state).
func Converged(packets []Packet) bool {
	return SettledCount(packets) >= LearningThreshold
}
