package app

import (
	"strconv"

	"github.com/go-via/via"
	"github.com/go-via/via/h"
)

// ReviewCard is the dedicated review surface (/review?key=<session>): the full
// anchored "question:" threads the card's badge only counts. Each thread is a
// surviving/undetermined mutant the fix oracle found — an honest test gap the green
// verdict hides. It is READ-ONLY and diagnostic: the threads are the session's
// latest connect-cycle findings (recomputed each cycle, off the economy ledger), so
// answering a question — strengthening the test until the mutant dies — makes it
// vanish on the next cycle (the mastery loop), never a scored transaction.
type ReviewCard struct {
	Key string `query:"key"`
}

// View renders the session's open question-threads, anchored File:Line with their
// Conventional-Comment body, or a calm empty state when the oracle left none.
func (c *ReviewCard) View(_ *via.CtxR) h.H {
	navKey := c.Key
	if navKey == "" {
		navKey = defaultSessionKey
	}
	threads := sessionOpenThreads(navKey)
	parts := []h.H{h.Class("review"), h.Data("state", "review"), navHeader(navKey)}
	if len(threads) == 0 {
		parts = append(parts, h.Div(h.Class("review__empty"),
			h.Text("No open questions — the oracle killed every mutant it tried (or this session hasn't run a cycle yet).")))
		return h.Div(parts...)
	}
	parts = append(parts, h.P(h.Class("review__lead"),
		h.Text(strconv.Itoa(len(threads))+" open — surviving mutants the tests didn't catch:")))
	for _, t := range threads {
		parts = append(parts, h.Div(
			h.Class("review-thread"),
			h.Data("file", t.File),
			h.Data("line", strconv.Itoa(t.StartLine)),
			h.Span(h.Class("review-thread__anchor"), h.Text(t.File+":"+strconv.Itoa(t.StartLine))),
			h.Span(h.Class("review-thread__body"), h.Text(t.Render())), // "question: <body>"
		))
	}
	return h.Div(parts...)
}
