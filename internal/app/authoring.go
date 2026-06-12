package app

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/go-via/via"
	"github.com/go-via/via/h"

	"github.com/joaomdsg/packets/internal/assist"
)

// draftAnalysis is the cached authoring-assist read of one draft: the exact text
// analyzed (so the editor decorates against the same bytes the offsets index), the
// producer's structured result, and a degrade reason when the run failed or its
// output was unreadable (Result nil in that case).
type draftAnalysis struct {
	Draft  string
	Result *assist.Analysis
	Reason string
}

// analyzeDraft is the seam the authoring assist runs through: it spawns a producer
// harness on the analysis prompt and returns its RAW stdout for ParseAnalysis.
// Default shells claude (process I/O — verified by build + manual run, not
// unit-tested, like RunProcess); tests swap it for a scripted reply.
var analyzeDraft = runAnalysisProcess

// analysisArgs is the claude argv for the authoring assist. It runs HAIKU at LOW
// effort because the assist reads the draft as the Lead types, so the read must be
// fast — Haiku alone still reasons at full effort (~40s observed); --effort low cuts
// the thinking so the read returns quickly. Plain one-shot text output (not a settled
// stream): the assist wants the agent's reply, never to touch the tree or the economy.
func analysisArgs(prompt, resumeID string) []string {
	args := []string{"-p", prompt, "--output-format", "text", "--model", "haiku", "--effort", "low"}
	if resumeID != "" {
		// Resume the session's WARM explored harness, forking a branch — so the read
		// reuses the repo context the warm-up built, without colliding with concurrent
		// reads or an order fill on the one base id.
		args = append(args, "--resume", resumeID, "--fork-session")
	}
	return args
}

// runAnalysisProcess runs claude headless on prompt in repoDir and returns its
// stdout text. Unlike the order harness (which reduces a stream into settled
// revisions), the authoring assist wants the agent's one-shot textual reply, so it
// runs in plain text output and never settles anything — analyzing a draft must
// touch neither the working tree nor the economy. resumeID, when set, resumes the
// session's warm harness so the read carries the explored repo context.
func runAnalysisProcess(ctx context.Context, repoDir, prompt, resumeID string) (string, error) {
	cmd := exec.CommandContext(ctx, "claude", analysisArgs(prompt, resumeID)...)
	cmd.Dir = repoDir
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("authoring: run analysis: %v", err)
	}
	return string(out), nil
}

// AnalyzeDraft runs a producer over the draft the Lead is authoring (the OrderPrompt
// the compose textarea binds) and caches its structured read — the summary,
// readiness verdict, flagged spans, and clarifying questions — so the card renders
// it. An empty draft is a silent no-op (nothing to analyze, no producer spawned). A
// failed run or unreadable output degrades to a calm "analysis unavailable" cache,
// never a broken card — the Lead can still place the order. FIREWALL: it writes only
// the off-economy analysis cache, never the ledger — analyzing mints nothing.
func (c *LiveCard) AnalyzeDraft(ctx *via.Ctx) {
	cfg, log := readLiveState(c.Key)
	if log == nil {
		return
	}
	e := lookupLiveEntry(c.Key)
	if e == nil {
		return
	}
	draft := strings.TrimSpace(c.OrderPrompt.Read(ctx))
	if draft == "" {
		return // nothing to analyze
	}
	// The assist auto-triggers on caret movement (past a blank line), which fires even
	// when the text is unchanged — re-running the producer on a draft already
	// successfully analyzed can only reproduce the cached read, so it is a no-op. A
	// prior FAILED read (Result nil) is NOT skipped, so a transient failure can retry.
	if prev := e.analysisSnapshot(); prev != nil && prev.Result != nil && prev.Draft == draft {
		return
	}
	// Cancel any prior in-flight read and run under a fresh context, so a fast-typing
	// Lead's superseded analyses are abandoned, never left racing this one.
	runCtx := e.beginAnalysis()
	raw, err := analyzeDraft(runCtx, cfg.RepoDir, assist.AnalysisPrompt(draft), e.resumeSessionID())
	if runCtx.Err() != nil {
		return // superseded by a newer analyze — let that one own the cache
	}
	if err != nil {
		e.setAnalysis(&draftAnalysis{Draft: draft, Reason: "the producer run failed — try again"})
		c.Analysis.Write(ctx, "err")
		return
	}
	a, err := assist.ParseAnalysis(raw, draft)
	if err != nil {
		e.setAnalysis(&draftAnalysis{Draft: draft, Reason: "the producer's output was unreadable — try again"})
		c.Analysis.Write(ctx, "err")
		return
	}
	e.setAnalysis(&draftAnalysis{Draft: draft, Result: &a})
	c.Analysis.Write(ctx, "ok")
}

