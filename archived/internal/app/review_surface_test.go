package app_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/go-via/via/vt"
)

// With no surviving mutants (or before a cycle ran), /review shows a calm empty
// state — never a fabricated or alarming surface. NOT parallel.
func TestReviewCard_showsCalmEmptyStateWhenNoOpenQuestions(t *testing.T) {
	server, _ := bootDefaultServer(t, defaultBootCfg)

	body := bodyOf(vt.NewClient(t, server, "/review").HTML())
	require.NotContains(t, body, "review-thread", "no threads when the oracle left no survivors")
	require.Contains(t, body, "No open questions", "a calm empty state, not a blank or alarming surface")
}
