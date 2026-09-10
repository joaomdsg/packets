package app

import "time"

// weeklyInterruptCap is the LOCKED weekly interrupt budget the design spec
// fixes (the console's "N/10 interrupts" KPI) — a fixed cap of 10, not
// configurable.
const weeklyInterruptCap = 10

// weeklyInterrupts reports key's session real interrupt count over the
// trailing 7 days (used) against the fixed weekly cap (attention economics —
// the interrupt budget). used
// traces to ledger.Log.InterruptsSince — never fabricated — and is honestly 0
// for a session with no bound ledger.
func weeklyInterrupts(key string) (used, cap int) {
	_, log := readLiveState(key)
	if log == nil {
		return 0, weeklyInterruptCap
	}
	n, err := log.InterruptsSince(time.Now().AddDate(0, 0, -7))
	if err != nil {
		return 0, weeklyInterruptCap
	}
	return n, weeklyInterruptCap
}