// renderAuthoring is the authoring-assist surface: an editable Monaco editor as the
// single draft source, with the producer's structured read (summary + clarifying
// questions) beneath it. The producer's flagged spans are decorated INLINE in the
// editor itself (not a separate mirror), and the readiness verdict reflects beside
// place. da is the latest cached analysis (nil before the first run).
func renderAuthoring(c *LiveCard) h.H {
	var da *draftAnalysis
	if e := lookupLiveEntry(c.Key); e != nil {
		da = e.analysisSnapshot()
	}
	parts := []h.H{h.Class("authoring"), composeSurface(da)}
	if p := renderAnalysisPanel(da); p != nil {
		parts = append(parts, p)
	}
	return h.Div(parts...)
}

// composeSurface is the editable Monaco composer. The PERSISTENT interactive subtree
// (.compose__live, data-ignore-morph) holds the editor + buttons + indicator so its
// DOM, the Lead's text, and the JS listeners survive every SSE re-render; the editor
// is the single draft source. The buttons dispatch CustomEvents the wrapper's
// data-on bridge lifts into $orderprompt before @posting the action (the maplibre /
// answer-form pattern that works without data-bind and survives morphs). The
// re-rendering bits (readiness, the highlights payload the editor decorates from) sit
// OUTSIDE the shield so a fresh analysis updates them in place.
func composeSurface(da *draftAnalysis) h.H {
	live := h.Div(
		h.Class("compose__live"),
		h.DataIgnoreMorph(),
		h.Attr("aria-label", "author a live order"),
		// The bridge: each button's CustomEvent carries the editor's value, which the
		// handler assigns to $orderprompt INLINE (so the signal is present at post time)
		// then @posts the action AnalyzeDraft/PlaceOrder reads.
		h.Data("on:viaanalyze", "$orderprompt=evt.detail.draft;@post('/_action/AnalyzeDraft')"),
		h.Data("on:viaplace", "$orderprompt=evt.detail.draft;@post('/_action/PlaceOrder')"),
		h.Div(h.ID("authoring-editor"), h.Class("compose__editor")),
		h.Button(h.Type("button"), h.Class("pk-btn pk-btn--quiet compose__analyze"), h.Text("Analyze draft")),
		h.Button(h.Type("button"), h.Class("pk-btn compose__place"), h.Text("Place order")),
		h.Span(h.Class("compose__analyzing"), h.Data("state", "idle"), h.Text("analyzing…")),
		h.Script(h.Src(monacoLoaderURL)),
		h.Script(h.Raw(authoringEditorJS)),
	)
	parts := []h.H{h.Class("compose"), live}
	// Once the producer has read the draft, reflect its readiness beside place — a
	// guide, never a gate (placing stays allowed at any readiness). Outside the shield
	// so a fresh verdict re-renders in place.
	if da != nil && da.Result != nil {
		state, note := "caution", "The producer flagged open questions — placing will run the draft as-is."
		if da.Result.Ready {
			state, note = "ready", "The producer judged this ready to run unattended."
		}
		parts = append(parts, h.Span(h.Class("compose__readiness"), h.Data("state", state), h.Text(note)))
	}
	// The highlights payload the editor decorates from. Outside the shield so each
	// fresh analysis replaces it; the editor's MutationObserver reapplies the
	// decorations in place. Empty before the first run (no spans to anchor).
	var hl []assist.Highlight
	if da != nil && da.Result != nil {
		hl = da.Result.Highlights
	}
	payload, _ := json.Marshal(struct {
		Highlights []assist.Highlight `json:"highlights"`
	}{Highlights: hl})
	parts = append(parts, h.Script(h.Type("application/json"), h.ID("authoring-analysis-data"), h.Raw(string(payload))))
	if tokenStore == nil || !tokenStore.Configured() {
		parts = append(parts, h.Div(
			h.Class("compose__needs-key"),
			h.Text("No Anthropic API key configured — "),
			h.A(h.Href("/settings"), h.Class("compose__needs-key-link"), h.Text("set one in settings")),
			h.Text(" to run live orders."),
		))
	}
	return h.Div(parts...)
}

// renderAnalysisPanel renders the producer's structured read beneath the editor: a
// calm unavailable note when the run failed, otherwise the summary + the clarifying
// questions to answer before re-analyzing. The flagged spans are decorated in the
// editor itself; the readiness reflects beside place — so this panel is the prose,
// not a second copy of the draft. Returns nil when there is no analysis yet.
func renderAnalysisPanel(da *draftAnalysis) h.H {
	if da == nil {
		return nil
	}
	if da.Result == nil {
		return h.Div(
			h.Class("pk-card analysis"),
			h.Data("state", "unavailable"),
			h.Span(h.Class("analysis__unavailable"), h.Text("Analysis unavailable — "+da.Reason+".")),
		)
	}
	a := da.Result
	state := "blocked"
	if a.Ready {
		state = "ready"
	}
	parts := []h.H{
		h.Class("pk-card analysis"),
		h.Data("state", state),
		h.Attr("aria-label", "draft analysis"),
		h.Span(h.Class("analysis__summary"), h.Text(a.Summary)),
	}
	if len(a.Questions) > 0 {
		qs := []h.H{h.Class("analysis__questions")}
		for _, q := range a.Questions {
			qs = append(qs, h.Li(h.Class("analysis__question"), h.Text(q)))
		}
		parts = append(parts,
			h.Span(h.Class("analysis__questions-label"), h.Text("Answer these, then re-analyze:")),
			h.Ul(qs...))
	}
	return h.Div(parts...)
}

