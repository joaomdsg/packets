package app

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/go-via/via"
	"github.com/go-via/via/h"

	"github.com/joaomdsg/packets/internal/pipe"
	"github.com/joaomdsg/packets/internal/reanchor"
	"github.com/joaomdsg/packets/internal/review"
)

// reviewFileReader reads a reviewed file's source at a revision — the I/O seam the
// editor island uses to embed the file the questions are anchored to. A package var
// so tests inject canned source (the git-show boundary is real subprocess I/O).
var reviewFileReader = reanchor.FileAt

// rerunWithOverlay re-runs the oracle at the fix rev with the reviewer's test
// injected — the seam behind answering a question. A package var so tests inject a
// canned verdict (the real one is a full oracle run over git worktrees).
var rerunWithOverlay = pipe.RerunWithTestOverlay

// answerTestFilename is where a reviewer's submitted test is written in the worktree
// — a fixed _test.go beside the reviewed file, so `go test ./...` compiles it into
// that package. Its content is the reviewer's submission (which declares its own
// package), so the name only has to be a unique _test.go in the right directory.
const answerTestFilename = "packets_review_answer_test.go"

// ReviewCard is the dedicated review surface (/review?key=<session>): the full
// anchored "question:" threads the card's badge only counts. Each thread is a
// surviving/undetermined mutant the fix oracle found — an honest test gap the green
// verdict hides. The threads are the session's latest connect-cycle findings
// (recomputed each cycle, off the economy ledger). A reviewer ANSWERS a question by
// submitting a test for its line (AnswerQuestion): the oracle re-runs with that test
// injected, and if the mutant dies the question vanishes — diagnostic only, never a
// scored transaction.
type ReviewCard struct {
	Key string `query:"key"`
	// The reviewer's answer submission, set by the editor before posting AnswerQuestion.
	AnswerFile via.SignalStr `via:"answerfile"`
	AnswerLine via.SignalStr `via:"answerline"`
	AnswerTest via.SignalStr `via:"answertest"`
}

// AnswerQuestion re-runs the oracle for the answered line with the reviewer's test
// injected into a throwaway worktree, and replaces the session's cached findings
// with the result: a test that KILLS the mutant leaves no finding (the question
// vanishes on the next render); a weak one leaves the survivor (the question stays
// open). FIREWALL: it writes ONLY the off-economy findings cache — never the ledger,
// never balance — so answering mints nothing (the vanishing question is the reward).
// A blank/invalid submission or a transient re-run error is a no-op: the question
// stays open, retryable (an Undetermined/failed run never falsely clears it).
func (c *ReviewCard) AnswerQuestion(ctx *via.Ctx) {
	key := c.Key
	e := lookupLiveEntry(key)
	if e == nil {
		return
	}
	file := c.AnswerFile.Read(ctx)
	test := c.AnswerTest.Read(ctx)
	line, err := strconv.Atoi(c.AnswerLine.Read(ctx))
	if file == "" || test == "" || err != nil || line < 1 {
		return // nothing to answer
	}
	cfg, _ := readLiveState(key)
	overlay := map[string]string{filepath.Join(filepath.Dir(file), answerTestFilename): test}
	newFindings, err := rerunWithOverlay(context.Background(), cfg.RepoDir, cfg.FixRev, file, line, cfg.TestCmd, overlay)
	if err != nil {
		return // transient — leave the question open, retryable (flaky-truth fence)
	}
	// Wholesale replace is correct because the live card anchors ONE line and
	// mutation.Run is scoped to it, so the cache only ever holds that line's findings.
	// (If multi-line answering is ever added, replace only the answered line's entries.)
	e.setFindings(newFindings) // diagnostic cache only; no ledger touch — FIREWALL
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
	// The answer affordance: write the killing test in an editable Monaco pane and
	// submit. Following the maplibre plugin's client→server pattern (the only one
	// that survives morphs cleanly): the editor + submit live in ONE data-ignore-morph
	// wrapper, the submit dispatches a CustomEvent carrying the editor's value, and the
	// wrapper's data-on:viaanswer ASSIGNS the answer signals from evt.detail INLINE,
	// then @posts AnswerQuestion. Assigning in the datastar expression (not data-bind)
	// is what makes the signal reliably present at post time. AnswerQuestion re-runs
	// the oracle with the test injected — a kill makes the question vanish
	// (diagnostic, off-economy).
	anchor := threads[0]
	parts = append(parts, h.Div(
		h.Class("review-answer"),
		h.P(h.Class("review-answer__label"),
			h.Text("Answer: write a test that kills the mutant on "+anchor.File+":"+strconv.Itoa(anchor.StartLine)+" (⌘/Ctrl+Enter to submit)")),
		h.Div(
			h.Class("review-answer__input"),
			h.DataIgnoreMorph(),
			// datastar catches the editor's submit CustomEvent, lifts its detail into
			// the answer signals, and posts the action — the maplibre-proven bridge.
			h.Data("on:viaanswer", "$answerfile=evt.detail.file;$answerline=evt.detail.line;$answertest=evt.detail.test;@post('/_action/AnswerQuestion')"),
			// while that post is in flight (the oracle re-run takes seconds), the
			// "answering" signal is true so the running line below reveals itself.
			h.Attr("data-indicator", "answering"),
			h.Div(h.ID("answer-editor"), h.Class("review-answer__editor")),
			h.Button(h.Type("button"), h.Class("review-answer__submit"),
				h.Text("Submit answer — re-run the oracle")),
			h.Script(h.Raw(answerEditorJS(anchor.File, anchor.StartLine))),
		),
		// Shown ONLY while the re-run is in flight (data-show on the indicator signal):
		// a calm status, not dead-air, for the seconds the oracle takes.
		h.Div(
			h.Attr("data-show", "$answering"),
			h.Class("review-answer__running"),
			h.Text("re-running the oracle — checking if your test kills the mutant…"),
		),
	))
	return h.Div(parts...)
}

