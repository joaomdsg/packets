package app

import (
	"strconv"

	"github.com/go-via/via/h"
	"github.com/go-via/via/on"

	"github.com/joaomdsg/packets/internal/ledger"
)

// annotationThread is one durable annotation and the whole conversation under
// it: the root comment plus every reply, flattened chronologically. One thread
// reads as one exchange, never a fragmenting tree.
type annotationThread struct {
	Root    ledger.AnnotationRecord
	Replies []ledger.AnnotationRecord
}

// foldAnnotationThreads groups durable annotation records into threads: a
// top-level annotation (empty ParentID) is a root, and every record that chains
// up to that root — a reply, or a reply to a reply — flattens into its Replies
// in append (chronological) order. A record whose parent chain never reaches a
// known root stands as its own thread rather than being dropped, so a durable
// record is never silently lost. Roots keep the order they were written.
func foldAnnotationThreads(records []ledger.AnnotationRecord) []annotationThread {
	byID := make(map[string]ledger.AnnotationRecord, len(records))
	for _, r := range records {
		byID[r.ID] = r
	}

	// rootID walks the parent chain to the top-level ancestor's id. An unknown
	// parent (or a cycle) stops the walk at the last resolvable record, so an
	// orphan resolves to itself and becomes its own root.
	rootID := func(r ledger.AnnotationRecord) string {
		seen := map[string]bool{}
		for r.ParentID != "" && !seen[r.ID] {
			seen[r.ID] = true
			parent, ok := byID[r.ParentID]
			if !ok {
				break
			}
			r = parent
		}
		return r.ID
	}

	var threads []annotationThread
	idx := make(map[string]int) // root id -> its index in threads
	for _, r := range records {
		root := rootID(r)
		if root == r.ID {
			idx[r.ID] = len(threads)
			threads = append(threads, annotationThread{Root: r})
		}
	}
	for _, r := range records {
		root := rootID(r)
		if root == r.ID {
			continue // already placed as a root
		}
		if i, ok := idx[root]; ok {
			threads[i].Replies = append(threads[i].Replies, r)
		}
	}
	return threads
}

// annotationAnchor renders a record's location: the file alone for a file-level
// annotation (no line), "file:line" for a single line, or "file:start-end" for a
// range — never a misleading ":0".
func annotationAnchor(r ledger.AnnotationRecord) string {
	if r.StartLine == 0 {
		return r.File
	}
	if r.EndLine > r.StartLine {
		return r.File + ":" + strconv.Itoa(r.StartLine) + "-" + strconv.Itoa(r.EndLine)
	}
	return r.File + ":" + strconv.Itoa(r.StartLine)
}

// renderAnnotationThreads renders each durable annotation as a card carrying its
// real author, anchor, and body, with every reply nested beneath it in order —
// the threaded conversation the rail shows alongside the oracle findings. The
// author chip shows who actually wrote each line (human or agent), never a
// hardcoded label. Each card ends with a reply form wired to ReplyToAnnotation,
// keyed to that card's root id, so the Lead can answer in place.
func renderAnnotationThreads(c *ReviewCard, threads []annotationThread) []h.H {
	var out []h.H
	for _, th := range threads {
		parts := []h.H{
			h.Class("review-thread annotation-card"),
			h.Data("file", th.Root.File),
			h.Data("line", strconv.Itoa(th.Root.StartLine)),
			h.Div(h.Class("annotation-card__head"),
				h.Span(h.Class("annotation-card__chip annotation-card__chip--author"), h.Text(th.Root.Author)),
				h.Span(h.Class("review-thread__anchor annotation-card__where"), h.Text(annotationAnchor(th.Root))),
			),
			h.Span(h.Class("review-thread__body annotation-card__body"), h.Text(th.Root.Body)),
		}
		for _, rep := range th.Replies {
			parts = append(parts, h.Div(h.Class("annotation-card__reply"),
				h.Span(h.Class("annotation-card__chip annotation-card__chip--author"), h.Text(rep.Author)),
				h.Span(h.Class("annotation-card__body"), h.Text(rep.Body)),
			))
		}
		// The reply affordance: the button sets replyparent to THIS card's root id,
		// then posts — the shared per-tab reply signals carry the right target
		// regardless of which card is answered.
		parts = append(parts, h.Div(h.Class("annotation-card__reply-form"),
			h.Input(h.Type("text"), c.ReplyText.Bind(), h.Class("pk-input annotation-card__reply-input"), h.Placeholder("reply…")),
			h.Button(
				on.Click(c.ReplyToAnnotation, on.SetSignal(&c.ReplyParent.Signal, th.Root.ID)),
				h.Class("pk-btn annotation-card__reply-btn"),
				h.Text("reply"),
			),
		))
		out = append(out, h.Div(parts...))
	}
	return out
}
