package app

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// The mark is the whole visual identity distilled to four squares — a cell
// missing or mis-classed would silently break the "packets" brand everywhere
// the mark mounts (nav today, brand surfaces later).
func TestPacketMark_rendersAllFourCellsOfTheLockedGrid(t *testing.T) {
	t.Parallel()
	body := renderHTML(t, packetMark(22))

	assert.Contains(t, body, `class="pk-mark"`, "the mark is a single grid container")
	assert.Contains(t, body, "--mark-cell: 22px", "the cell size threads through as the one parameterized dimension")
	assert.Equal(t, 4, strings.Count(body, `class="pk-mark__cell`),
		"the mark is exactly four cells — TL/TR/BL/BR, never more or fewer")
	assert.Equal(t, 2, strings.Count(body, "pk-mark__cell--signal"),
		"TL and BL both fill signal in the non-held mark")
	assert.Contains(t, body, `pk-mark__cell--delivered"`, "BR fills delivered")
}

// Below the locked small-size threshold the ghost's 18%-of-cell stroke goes
// sub-pixel and reads as noise rather than a composing state, so the spec
// forbids it outright — a nav-sized (8px) mark must fall back to a solid
// delivered-mid TR cell instead of the ghost outline. 13px sits one pixel
// under the locked boundary, pinning the threshold at exactly 14 rather than
// some looser "small-ish" cutoff.
func TestPacketMark_forbidsTheGhostOutlineBelowTheSmallSizeThreshold(t *testing.T) {
	t.Parallel()
	for _, cell := range []int{8, 13} {
		cell := cell
		t.Run(strings.ToLower(fmt.Sprintf("%dpx", cell)), func(t *testing.T) {
			t.Parallel()
			body := renderHTML(t, packetMark(cell))
			assert.NotContains(t, body, "pk-mark__cell--ghost",
				"below the small-size threshold the mark must never render the ghost-outline cell")
			assert.Contains(t, body, "pk-mark__cell--delivered-mid",
				"below the small-size threshold the TR cell falls back to solid delivered-mid")
		})
	}
}

// At or above the threshold the composing story (a ghost outline promising
// the same edge that later fills solid) is legible again, so the ghost cell
// must render instead of the small-size fallback.
func TestPacketMark_rendersTheGhostOutlineAtOrAboveTheSmallSizeThreshold(t *testing.T) {
	t.Parallel()
	for _, cell := range []int{14, 22} {
		cell := cell
		t.Run(strings.ToLower(fmt.Sprintf("%dpx", cell)), func(t *testing.T) {
			t.Parallel()
			body := renderHTML(t, packetMark(cell))
			assert.Contains(t, body, "pk-mark__cell--ghost",
				"at or above the small-size threshold the mark renders the ghost-outline TR cell")
			assert.NotContains(t, body, "pk-mark__cell--delivered-mid",
				"at or above the small-size threshold the mark must not fall back to the small-size solid TR")
		})
	}
}

// The held variant is the mark's own "something is blocking" story: BL burns
// risk-red instead of the calm signal fill, leaving the grid shape, TL, TR,
// and BR completely untouched.
func TestPacketMarkHeld_burnsBottomLeftRiskInsteadOfSignal(t *testing.T) {
	t.Parallel()
	body := renderHTML(t, packetMarkHeld(22))

	assert.Equal(t, 4, strings.Count(body, `class="pk-mark__cell`),
		"the held mark is still exactly four cells")
	assert.Contains(t, body, "pk-mark__cell--held",
		"the held variant marks BL with its own risk-colored class")
	assert.Equal(t, 1, strings.Count(body, "pk-mark__cell--signal"),
		"only TL stays signal-colored once BL is held")
	assert.Contains(t, body, "pk-mark__cell--ghost",
		"TR keeps its ordinary ghost-composing state — held only changes BL")
	assert.Contains(t, body, `pk-mark__cell--delivered"`,
		"BR keeps its ordinary delivered fill — held only changes BL")
}

// The stacked lockup is the compact in-chrome brand form (mark + "packets"
// over a per-surface label) — the only wordmark form the spec allows beside a
// breadcrumb, so nav can mount it without violating the "never mark + full
// wordmark + breadcrumb" rule.
func TestPacketLockup_rendersMarkWithWordmarkOverSublabel(t *testing.T) {
	t.Parallel()
	body := renderHTML(t, packetLockup(8, "console"))

	assert.Contains(t, body, `class="pk-mark"`, "the lockup mounts the mark")
	assert.Contains(t, body, "pk-mark__cell--delivered-mid",
		"at 8px the lockup's mark still obeys the small-size rule")
	assert.Contains(t, body, "pk-lockup__word", "the wordmark line has its own class hook")
	assert.Contains(t, body, ">packets<", "the wordmark reads lowercase 'packets' as content — never capitalized")
	assert.Contains(t, body, "pk-lockup__sub", "the sublabel line has its own class hook")
	assert.Contains(t, body, ">console<",
		"the sublabel carries the per-surface label verbatim as lowercase content — CSS, not content, uppercases it")
}
