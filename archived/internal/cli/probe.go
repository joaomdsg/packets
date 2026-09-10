package cli

import (
	"context"
	"fmt"
	"io"

	"github.com/joaomdsg/packets/internal/packet"
)

// runProbe is the `probe` subcommand: runs the adversarial inspection mode
// (design/guidelines/concepts.md) on demand — a self-contained known-bad
// revision through the real gate, reported honestly. Exits non-zero when a
// probe escapes, so it doubles as a gate-health check a human or CI can run
// anytime.
func runProbe(out io.Writer) error {
	report, err := packet.RunAdversarialProbe(context.Background())
	if err != nil {
		return fmt.Errorf("probe: %w", err)
	}
	fmt.Fprintln(out, report.String())
	if !report.Caught {
		return fmt.Errorf("probe: %s escaped", report.Name)
	}
	return nil
}
