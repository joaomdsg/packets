package app

import (
	"math/rand"

	"github.com/joaomdsg/packets/internal/packet"
)

// drawCalibration selects the Console's calibration sample (attention
// economics — the calibration draw): a skim-worthy pick from the
// auto-forwarded set (packets whose State is
// Verified — reached that state without ever being held). The draw is
// STABLE across renders — if previous still qualifies, it is kept verbatim
// rather than re-rolled every poll tick — and only falls back to a fresh
// random pick when previous has aged out (removed entirely, or demoted to a
// non-Verified state). ok is false only when nothing currently qualifies.
func drawCalibration(packets []packet.Packet, previous int) (id int, ok bool) {
	var candidates []int
	for _, p := range packets {
		if p.State == packet.Verified {
			candidates = append(candidates, p.ID)
		}
	}
	if len(candidates) == 0 {
		return 0, false
	}
	for _, c := range candidates {
		if c == previous {
			return previous, true
		}
	}
	return candidates[rand.Intn(len(candidates))], true
}
