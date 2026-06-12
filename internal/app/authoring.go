package app

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/go-via/via"
	"github.com/go-via/via/h"
	"github.com/go-via/via/on"

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

// runAnalysisProcess runs claude headless on prompt in repoDir and returns its
// stdout text. Unlike the order harness (which reduces a stream into settled
// revisions), the authoring assist wants the agent's one-shot textual reply, so it
// runs in plain text output and never settles anything — analyzing a draft must
// touch neither the working tree nor the economy.
func runAnalysisProcess(ctx context.Context, repoDir, prompt string) (string, error) {
	cmd := exec.CommandContext(ctx, "claude", "-p", prompt, "--output-format", "text")
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
	raw, err := analyzeDraft(context.Background(), cfg.RepoDir, assist.AnalysisPrompt(draft))
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

// renderAnalysis renders the producer's structured read beneath the compose control:
// a calm unavailable note when the run failed, otherwise the summary, a readiness
// hook (ready|blocked, colored in the palette), the clarifying questions, and the
// Monaco authoring island that decorates the analyzed draft with the flagged spans.
// Returns nil when there is no analysis yet (nothing to show).
func renderAnalysis(da *draftAnalysis) h.H {
	if da == nil {
		return nil
	}
	if da.Result == nil {
		return h.Div(
			h.Class("analysis"),
			h.Data("state", "unavailable"),
			h.Span(h.Class("analysis__unavailable"), h.Text("Analysis unavailable — "+da.Reason+".")),
		)
	}
	a := da.Result
	state := "blocked"
	readiness := "Not yet ready to run unattended — sharpen the draft below."
	if a.Ready {
		state, readiness = "ready", "Ready to run unattended."
	}
	parts := []h.H{
		h.Class("analysis"),
		h.Data("state", state),
		h.Attr("aria-label", "draft analysis"),
		h.Span(h.Class("analysis__summary"), h.Text(a.Summary)),
		h.Span(h.Class("analysis__readiness"), h.Data("state", state), h.Text(readiness)),
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
	parts = append(parts, authoringIsland(da))
	return h.Div(parts...)
}

// authoringIsland renders the Monaco mount point + a JSON payload the editor reads:
// the analyzed draft and the flagged spans, so the editor anchors a decoration on
// exactly the bytes the producer flagged (offsets are draft byte offsets; the
// bootstrap maps them to positions via model.getPositionAt — exact for ASCII drafts,
// the common case). The payload is the SAME analysis the server text above renders —
// one source, no drift. Progressive enhancement: a loader/parse failure leaves the
// server-rendered summary + questions intact.
func authoringIsland(da *draftAnalysis) h.H {
	payload, _ := json.Marshal(struct {
		Draft      string             `json:"draft"`
		Highlights []assist.Highlight `json:"highlights"`
	}{Draft: da.Draft, Highlights: da.Result.Highlights})
	return h.Div(
		h.Class("authoring-island"),
		h.DataIgnoreMorph(),
		h.Script(h.Type("application/json"), h.ID("authoring-analysis-data"), h.Raw(string(payload))),
		h.Div(h.ID("authoring-editor"), h.Class("authoring-editor")),
		h.Script(h.Src(monacoLoaderURL)),
		h.Script(h.Raw(authoringBootstrapJS)),
	)
}

// authoringBootstrapJS mounts a read-only Monaco editor over the analyzed draft and
// decorates each flagged span with its note as a hover. Defensive (guards +
// try/catch); require is loaded by this island's loader.
const authoringBootstrapJS = `(function(){
  if (typeof require === 'undefined') return;
  require.config({ paths: { vs: 'https://cdn.jsdelivr.net/npm/monaco-editor@` + monacoVersion + `/min/vs' } });
  require(['vs/editor/editor.main'], function(){
    var el = document.getElementById('authoring-editor');
    var dataEl = document.getElementById('authoring-analysis-data');
    if (!el || !dataEl || el.dataset.mounted) return;
    el.dataset.mounted = '1';
    var d;
    try { d = JSON.parse(dataEl.textContent); } catch (e) { return; }
    var model = monaco.editor.createModel(d.draft || '', 'markdown');
    var editor = monaco.editor.create(el, { model: model, readOnly: true, automaticLayout: true, theme: 'vs-dark', wordWrap: 'on', minimap: { enabled: false }, scrollBeyondLastLine: false, lineNumbers: 'off' });
    var hs = d.highlights || [], decos = [];
    for (var i = 0; i < hs.length; i++) {
      var s = model.getPositionAt(hs[i].start), e = model.getPositionAt(hs[i].end);
      decos.push({ range: new monaco.Range(s.lineNumber, s.column, e.lineNumber, e.column), options: { inlineClassName: 'authoring-flag-' + (hs[i].severity || 'note'), hoverMessage: { value: hs[i].note } } });
    }
    editor.createDecorationsCollection(decos);
  });
})();`

// renderAuthoring renders the compose control plus, beneath it, the producer's
// structured analysis when one is cached — the authoring assist surface. The compose
// control carries an "Analyze draft" button (on.Click AnalyzeDraft reads the bound
// OrderPrompt) so the Lead can author, analyze, sharpen, and place in one place.
func renderAuthoring(c *LiveCard) h.H {
	var da *draftAnalysis
	if e := lookupLiveEntry(c.Key); e != nil {
		da = e.analysisSnapshot()
	}
	parts := []h.H{h.Class("authoring"), renderComposeWithAnalyze(c, da)}
	if a := renderAnalysis(da); a != nil {
		parts = append(parts, a)
	}
	return h.Div(parts...)
}

// renderComposeWithAnalyze is the order composer: the draft textarea, the analyze +
// place buttons (both reading the same authored OrderPrompt), the live debounced
// re-analysis wiring (the producer listens as the Lead types), and — once an
// analysis is cached — a readiness reflection beside place. da is the latest cached
// analysis (nil before the first run).
func renderComposeWithAnalyze(c *LiveCard, da *draftAnalysis) h.H {
	parts := []h.H{
		h.Class("compose"),
		h.Attr("aria-label", "author a live order"),
		h.Textarea(c.OrderPrompt.Bind(), h.Class("compose__prompt"),
			h.Placeholder("Describe the task for a live order…")),
		h.Button(on.Click(c.AnalyzeDraft), h.Class("compose__analyze"), h.Text("Analyze draft")),
		h.Button(on.Click(c.PlaceOrder), h.Class("compose__place"), h.Text("Place order")),
		// The live read indicator (driven by the debounce script) and the script that
		// re-analyzes on a typing pause by clicking the proven analyze action — so the
		// producer keeps pace with the draft without a new server seam. Progressive
		// enhancement: the manual button works with JS off.
		h.Span(h.Class("compose__analyzing"), h.Data("state", "idle"), h.Text("analyzing…")),
		h.Script(h.Raw(liveAnalyzeJS)),
	}
	// Once the producer has read the draft, reflect its readiness verdict beside
	// place — a guide, never a gate (placing stays allowed at any readiness).
	if da != nil && da.Result != nil {
		state, note := "caution", "The producer flagged open questions — placing will run the draft as-is."
		if da.Result.Ready {
			state, note = "ready", "The producer judged this ready to run unattended."
		}
		parts = append(parts, h.Span(h.Class("compose__readiness"), h.Data("state", state), h.Text(note)))
	}
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

// liveAnalyzeJS makes the producer listen as the Lead types: on a pause in input
// (debounced), it triggers the proven analyze action by clicking its button, and
// flips the analyzing indicator while the read is pending — so the analysis keeps
// pace with the draft without a second server seam. A dataset guard prevents
// re-wiring across SSE re-renders; with JS off the manual button still analyzes.
const liveAnalyzeJS = `(function(){
  var t = document.querySelector('.compose__prompt');
  var b = document.querySelector('.compose__analyze');
  var ind = document.querySelector('.compose__analyzing');
  if (!t || !b || t.dataset.liveWired) return;
  t.dataset.liveWired = '1';
  var timer;
  t.addEventListener('input', function(){
    clearTimeout(timer);
    if (ind) ind.dataset.state = 'pending';
    timer = setTimeout(function(){ if (ind) ind.dataset.state = 'analyzing'; b.click(); }, 900);
  });
})();`
