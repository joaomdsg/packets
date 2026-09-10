package build

import "time"

// Clock abstracts wall-clock reads so tests can control elapsed time
// without sleeping (§0.5, minutes_used accounting).
type Clock interface {
	Now() time.Time
}

// RealClock is time.Now.
type RealClock struct{}

func (RealClock) Now() time.Time { return time.Now() }
