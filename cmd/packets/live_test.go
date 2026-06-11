package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/packets/internal/ledger"
)

// A -live spec seeds a PROMPT-BEARING work order: the Lead names the task plus the
// PRE-SPECIFIED anchor (file/line) the catch is checked against. No fix rev — the
// agent produces it — so tip defaults to base.
func TestParseLiveSpec_seedsAPromptBearingTargetWithThePreSpecifiedAnchor(t *testing.T) {
	t.Parallel()
	got, err := parseLiveSpec("file=internal/auth.go,line=42,base=abc123,prompt=fix the off-by-one")
	require.NoError(t, err)
	assert.Equal(t, ledger.Target{
		BaseRev: "abc123",
		TipRev:  "abc123", // tip defaults to base; the agent produces the fix, so no FixRev
		Path:    "internal/auth.go",
		Line:    42,
		Prompt:  "fix the off-by-one",
	}, got)
}

// The prompt is free-text the Lead writes: it must survive commas and '=' (a real
// task description), so prompt= is the trailing remainder, not a comma-split pair.
func TestParseLiveSpec_capturesAPromptContainingCommasAndEquals(t *testing.T) {
	t.Parallel()
	got, err := parseLiveSpec("file=a.go,line=5,base=b,prompt=fix x, then y = z")
	require.NoError(t, err)
	assert.Equal(t, "fix x, then y = z", got.Prompt, "the whole trailing free-text is the prompt")
}

func TestParseLiveSpec_honoursAnExplicitTip(t *testing.T) {
	t.Parallel()
	got, err := parseLiveSpec("file=a.go,line=5,base=b,tip=t,prompt=do it")
	require.NoError(t, err)
	assert.Equal(t, "t", got.TipRev, "an explicit tip overrides the base default")
}

func TestParseLiveSpec_trimsSurroundingWhitespace(t *testing.T) {
	t.Parallel()
	got, err := parseLiveSpec(" file = a.go , line = 5 , base = b , prompt =  fix it  ")
	require.NoError(t, err)
	assert.Equal(t, "a.go", got.Path)
	assert.Equal(t, "b", got.BaseRev)
	assert.Equal(t, "fix it", got.Prompt, "the prompt is trimmed of surrounding whitespace")
}

// prompt= is the trailing remainder, so it must come LAST: a key placed after it
// is swallowed into the free-text, leaving its required anchor key missing — which
// fail-closes (rejected) rather than silently mis-targeting the catch.
func TestParseLiveSpec_requiresPromptToBeLast(t *testing.T) {
	t.Parallel()
	_, err := parseLiveSpec("line=5,base=b,prompt=do it,file=a.go") // file after prompt → swallowed → missing file
	assert.Error(t, err, "a required key after prompt= is swallowed, so the spec is rejected")
}

func TestParseLiveSpec_rejectsMalformedOrIncompleteSpecs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		spec string
	}{
		{"missing prompt", "file=a.go,line=5,base=b"},
		{"empty prompt", "file=a.go,line=5,base=b,prompt="},
		{"whitespace-only prompt", "file=a.go,line=5,base=b,prompt=   "},
		{"missing file", "line=5,base=b,prompt=do it"},
		{"missing line", "file=a.go,base=b,prompt=do it"},
		{"missing base", "file=a.go,line=5,prompt=do it"},
		{"non-numeric line", "file=a.go,line=x,base=b,prompt=do it"},
		{"non-positive line", "file=a.go,line=0,base=b,prompt=do it"},
		{"malformed pair", "file=a.go,line,base=b,prompt=do it"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := parseLiveSpec(tt.spec)
			assert.Error(t, err, "spec %q must be rejected", tt.spec)
		})
	}
}
