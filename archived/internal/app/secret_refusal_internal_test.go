package app

import (
	"strings"
	"testing"

	"github.com/joaomdsg/packets/internal/settle"
)

// The pre-push secret gate refuses rather than leak. When a push carries SEVERAL secrets,
// the refusal must name them all in one message — else the Lead fixes one, re-lands, hits
// the next, and burns N round-trips. formatSecretRefusal builds that message.
func TestFormatSecretRefusal_namesEverySecretSoTheyreFixedInOnePass(t *testing.T) {
	t.Run("a single secret reads in the singular and names the one site", func(t *testing.T) {
		msg := formatSecretRefusal([]settle.SecretHit{{File: "a.go", Line: 12, Rule: "aws-access-key-id"}})
		if !strings.Contains(msg, "1 secret detected") {
			t.Errorf("want singular '1 secret detected' lead, got:\n%s", msg)
		}
		if strings.Contains(msg, "secrets detected") {
			t.Errorf("a single hit must not pluralize, got:\n%s", msg)
		}
		if !strings.Contains(msg, "a.go:12 (aws-access-key-id)") {
			t.Errorf("want the site named, got:\n%s", msg)
		}
	})

	t.Run("multiple secrets pluralize and name every site in scan order", func(t *testing.T) {
		hits := []settle.SecretHit{
			{File: "a.go", Line: 12, Rule: "aws-access-key-id"},
			{File: "b.go", Line: 3, Rule: "private-key"},
			{File: "c.go", Line: 99, Rule: "slack-token"},
		}
		msg := formatSecretRefusal(hits)
		if !strings.Contains(msg, "3 secrets detected") {
			t.Errorf("want plural '3 secrets detected', got:\n%s", msg)
		}
		a := strings.Index(msg, "a.go:12 (aws-access-key-id)")
		b := strings.Index(msg, "b.go:3 (private-key)")
		c := strings.Index(msg, "c.go:99 (slack-token)")
		if a < 0 || b < 0 || c < 0 {
			t.Fatalf("every site must be named, got:\n%s", msg)
		}
		if !(a < b && b < c) {
			t.Errorf("sites must appear in input (scan) order; got offsets a=%d b=%d c=%d", a, b, c)
		}
	})

	t.Run("an empty slice is the empty string", func(t *testing.T) {
		if got := formatSecretRefusal(nil); got != "" {
			t.Errorf("empty hits → empty message, got %q", got)
		}
	})
}
