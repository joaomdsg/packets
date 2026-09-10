package ci

import "time"

// Clock abstracts wall-clock reads and waits so the poll loop can be
// driven by a virtual clock in tests instead of a real sleep.
type Clock interface {
	Now() time.Time
	Sleep(d time.Duration)
}

// RealClock sleeps for real.
type RealClock struct{}

func (RealClock) Now() time.Time        { return time.Now() }
func (RealClock) Sleep(d time.Duration) { time.Sleep(d) }
