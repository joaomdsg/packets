package app

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/go-via/via"
	"github.com/go-via/via/h"

	"github.com/joaomdsg/packets/internal/reanchor"
	"github.com/joaomdsg/packets/internal/review"
)

// reviewFileReader reads a reviewed file's source at a revision — the I/O seam the
// editor island uses to embed the file the questions are anchored to. A package var
// so tests inject canned source (the git-show boundary is real subprocess I/O).
var reviewFileReader = reanchor.FileAt

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
	cfg, _ := readLiveState(navKey)
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
	// The editor island: a DOM subtree the client-side Monaco review editor (a later
	// slice) mounts into, plus the SAME threads as a machine-readable JSON payload so
	// the editor reads structured data, not the human text above. data-ignore-morph
	// shields the editor's own DOM from being clobbered by an SSE re-render. Emitted
	// only when there ARE questions — nothing to scaffold over an empty set.
	parts = append(parts, reviewEditorIsland(cfg, threads))
	return h.Div(parts...)
}

// reviewEditorIsland renders the Monaco mount point + a JSON payload the editor
// reads: the reviewed file sources keyed by path, plus the anchored threads — all
// from the SAME projection as the server text (one source, no drift).
// encoding/json HTML-escapes <, >, & by default, so file source or an oracle
// message containing "</script>" can't break out of the script element.
func reviewEditorIsland(cfg LiveConfig, threads []review.Thread) h.H {
	payload, _ := json.Marshal(reviewIslandData(cfg, threads))
	return h.Div(
		h.Class("review-editor"),
		h.ID("review-editor"),
		h.DataIgnoreMorph(),
		h.Script(h.Type("application/json"), h.ID("review-threads-data"), h.Raw(string(payload))),
	)
}

// reviewIsland is the contract the client editor consumes: the reviewed file
// sources (so it can render the file the questions are anchored to) keyed by path,
// plus the per-thread anchors.
type reviewIsland struct {
	Files   map[string]string     `json:"files"`
	Threads []reviewThreadPayload `json:"threads"`
}

// reviewThreadPayload is the per-thread shape the client editor consumes: the
// minimum to anchor a decoration (file, line) and show the question (tag, body).
type reviewThreadPayload struct {
	File string `json:"file"`
	Line int    `json:"line"`
	Tag  string `json:"tag"`
	Body string `json:"body"`
}

// reviewIslandData assembles the editor payload. It reads each referenced file's
// source ONCE at the reviewed (fix) revision through reviewFileReader; a file whose
// source can't be read (lost anchor, deleted file, transient git error) is OMITTED
// from Files rather than emitted as an empty-string lie — the editor still gets the
// threads and degrades to anchoring against no source.
func reviewIslandData(cfg LiveConfig, threads []review.Thread) reviewIsland {
	files := map[string]string{}
	attempted := map[string]bool{}
	for _, t := range threads {
		if attempted[t.File] {
			continue
		}
		attempted[t.File] = true
		if src, err := reviewFileReader(context.Background(), cfg.RepoDir, cfg.FixRev, t.File); err == nil {
			files[t.File] = src
		}
	}
	return reviewIsland{Files: files, Threads: reviewThreadPayloads(threads)}
}

func reviewThreadPayloads(threads []review.Thread) []reviewThreadPayload {
	out := make([]reviewThreadPayload, 0, len(threads))
	for _, t := range threads {
		out = append(out, reviewThreadPayload{File: t.File, Line: t.StartLine, Tag: t.Tag, Body: t.Body})
	}
	return out
}