// answerEditorJS mounts an EDITABLE Monaco editor (Go, vs-dark) into #answer-editor
// and wires its submit: a button click OR ⌘/Ctrl+Enter dispatches a "viaanswer"
// CustomEvent carrying {file, line, test:<editor content>} on the wrapper, where the
// data-on:viaanswer handler lifts it into signals and posts. This mirrors the
// maplibre plugin (client lib → CustomEvent → datastar expr → @post), the bridge
// that works without data-bind and survives morphs. require is loaded by the
// read-only island's loader; a dataset guard prevents a double-mount.
func answerEditorJS(file string, line int) string {
	detail := fmt.Sprintf("{file:%s,line:%s,test:ed.getValue()}", strconv.Quote(file), strconv.Quote(strconv.Itoa(line)))
	return `(function(){
  if (typeof require === 'undefined') return;
  require(['vs/editor/editor.main'], function(){
    var el = document.getElementById('answer-editor');
    if (!el || el.dataset.mounted) return;
    el.dataset.mounted = '1';
    var wrap = el.closest('.review-answer__input');
    var ed = monaco.editor.create(el, { value: '', language: 'go', readOnly: false, automaticLayout: true, theme: 'vs-dark', minimap: { enabled: false }, scrollBeyondLastLine: false, lineNumbers: 'on' });
    var submit = function(){ if (wrap) wrap.dispatchEvent(new CustomEvent('viaanswer', { detail: ` + detail + ` })); };
    var btn = wrap ? wrap.querySelector('.review-answer__submit') : null;
    if (btn) btn.addEventListener('click', submit);
    ed.addCommand(monaco.KeyMod.CtrlCmd | monaco.KeyCode.Enter, submit);
    ed.focus();
  });
})();`
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
    // The live card anchors ONE file, so every finding is in that file (mutation.Run
    // mutates a single file). Render it and decorate each surviving-mutant line.
    var path = null;
    for (var i = 0; i < threads.length; i++) { if (files[threads[i].file] != null) { path = threads[i].file; break; } }
    if (!path) return;
    var model = monaco.editor.createModel(files[path], undefined, monaco.Uri.file(path));
    var editor = monaco.editor.create(el, { model: model, readOnly: true, automaticLayout: true, glyphMargin: true, theme: 'vs-dark', minimap: { enabled: false }, scrollBeyondLastLine: false });
    var decos = [], firstLine = null;
    for (var j = 0; j < threads.length; j++) {
      var t = threads[j];
      if (t.file !== path) continue;
      if (firstLine === null) firstLine = t.line;
      decos.push({ range: new monaco.Range(t.line, 1, t.line, 1), options: { isWholeLine: true, className: 'review-survivor-line', glyphMarginClassName: 'review-survivor-glyph', glyphMarginHoverMessage: { value: t.tag + ': ' + t.body } } });
    }
    editor.createDecorationsCollection(decos);
    if (firstLine) editor.revealLineInCenter(firstLine); // open ON the first question, not line 1
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