// authoringEditorJS mounts the EDITABLE Monaco editor (the single draft source),
// wires the buttons + a debounced live re-analysis to the CustomEvent bridge, and
// decorates the flagged spans INLINE — reapplying them whenever a fresh analysis
// payload arrives (a MutationObserver on the out-of-shield payload element). A
// dataset guard keeps the mount idempotent across re-renders; require is loaded by
// this surface's loader. Offsets map to positions via model.getPositionAt (exact for
// ASCII drafts, the common case). Progressive enhancement: a loader/parse failure
// leaves the server-rendered summary + questions intact.
const authoringEditorJS = `(function(){
  if (typeof require === 'undefined') return;
  require.config({ paths: { vs: 'https://cdn.jsdelivr.net/npm/monaco-editor@` + monacoVersion + `/min/vs' } });
  require(['vs/editor/editor.main'], function(){
    var el = document.getElementById('authoring-editor');
    if (!el || el.dataset.mounted) return;
    el.dataset.mounted = '1';
    var live = el.closest('.compose__live');
    var ed = monaco.editor.create(el, { value: '', language: 'markdown', readOnly: false, automaticLayout: true, theme: 'vs-dark', wordWrap: 'on', minimap: { enabled: false }, scrollBeyondLastLine: false, lineNumbers: 'off' });
    var col = ed.createDecorationsCollection([]);
    var ind = live ? live.querySelector('.compose__analyzing') : null;
    function applyDecos(){
      var dataEl = document.getElementById('authoring-analysis-data');
      if (!dataEl) return;
      var hs = [];
      try { hs = (JSON.parse(dataEl.textContent) || {}).highlights || []; } catch (e) { return; }
      var model = ed.getModel(), decos = [];
      for (var i = 0; i < hs.length; i++) {
        var s = model.getPositionAt(hs[i].start), e = model.getPositionAt(hs[i].end);
        decos.push({ range: new monaco.Range(s.lineNumber, s.column, e.lineNumber, e.column), options: { inlineClassName: 'authoring-flag-' + (hs[i].severity || 'note'), hoverMessage: { value: hs[i].note } } });
      }
      col.set(decos);
    }
    var lastAnalyzed = null;
    function analyze(){ if (live) { lastAnalyzed = ed.getValue(); live.dispatchEvent(new CustomEvent('viaanalyze', { detail: { draft: lastAnalyzed } })); } }
    function place(){ if (live) live.dispatchEvent(new CustomEvent('viaplace', { detail: { draft: ed.getValue() } })); }
    var aBtn = live ? live.querySelector('.compose__analyze') : null;
    var pBtn = live ? live.querySelector('.compose__place') : null;
    if (aBtn) aBtn.addEventListener('click', analyze);
    if (pBtn) pBtn.addEventListener('click', place);
    // Trigger the assist SPARINGLY: only when the caret moves DOWN past a blank
    // (paragraph) line, or lands on a blank line just after content (a finished
    // block) — never on every keystroke. Fast movement keeps clearing the pending
    // timer, so only a settled pause fires; the server cancels any superseded
    // in-flight run. A short 350ms settle keeps it responsive (the read runs Haiku).
    var lastLine = 1, timer;
    function blankLine(n){ return ed.getModel().getLineContent(n).trim() === ''; }
    ed.onDidChangeCursorPosition(function(e){
      var line = e.position.lineNumber, model = ed.getModel(), fire = false;
      if (line > lastLine) {
        for (var i = lastLine; i < line; i++) { if (model.getLineContent(i).trim() === '') { fire = true; break; } }
      }
      if (!fire && blankLine(line) && line > 1 && !blankLine(line - 1)) fire = true;
      if (!fire) return;
      clearTimeout(timer); // moving fast keeps cancelling the pending trigger
      if (ind) ind.dataset.state = 'pending';
      timer = setTimeout(function(){
        lastLine = line;
        // Skip when the draft is unchanged since the last analysis — the auto-trigger
        // fires on caret movement, so this guards the common "moved, didn't edit" case
        // (no round-trip, no stuck 'analyzing' indicator). The explicit Analyze button
        // bypasses this — it calls analyze() directly.
        if (ed.getValue() === lastAnalyzed) { if (ind) ind.dataset.state = 'idle'; return; }
        if (ind) ind.dataset.state = 'analyzing';
        analyze();
      }, 350);
    });
    var dataEl = document.getElementById('authoring-analysis-data');
    if (dataEl && window.MutationObserver) {
      new MutationObserver(function(){ applyDecos(); if (ind) ind.dataset.state = 'idle'; }).observe(dataEl, { childList: true, characterData: true, subtree: true });
    }
    applyDecos();
    ed.focus();
  });
})();`
