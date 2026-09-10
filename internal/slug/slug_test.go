package slug_test

import (
	"testing"
	"time"

	"github.com/joaomdsg/packets/internal/slug"
	"github.com/stretchr/testify/assert"
)

func TestFabric_derivesSlugFromRemoteURL(t *testing.T) {
	tests := []struct {
		name   string
		remote string
		want   string
	}{
		{"plan worked example", "git@github.com:acme/Auth-Service.git", "auth-service"},
		{"https remote", "https://github.com/acme/widget-factory.git", "widget-factory"},
		{"no .git suffix", "git@github.com:acme/Widgets", "widgets"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, slug.Fabric(tt.remote))
		})
	}
}

func TestPacket_derivesSlugFromGoal(t *testing.T) {
	tests := []struct {
		name string
		goal string
		want string
	}{
		{
			"plan worked example",
			"Fix the login timeout on slow networks",
			"fix-the-login-timeout",
		},
		{
			"punctuation collapses to single dashes",
			"Improve, clean-up!! the widgets",
			"improve-clean-up-the-widgets",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, slug.Packet(tt.goal))
		})
	}
}

func TestPacket_capsAt40Chars(t *testing.T) {
	t.Parallel()

	got := slug.Packet("Reallylongwordthatexceedsfortycharacters otherword third fourth")

	assert.LessOrEqual(t, len(got), 40)
}

func TestDeconflict_returnsBaseWhenNoCollision(t *testing.T) {
	t.Parallel()
	exists := func(string) bool { return false }

	got, collided := slug.Deconflict("fix-login", time.Now(), exists)

	assert.Equal(t, "fix-login", got)
	assert.False(t, collided)
}

func TestDeconflict_appendsDateSuffixOnCollision(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC)
	exists := func(candidate string) bool { return candidate == "fix-login" }

	got, collided := slug.Deconflict("fix-login", now, exists)

	assert.Equal(t, "fix-login-0910", got)
	assert.True(t, collided)
}

func TestDeconflict_appendsCounterWhenDateSuffixAlsoTaken(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC)
	taken := map[string]bool{
		"fix-login":        true,
		"fix-login-0910":   true,
		"fix-login-0910-2": true,
	}
	exists := func(candidate string) bool { return taken[candidate] }

	got, collided := slug.Deconflict("fix-login", now, exists)

	assert.Equal(t, "fix-login-0910-3", got)
	assert.True(t, collided)
}
