package app

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// DRY: .review-answer__submit reuses .pk-btn (which already owns the hairline
// border). The submit rule must NOT re-declare a full `border:` shorthand — it only
// reinforces the accent via `border-color`, letting .pk-btn own the border width/style.
func TestReviewSubmit_doesNotRedeclareTheFullBPacketShorthand(t *testing.T) {
	require.Contains(t, packetsStyle, "border-color: var(--signal)",
		".review-answer__submit reinforces the accent via border-color, not a full border shorthand")
	require.NotContains(t, packetsStyle, "border: 1px solid var(--signal)",
		".review-answer__submit must not re-declare the hairline border that .pk-btn already owns")
}

// Token port: the state grammar IS the product (the design system's
// "Brand pack") — a wrong hex silently breaks the honest-state legibility the
// whole design depends on, so the exact values are pinned, not just presence.
func TestBaseStylesheet_definesStateGrammarTokensWithExactHexValues(t *testing.T) {
	for _, tt := range []struct {
		name  string
		token string
		hex   string
	}{
		{"in-flight/you/accent reads signal-cyan", "--signal", "#4cc4d4"},
		{"a real catch reads verified-green", "--verified", "#46c08a"},
		{"an advisory hold reads held-amber", "--held", "#e6b23e"},
		{"a blocking hold reads risk-red", "--risk", "#f0666b"},
		{"agent authorship reads agent-purple", "--agent", "#a78bfa"},
		{"a landed/settled thing reads delivered-teal", "--delivered", "#2a7683"},
		{"the page ground is near-black", "--ground", "#0a0d13"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Containsf(t, packetsStyle, tt.token+": "+tt.hex,
				"%s must be exactly %s", tt.token, tt.hex)
		})
	}
}

// The type system is IBM Plex end to end (mono = the machine's voice, sans =
// prose) — without the import the base font stacks fall back to the system
// font and the whole voice distinction is lost.
func TestBaseStylesheet_loadsIBMPlexFonts(t *testing.T) {
	require.Contains(t, packetsStyle,
		`fonts.googleapis.com/css2?family=IBM+Plex+Sans:wght@400;500;600;700&family=IBM+Plex+Mono:wght@400;500;600;700`,
		"the stylesheet must @import IBM Plex Sans + Mono at weights 400/500/600/700")
	require.Contains(t, packetsStyle, "--font-ui:", "the sans (prose) stack must be a named token")
	require.Contains(t, packetsStyle, "--font-mono:", "the mono (machine voice) stack must be a named token")
}

// The three motion primitives (live pulse, held pulse, settle flow) are the
// ONLY animation kinds the design allows (guardrail: no bounces/slides) — a
// missing keyframe silently strands whatever rule references it.
func TestBaseStylesheet_definesTheThreeMotionKeyframes(t *testing.T) {
	for _, name := range []string{"pk-pulse", "pk-held-pulse", "pk-flow"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			require.Containsf(t, packetsStyle, "@keyframes "+name,
				"the stylesheet must define @keyframes %s", name)
		})
	}
}

// A surviving `--pk-` token would mean some rule still points at the retired
// skin instead of the ported brand pack — the two palettes must never
// coexist, or a surface could silently re-skin back to the old system.
func TestBaseStylesheet_carriesNoLegacyPkTokens(t *testing.T) {
	require.NotContains(t, packetsStyle, "--pk-",
		"no `--pk-` custom property may remain once the token port is complete")
}

// The design system marks --heading-cream, --glow-cta, --fs-h2, and --fs-h1 as
// marketing-only ("never in-app"). A verbatim copy-paste of the design/tokens
// source files would wrongly drag these into the product surface, so their
// absence is pinned as its own contract, not just a review note.
func TestBaseStylesheet_excludesMarketingOnlyTokens(t *testing.T) {
	for _, token := range []string{"--heading-cream", "--glow-cta", "--fs-h2", "--fs-h1"} {
		t.Run(token, func(t *testing.T) {
			t.Parallel()
			require.NotContainsf(t, packetsStyle, token,
				"%s is marketing-only and must never reach the in-app stylesheet", token)
		})
	}
}
