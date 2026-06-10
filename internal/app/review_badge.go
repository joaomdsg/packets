package app

import (
	"net/url"

	"github.com/go-via/via/h"
)

// reviewQuestionsBadge renders the calm, gated "open questions" summary on the
// live card: when the fix oracle left surviving/undetermined mutants, the verdict
// can read green while honest test gaps remain ("green is a lie here"). The badge
// states the COUNT only — a humble summary, not the full threads — and LINKS to the
// /review surface for that session, where the full anchored threads live, so it
// never clutters the dense card. Returns nil when there are no open questions
// (count empty or "0"), so the caller omits it entirely.
func reviewQuestionsBadge(count, key string) h.H {
	if count == "" || count == "0" {
		return nil
	}
	noun := "open questions"
	if count == "1" {
		noun = "open question"
	}
	return h.A(
		h.Href("/review?key="+url.QueryEscape(key)),
		h.Class("review-questions"),
		h.Data("count", count),
		h.Text(count+" "+noun+" — the oracle found unkilled mutants the tests didn't catch"),
	)
}
