package app

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-via/via/h"

	"github.com/joaomdsg/packets/internal/ledger"
)

// renderHTMLNodes renders a slice of nodes (wrapping them in a div) so a
// multi-card render can be asserted as one string.
func renderHTMLNodes(t *testing.T, nodes []h.H) string {
	t.Helper()
	return renderHTML(t, h.Div(nodes...))
}

// A durable annotation renders as a card carrying its real author, its body, and
// its file:line anchor — human-authored, so the author chip shows who actually
// wrote it, never a hardcoded "agent".
func TestRenderAnnotationThreads_showsTheRootAuthorAnchorAndBody(t *testing.T) {
	cards := renderAnnotationThreads(&ReviewCard{Key: "k"}, []annotationThread{{
		Root: ledger.AnnotationRecord{ID: "a", File: "pay/charge.go", StartLine: 42, Author: "lead", Body: "never rejects a negative amount"},
	}})
	body := renderHTMLNodes(t, cards)
	assert.Contains(t, body, "annotation-card")
	assert.Contains(t, body, "lead", "the real author is shown, not a hardcoded agent")
	assert.Contains(t, body, "never rejects a negative amount")
	assert.Contains(t, body, "pay/charge.go:42", "the anchor names the file and line")
}

// Replies render nested under their root, in order, each with its own author —
// the conversation reads top to bottom as it happened.
func TestRenderAnnotationThreads_nestsRepliesUnderTheRootInOrder(t *testing.T) {
	cards := renderAnnotationThreads(&ReviewCard{Key: "k"}, []annotationThread{{
		Root: ledger.AnnotationRecord{ID: "root", File: "a.go", StartLine: 3, Author: "lead", Body: "why not <=?"},
		Replies: []ledger.AnnotationRecord{
			{ID: "r1", ParentID: "root", Author: "agent", Body: "fixed it"},
			{ID: "r2", ParentID: "root", Author: "lead", Body: "thanks"},
		},
	}})
	body := renderHTMLNodes(t, cards)
	assert.Equal(t, 2, strings.Count(body, `annotation-card__reply"`), "both replies render as nested reply rows (exact class, not the reply-form)")
	assert.Contains(t, body, "fixed it")
	assert.Contains(t, body, "thanks")
	first := strings.Index(body, "fixed it")
	second := strings.Index(body, "thanks")
	assert.True(t, first >= 0 && first < second, "replies keep their chronological order")
}

// A file-level annotation (no line) anchors to the file alone — never a
// misleading ":0" that reads like line zero.
func TestRenderAnnotationThreads_fileLevelAnnotationHasNoLineInAnchor(t *testing.T) {
	cards := renderAnnotationThreads(&ReviewCard{Key: "k"}, []annotationThread{{
		Root: ledger.AnnotationRecord{ID: "f", File: "go.mod", Author: "lead", Body: "don't touch"},
	}})
	body := renderHTMLNodes(t, cards)
	assert.Contains(t, body, "go.mod")
	assert.NotContains(t, body, "go.mod:0", "a file-level annotation shows no bogus line zero")
}

// A line RANGE anchors as file:start-end, so a multi-line selection reads as the
// span it actually covers.
func TestRenderAnnotationThreads_lineRangeAnchorsAsAspan(t *testing.T) {
	cards := renderAnnotationThreads(&ReviewCard{Key: "k"}, []annotationThread{{
		Root: ledger.AnnotationRecord{ID: "s", File: "a.go", StartLine: 10, EndLine: 14, Author: "lead", Body: "this block"},
	}})
	body := renderHTMLNodes(t, cards)
	assert.Contains(t, body, "a.go:10-14", "a range anchors to its span")
}

// Each thread card offers a reply affordance wired to ReplyToAnnotation and
// keyed to that card's root id, so a reply lands under the right parent. (The
// signal BINDING is verified through the real /review render, where the card's
// signals are framework-initialized; a bare struct here renders empty names.)
func TestRenderAnnotationThreads_offersAReplyFormKeyedToTheRootID(t *testing.T) {
	cards := renderAnnotationThreads(&ReviewCard{Key: "k"}, []annotationThread{{
		Root: ledger.AnnotationRecord{ID: "ann-7", File: "a.go", StartLine: 3, Author: "lead", Body: "why?"},
	}})
	body := renderHTMLNodes(t, cards)
	assert.Contains(t, body, "annotation-card__reply-form", "the card offers a reply affordance")
	assert.Contains(t, body, "ReplyToAnnotation", "the reply button posts the reply action")
	assert.Contains(t, body, "ann-7", "the reply is keyed to this card's root id")
}

func TestRenderAnnotationThreads_isEmptyForNoThreads(t *testing.T) {
	require.Empty(t, renderAnnotationThreads(&ReviewCard{Key: "k"}, nil))
}
