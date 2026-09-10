package deps_test

import (
	"fmt"
	"testing"

	"github.com/joaomdsg/packets/internal/deps"
	"github.com/stretchr/testify/assert"
)

func TestCheck_reportsMissingBinaries(t *testing.T) {
	tests := []struct {
		name      string
		missing   map[string]bool
		wantError bool
		wantMsg   string
	}{
		{"all present", nil, false, ""},
		{"docker missing", map[string]bool{"docker": true}, true, "docker"},
		{"git and gh missing", map[string]bool{"git": true, "gh": true}, true, "git, gh"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			lookup := func(file string) (string, error) {
				if tt.missing[file] {
					return "", fmt.Errorf("not found")
				}
				return "/usr/bin/" + file, nil
			}

			err := deps.Check(lookup)

			if tt.wantError {
				assert.ErrorContains(t, err, tt.wantMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
