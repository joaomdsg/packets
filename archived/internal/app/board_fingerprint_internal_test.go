package app

import "testing"

// A terminal merge OUTCOME (merged / bounced) changing on a row must move the fleet
// fingerprint, so a board tab live-refreshes the new lifecycle badge without a manual
// reload — the same cross-session "watch the shop" value the counts already get.
func TestFleetFingerprint_movesWhenATerminalLifecycleAppears(t *testing.T) {
	base := []CardRow{{Key: "s"}}
	merged := []CardRow{{Key: "s", LandLifecycle: string(lifecycleMerged)}}
	if fleetFingerprint(base) == fleetFingerprint(merged) {
		t.Fatal("a row gaining a MERGED lifecycle must change the fingerprint so the board live-refreshes")
	}
	bounced := []CardRow{{Key: "s", LandLifecycle: string(lifecycleBounced)}}
	if fleetFingerprint(base) == fleetFingerprint(bounced) {
		t.Fatal("a row gaining a BOUNCED lifecycle must change the fingerprint")
	}
	if fleetFingerprint(merged) == fleetFingerprint(bounced) {
		t.Fatal("merged and bounced are distinct board outcomes — distinct fingerprints")
	}
}

// The fingerprint folds only what the board DISPLAYS: the routine non-terminal
// "landed — not yet merged" transient is hidden on the board (boardLifecycle show=false),
// so a "" → landed transition must NOT move the fingerprint — otherwise the board would
// push a re-render frame that looks identical, breaking the idle-no-flood invariant.
func TestFleetFingerprint_ignoresTheHiddenLandedTransient(t *testing.T) {
	base := []CardRow{{Key: "s"}}
	landed := []CardRow{{Key: "s", LandLifecycle: string(lifecycleLanded)}}
	if fleetFingerprint(base) != fleetFingerprint(landed) {
		t.Fatal("the hidden landed-not-merged transient does not change the board, so it must not move the fingerprint")
	}
	// An unrecognized lifecycle string is also hidden on the board (boardLifecycle
	// show=false) — it must not move the fingerprint and trigger a look-identical frame.
	unknown := []CardRow{{Key: "s", LandLifecycle: "weird"}}
	if fleetFingerprint(base) != fleetFingerprint(unknown) {
		t.Fatal("an unknown lifecycle is hidden on the board, so it must not move the fingerprint")
	}
}
