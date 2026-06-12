package app

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// FLOW B: spend (balance hue) and place-order (bandwidth hue) both FUND work but
// read as unrelated controls gating on different currencies with no shared story —
// the most confusing moment in the loop. They co-locate under ONE "fund work" group
// with a dim two-currency explainer, so the Lead sees both funding moves and the
// currency each draws. NOT parallel (shared liveReg/liveFabric).
func TestLiveCard_colocatesBothFundingMovesUnderOneGroup(t *testing.T) {
	body := actNowCardBody(t)

	require.Contains(t, body, "fund-work__label", "the funding moves share one labelled group")
	require.Contains(t, strings.ToLower(body), "fund work", "the group is labelled 'fund work'")
	require.Contains(t, body, "fund-work__explainer", "a dim one-line two-currency explainer accompanies the group")
	require.Contains(t, body, "balance spends a catch", "the explainer names the balance currency")
	require.Contains(t, body, "bandwidth places a live order", "the explainer names the bandwidth currency")

	// Both affordances render UNDER the one group, not as scattered siblings.
	start := strings.Index(body, "fund-work__label")
	require.GreaterOrEqual(t, start, 0)
	// The group div opens before its label; slice from the group's own class to the
	// end of the act-now section to scope the membership assertions.
	groupStart := strings.LastIndex(body[:start], `class="fund-work"`)
	require.GreaterOrEqual(t, groupStart, 0, "the label sits inside a .fund-work group")
	rest := body[groupStart:]
	benchAt := strings.Index(rest, `class="bench"`)
	if benchAt < 0 {
		benchAt = len(rest)
	}
	group := rest[:benchAt]
	require.Contains(t, group, "spend-action", "Spend (balance) is inside the fund-work group")
	require.Contains(t, group, "compose__place", "Place order (bandwidth) is inside the fund-work group")
}

// FLOW B (guardrail): unifying the funding story must NOT introduce any
// meter/gauge/progress/ratio/fill markup — the honest-state aesthetic forbids it
// (constraint 3). The funding group is a labelled affordance PAIR only.
func TestLiveCard_fundingGroupAddsNoGaugeOrMeterMarkup(t *testing.T) {
	body := strings.ToLower(actNowCardBody(t))

	for _, banned := range []string{"progress-bar", "<progress", "<meter", "role=\"progressbar\"", "gauge"} {
		require.NotContains(t, body, banned,
			"the funding group is a labelled pair, never a "+banned)
	}
}
