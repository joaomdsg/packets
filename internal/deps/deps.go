// Package deps checks that the external binaries the harness shells out to
// are present on PATH before any command runs (§0.3).
package deps

import (
	"fmt"
	"strings"
)

// Required lists every external binary packets needs at runtime.
var Required = []string{"git", "gh", "docker"}

// LookPath abstracts exec.LookPath so tests can simulate a missing binary
// without touching the real PATH.
type LookPath func(file string) (string, error)

// Check reports a clear error naming every required binary lookup fails
// to find, or nil if all are present.
func Check(lookup LookPath) error {
	var missing []string
	for _, bin := range Required {
		if _, err := lookup(bin); err != nil {
			missing = append(missing, bin)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("packets: required tool(s) not found on PATH: %s", strings.Join(missing, ", "))
	}
	return nil
}
