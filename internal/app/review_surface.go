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
		h.Class("review-editor-island"),
		h.DataIgnoreMorph(),
		// The structured payload the client editor reads (file sources + anchors).
		h.Script(h.Type("application/json"), h.ID("review-threads-data"), h.Raw(string(payload))),
		// The mount point Monaco renders into (sized in style.go).
		h.Div(h.Class("review-editor"), h.ID("review-editor")),
		// The Monaco AMD loader (pinned CDN) + the read-only bootstrap. Scoped to
		// /review (not AppendToHead) so the editor loads only on this surface. The
		// bootstrap is defensive — guards on the container/payload and try/catch — so
		// a load or parse failure leaves the server-rendered text threads above intact
		// (progressive enhancement, never a broken page).
		h.Script(h.Src(monacoLoaderURL)),
		h.Script(h.Raw(monacoBootstrapJS)),
	)
}

// monacoLoaderURL pins the Monaco editor AMD loader to a fixed CDN version
// (reproducible; not @latest). Vendoring it behind a /static handler is a later
// hardening slice — CDN-first gets the editor visible without new asset plumbing.
const monacoVersion = "0.52.2"
const monacoLoaderURL = "https://cdn.jsdelivr.net/npm/monaco-editor@" + monacoVersion + "/min/vs/loader.js"

// monacoBootstrapJS mounts a READ-ONLY Monaco editor over the #review-threads-data
// payload: it renders the first reviewed file whose source is present and decorates
// each surviving-mutant line with the "question:" body as a glyph-margin hover. It
// is deliberately defensive — every step guards and the JSON parse is wrapped — so
// any failure (loader blocked, parse error, no source) simply leaves the
// server-rendered text threads as the fallback. Read-only per council R62 (editable
// answering is a deferred, maintainer-gated fork).
const monacoBootstrapJS = `(function(){
  if (typeof require === 'undefined') return;
  require.config({ paths: { vs: 'https://cdn.jsdelivr.net/npm/monaco-editor@` + monacoVersion + `/min/vs' } });
  require(['vs/editor/editor.main'], function(){
    var el = document.getElementById('review-editor');
    var dataEl = document.getElementById('review-threads-data');
    if (!el || !dataEl) return;
    var island;
    try { island = JSON.parse(dataEl.textContent); } catch (e) { return; }
    var files = island.files || {};
    var threads = island.threads || [];
    var path = null;
    for (var i = 0; i < threads.length; i++) { if (files[threads[i].file] != null) { path = threads[i].file; break; } }
    if (!path) return;
    var model = monaco.editor.createModel(files[path], undefined, monaco.Uri.file(path));
    var editor = monaco.editor.create(el, { model: model, readOnly: true, automaticLayout: true, glyphMargin: true, theme: 'vs-dark', minimap: { enabled: false }, scrollBeyondLastLine: false });
    var decos = [];
    for (var j = 0; j < threads.length; j++) {
      var t = threads[j];
      if (t.file !== path) continue;
      decos.push({ range: new monaco.Range(t.line, 1, t.line, 1), options: { isWholeLine: true, className: 'review-survivor-line', glyphMarginClassName: 'review-survivor-glyph', glyphMarginHoverMessage: { value: t.tag + ': ' + t.body } } });
    }
    editor.createDecorationsCollection(decos);
  });
})();`

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
