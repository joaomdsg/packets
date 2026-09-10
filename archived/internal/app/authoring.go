package app

import (
	"context"
	"encoding/json"
	"fmt"
	stdlog "log"
	"os/exec"
	"strconv"
	"strings"

	"github.com/go-via/via"
	"github.com/go-via/via/h"
	"github.com/go-via/via/on"

	"github.com/joaomdsg/packets/internal/assist"
	"github.com/joaomdsg/packets/internal/packet"
)

// draftAnalysis is the cached authoring-assist read of one draft: the exact text
// analyzed (so the editor decorates against the same bytes the offsets index), the
// assist's structured result, and a degrade reason when the run failed or its
// output was unreadable (Result nil in that case).
type draftAnalysis struct {
	Draft  string
	Result *assist.Analysis
	Reason string
}

// analyzeDraft is the seam the authoring assist runs through: it spawns a assist
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
		// reads or a packet fill on the one base id.
		args = append(args, "--resume", resumeID, "--fork-session")
	}
	return args
}

// runAnalysisProcess runs claude headless on prompt in repoDir and returns its
// stdout text. Unlike the packet harness (which reduces a stream into settled
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

// AnalyzeDraft runs a assist over the draft the Lead is authoring (the Draft
// the compose textarea binds) and caches its structured read — the summary,
// readiness verdict, flagged spans, and clarifying questions — so the card renders
// it. An empty draft is a silent no-op (nothing to analyze, no assist spawned). A
// failed run or unreadable output degrades to a calm "analysis unavailable" cache,
// never a broken card — the Lead can still place the packet. FIREWALL: it writes only
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
	draft := strings.TrimSpace(c.Draft.Read(ctx))
	if draft == "" {
		return // nothing to analyze
	}
	// The assist auto-triggers on caret movement (past a blank line), which fires even
	// when the text is unchanged — re-running the assist on a draft already
	// successfully analyzed can only reproduce the cached read, so it is a no-op. A
	// prior FAILED read (Result nil) is NOT skipped, so a transient failure can retry.
	if prev := e.analysisSnapshot(); prev != nil && prev.Result != nil && prev.Draft == draft {
		return
	}
	// Cancel any prior in-flight read and run under a fresh context, so a fast-typing
	// Lead's superseded analyses are abandoned, never left racing this one.
	runCtx := e.beginAnalysis()
	prompt := assist.AnalysisPrompt(draft)
	resumeID := e.resumeSessionID()
	raw, err := analyzeDraft(runCtx, cfg.RepoDir, prompt, resumeID)
	// A --resume run can fail because the warm harness session is missing or no longer
	// resumable (a stale/half-established id) — that strands authoring even though a
	// fresh read would work. Retry COLD (no resume) once before degrading, so a broken
	// warm context falls back to a working analysis instead of "assist run failed".
	if err != nil && resumeID != "" && runCtx.Err() == nil {
		stdlog.Printf("authoring: resume analysis failed (%v) — retrying cold", err)
		raw, err = analyzeDraft(runCtx, cfg.RepoDir, prompt, "")
	}
	if runCtx.Err() != nil {
		return // superseded by a newer analyze — let that one own the cache
	}
	if err != nil {
		stdlog.Printf("authoring: analysis run failed: %v", err)
		e.setAnalysis(&draftAnalysis{Draft: draft, Reason: "the assist run failed — try again"})
		c.Analysis.Write(ctx, "err")
		return
	}
	a, err := assist.ParseAnalysis(raw, draft)
	if err != nil {
		e.setAnalysis(&draftAnalysis{Draft: draft, Reason: "the assist's output was unreadable — try again"})
		c.Analysis.Write(ctx, "err")
		return
	}
	e.setAnalysis(&draftAnalysis{Draft: draft, Result: &a})
	// A fresh analysis supersedes any pending rewrite from a prior update — the Lead is
	// reading the current draft now, so a stale rewrite must never be re-pushed later.
	e.setRewrite("")
	c.Analysis.Write(ctx, "ok")
}

// UpdateDraft folds the Lead's answers to the clarifying questions back into the
// draft: it runs a assist over the current draft + the answers (the rewrite seam,
// the same haiku/warm read AnalyzeDraft uses) and swaps the editor to the rewritten
// text. An empty draft or empty/malformed answer set is a silent no-op (nothing to
// fold in). A failed or empty rewrite degrades calmly — the draft and the questions
// are kept so the Lead can retry, never wiped. On success the stale analysis is
// CLEARED (the questions were answered against a draft that no longer exists; the Lead
// re-analyzes the new draft when ready — replace-only, no auto re-analyze). FIREWALL:
// like AnalyzeDraft it writes only the off-economy caches, never the ledger.
func (c *LiveCard) UpdateDraft(ctx *via.Ctx) {
	cfg, log := readLiveState(c.Key)
	if log == nil {
		return
	}
	e := lookupLiveEntry(c.Key)
	if e == nil {
		return
	}
	draft := strings.TrimSpace(c.Draft.Read(ctx))
	if draft == "" {
		return // nothing to rewrite
	}
	var answers []assist.Answer
	if err := json.Unmarshal([]byte(c.DraftAnswers.Read(ctx)), &answers); err != nil || len(answers) == 0 {
		return // malformed or empty answers — nothing to fold in, never a crash
	}
	// Supersede any in-flight analyze so a stale read can't overwrite the rewrite's
	// cleared-analysis state, and run the rewrite under a cancellable context.
	runCtx := e.beginAnalysis()
	prompt := assist.RewritePrompt(draft, answers)
	resumeID := e.resumeSessionID()
	raw, err := analyzeDraft(runCtx, cfg.RepoDir, prompt, resumeID)
	// Same warm-resume fallback as AnalyzeDraft: a stale/missing warm session strands
	// the rewrite even though a cold run would work — retry cold once before degrading.
	if err != nil && resumeID != "" && runCtx.Err() == nil {
		stdlog.Printf("authoring: resume rewrite failed (%v) — retrying cold", err)
		raw, err = analyzeDraft(runCtx, cfg.RepoDir, prompt, "")
	}
	if runCtx.Err() != nil {
		return // superseded by a newer run — let that one own the state
	}
	if err != nil {
		stdlog.Printf("authoring: rewrite run failed: %v", err)
		return // keep the draft and questions so the Lead can retry
	}
	newDraft := assist.ParseRewrite(raw)
	if newDraft == "" {
		stdlog.Printf("authoring: rewrite produced empty draft — keeping the current draft")
		return // an empty rewrite would wipe the editor; treat as failure
	}
	e.setRewrite(newDraft)
	e.setAnalysis(nil) // the questions were answered; the draft they flagged is gone
	c.Rewrite.Write(ctx, newDraft)
}

// parseHandshakeStrength maps the compose card's strength pick to a
// packet.HandshakeStrength, reporting ok=false for a blank/unrecognized pick
// — strength is SELF-DECLARED by the Lead (the handshake's strength gradient),
// never inferred or defaulted, so an unrecognized pick is refused rather
// than silently coerced to some default rung.
func parseHandshakeStrength(s string) (packet.HandshakeStrength, bool) {
	switch s {
	case "examples":
		return packet.StrengthExamples, true
	case "properties":
		return packet.StrengthProperties, true
	default:
		return packet.StrengthNone, false
	}
}

// AuthorHandshake writes the handshake the Lead composed in the
// compose card's handshake control to the protected handshake/ directory —
// internal/settle's deny-rule then refuses any LATER agent turn that touches it,
// so the contract is authored independently of, and before, the live packet's own
// code. A blank draft or an unrecognized/blank strength pick is a silent no-op —
// nothing is written, and there is nothing dishonest to fall back to (Send
// simply keeps refusing until one is authored). On success the resulting
// packet.Handshake (path/hash/self-declared strength) is cached on the session so
// Send can fold it into the next sent packet's Target. FIREWALL: like
// AnalyzeDraft, it never touches the ledger — authoring a handshake mints nothing.
func (c *LiveCard) AuthorHandshake(ctx *via.Ctx) {
	cfg, log := readLiveState(c.Key)
	if log == nil {
		return
	}
	e := lookupLiveEntry(c.Key)
	if e == nil {
		return
	}
	draft := strings.TrimSpace(c.HandshakeDraft.Read(ctx))
	strength, ok := parseHandshakeStrength(c.HandshakeStrengthPick.Read(ctx))
	if draft == "" || !ok {
		return // nothing to author — never a guessed strength or an empty contract
	}
	h, err := packet.WriteHandshake(cfg.RepoDir, "spec_test", draft, strength)
	if err != nil {
		stdlog.Printf("authoring: write handshake failed: %v", err)
		return
	}
	e.setPendingHandshake(&h)
	e.setComposeMessage("") // a fresh handshake clears any earlier "author one" refusal
	c.HandshakeAuthored.Write(ctx, "ok")
}

// renderAuthoring is the authoring-assist surface: an editable Monaco editor as the
// single draft source, with the assist's structured read (summary + clarifying
// questions) beneath it. The assist's flagged spans are decorated INLINE in the
// editor itself (not a separate mirror), and the readiness verdict reflects beside
// place. da is the latest cached analysis (nil before the first run).
func renderAuthoring(c *LiveCard) h.H {
	var da *draftAnalysis
	var rewrite string
	var handshakeAuthored bool
	var composeMessage string
	if e := lookupLiveEntry(c.Key); e != nil {
		da = e.analysisSnapshot()
		rewrite = e.rewriteSnapshot()
		handshakeAuthored = e.pendingHandshakeSnapshot() != nil
		composeMessage = e.composeMessageSnapshot()
	}
	parts := []h.H{
		h.Class("authoring"),
		renderHandshakeAuthoring(c, handshakeAuthored, composeMessage),
		composeSurface(da, rewrite),
	}
	if p := renderAnalysisPanel(da); p != nil {
		parts = append(parts, p)
	}
	return h.Div(parts...)
}

// renderHandshakeAuthoring is the handshake compose control:
// a plain textarea + a self-declared strength pick, bound directly (data-bind,
// no CustomEvent bridge — unlike the Monaco draft editor, this is a small plain
// form) and posted to AuthorHandshake. authored reflects whether the session
// currently has one cached (Send consumes it on a successful placement,
// so this reverts to "none" after each send). message is Send's
// honest inline refusal ("" when there is none to show).
func renderHandshakeAuthoring(c *LiveCard, authored bool, message string) h.H {
	state, statusText := "none", "no handshake authored yet"
	if authored {
		state, statusText = "authored", "handshake authored"
	}
	parts := []h.H{
		h.Class("compose__handshake"),
		h.Attr("aria-label", "author a handshake"),
		h.P(h.Class("compose__handshake-label"),
			h.Text("Author a handshake — a runnable contract the agent's turn cannot touch:")),
		h.Textarea(c.HandshakeDraft.Bind(), h.Class("pk-input compose__handshake-draft"),
			h.Placeholder("package handshake\n\nfunc TestX(t *testing.T) { ... }")),
		h.Select(c.HandshakeStrengthPick.Bind(), h.Class("pk-input compose__handshake-strength"),
			h.Option(h.Attr("value", ""), h.Text("choose a strength")),
			h.Option(h.Attr("value", "examples"), h.Text("examples")),
			h.Option(h.Attr("value", "properties"), h.Text("properties")),
		),
		h.Button(on.Click(c.AuthorHandshake), h.Class("pk-btn pk-btn--quiet compose__handshake-submit"), h.Text("Author handshake")),
		h.Span(h.Class("compose__handshake-status"), h.Data("state", state), h.Text(statusText)),
	}
	if message != "" {
		parts = append(parts, h.P(h.Class("compose__handshake-message"), h.Data("state", "blocking"), h.Text(message)))
	}
	return h.Div(parts...)
}

// composeSurface is the editable Monaco composer. The PERSISTENT interactive subtree
// (.compose__live, data-ignore-morph) holds the editor + buttons + indicator so its
// DOM, the Lead's text, and the JS listeners survive every SSE re-render; the editor
// is the single draft source. The buttons dispatch CustomEvents the wrapper's
// data-on bridge lifts into $draft before @posting the action (the maplibre /
// answer-form pattern that works without data-bind and survives morphs). The
// re-rendering bits (readiness, the highlights payload the editor decorates from) sit
// OUTSIDE the shield so a fresh analysis updates them in place.
func composeSurface(da *draftAnalysis, rewrite string) h.H {
	live := h.Div(
		h.Class("compose__live"),
		h.DataIgnoreMorph(),
		h.Attr("aria-label", "author a live packet"),
		// The bridge: each button's CustomEvent carries the editor's value, which the
		// handler assigns to $draft INLINE (so the signal is present at post time)
		// then @posts the action AnalyzeDraft/Send reads. The update bridge also
		// carries the gathered answers into $draftanswers for UpdateDraft.
		h.Data("on:viaanalyze", "$draft=evt.detail.draft;@post('/_action/AnalyzeDraft')"),
		h.Data("on:viaplace", "$draft=evt.detail.draft;@post('/_action/Send')"),
		h.Data("on:viaupdatedraft", "$draft=evt.detail.draft;$draftanswers=evt.detail.answers;@post('/_action/UpdateDraft')"),
		h.Div(h.ID("authoring-editor"), h.Class("compose__editor")),
		h.Button(h.Type("button"), h.Class("pk-btn pk-btn--quiet compose__analyze"), h.Text("Analyze intent")),
		h.Button(h.Type("button"), h.Class("pk-btn compose__place"), h.Text("Compose packet")),
		h.Span(h.Class("compose__analyzing"), h.Data("state", "idle"), h.Text("analyzing…")),
		h.Script(h.Src(monacoLoaderURL)),
		h.Script(h.Raw(authoringEditorJS)),
	)
	parts := []h.H{h.Class("compose"), live}
	// Once the assist has read the draft, reflect its readiness beside place — a
	// guide, never a gate (placing stays allowed at any readiness). Outside the shield
	// so a fresh verdict re-renders in place.
	if da != nil && da.Result != nil {
		state, note := "caution", "The assist flagged open questions — placing will run the draft as-is."
		if da.Result.Ready {
			state, note = "ready", "The assist judged this ready to run unattended."
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
	// The rewrite payload the editor swaps to: UpdateDraft stashes the assist's
	// rewritten draft, and the editor's MutationObserver calls setValue when this
	// payload changes to a non-empty draft (json.Marshal escapes <,>,& so it is safe
	// inside the <script>). Empty before any update — the observer ignores empty.
	rw, _ := json.Marshal(struct {
		Draft string `json:"draft"`
	}{Draft: rewrite})
	parts = append(parts, h.Script(h.Type("application/json"), h.ID("authoring-rewrite-data"), h.Raw(string(rw))))
	if tokenStore == nil || !tokenStore.Configured() {
		parts = append(parts, h.Div(
			h.Class("compose__needs-key"),
			h.Text("No Anthropic API key configured — "),
			h.A(h.Href("/settings"), h.Class("compose__needs-key-link"), h.Text("set one in settings")),
			h.Text(" to compose live packets."),
		))
	}
	return h.Div(parts...)
}

// renderAnalysisPanel renders the assist's structured read beneath the editor: a
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
	// The assist's flagged spans, surfaced as readable cards (not only as inline
	// editor decorations) — the built composer's harness-pair panel, from real
	// assist output.
	if flags := renderHighlightCards(da.Draft, a.Highlights); len(flags) > 0 {
		parts = append(parts, h.Span(h.Class("analysis__flags-label"), h.Text("Flagged in the draft:")))
		parts = append(parts, flags...)
	}
	if len(a.Questions) > 0 {
		qs := []h.H{h.Class("analysis__questions")}
		for i, q := range a.Questions {
			qs = append(qs, renderQuestion(i, q))
		}
		parts = append(parts,
			h.Span(h.Class("analysis__questions-label"), h.Text("Answer these, then update the intent:")),
			h.Ul(qs...),
			// One Update-draft control at the END (not per question): the Lead answers
			// all the questions, then this single button gathers the picks + notes and the
			// current draft into the viaupdatedraft bridge (see authoringEditorJS) so the
			// assist rewrites the draft incorporating them. It is a plain button (no
			// data-bind) — the delegated click handler fires it — so it survives morphs.
			h.Button(h.Type("button"), h.Class("pk-btn analysis__update"), h.Text("Update intent")),
		)
	}
	return h.Div(parts...)
}

// lineOfOffset maps a byte offset into draft to its 1-based line — the "where" a
// flag names. It counts newlines before the offset, clamping a negative offset to
// line 1 and a past-the-end offset to the last line, so a stale/garbled offset
// yields a real location rather than line 0 or a panic.
func lineOfOffset(draft string, offset int) int {
	if offset < 0 {
		offset = 0
	}
	if offset > len(draft) {
		offset = len(draft)
	}
	return 1 + strings.Count(draft[:offset], "\n")
}

// renderHighlightCards renders the assist's flagged spans as cards: each
// carries the flag's note, its line location (computed from the real draft +
// offset), and — when present — its severity as a tag. A flag with no readable
// note (empty or whitespace-only) is skipped rather than shown as a blank card —
// never a decorative flag with nothing to say. Real assist output only; no
// fabricated flag, no invented line.
func renderHighlightCards(draft string, hs []assist.Highlight) []h.H {
	var out []h.H
	for _, hl := range hs {
		note := strings.TrimSpace(hl.Note)
		if note == "" {
			continue
		}
		head := []h.H{h.Class("analysis__flag-head")}
		if sev := strings.TrimSpace(hl.Severity); sev != "" {
			head = append(head, h.Span(h.Class("analysis__flag-severity"), h.Text(sev)))
		}
		head = append(head, h.Span(h.Class("analysis__flag-where"),
			h.Text("line "+strconv.Itoa(lineOfOffset(draft, hl.Start)))))
		// The card carries the flag's byte offsets so a delegated click can reveal the
		// exact span in the compose editor (analysis__flag--nav marks it navigable) —
		// keyed on the real offsets, converted to editor positions client-side.
		out = append(out, h.Div(
			h.Class("pk-card analysis__flag analysis__flag--nav"),
			h.Data("start", strconv.Itoa(hl.Start)),
			h.Data("end", strconv.Itoa(hl.End)),
			h.Div(head...),
			h.Span(h.Class("analysis__flag-note"), h.Text(note)),
		))
	}
	return out
}

// renderQuestion renders one clarifying question as an answerable form item: the
// question text, its suggested answers as a choice set (radios when one answer
// applies, checkboxes when several may — multiselect), and a free-text note input so
// the Lead can add context or give a different answer entirely. Inputs are scoped to
// this question by index (name="qans-<i>" / "qnote-<i>") so picks never bleed across
// questions, and the JS gathers answers by that convention. A question with no
// suggestions is still answerable via its note.
func renderQuestion(i int, q assist.Question) h.H {
	idx := strconv.Itoa(i)
	item := []h.H{
		h.Class("analysis__question"), h.Data("q", idx),
		h.Span(h.Class("analysis__question-text"), h.Text(q.Q)),
	}
	if len(q.Suggestions) > 0 {
		inputType := "radio"
		if q.Multiselect {
			inputType = "checkbox"
		}
		choices := []h.H{h.Class("analysis__choices")}
		for _, s := range q.Suggestions {
			choices = append(choices, h.Label(h.Class("analysis__choice"),
				h.Input(h.Type(inputType), h.Attr("name", "qans-"+idx), h.Attr("value", s)),
				h.Span(h.Class("analysis__choice-text"), h.Text(s)),
			))
		}
		item = append(item, h.Div(choices...))
	}
	item = append(item, h.Input(
		h.Type("text"), h.Class("analysis__note"), h.Attr("name", "qnote-"+idx),
		h.Placeholder("add a note or a different answer"),
	))
	return h.Li(item...)
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
    // The Update-draft control lives in the analysis panel (OUTSIDE the editor shield),
    // so it re-renders with each fresh analysis — a DELEGATED click on document survives
    // those re-renders. It gathers each question's picked suggestions + note into the
    // {Q,Answers,Note} shape UpdateDraft decodes (scoped by the data-q index / qans-/
    // qnote- name convention renderQuestion emits) and dispatches the bridge with the
    // current draft + answers JSON.
    document.addEventListener('click', function(ev){
      var btn = ev.target.closest ? ev.target.closest('.analysis__update') : null;
      if (!btn || !live) return;
      var qs = document.querySelectorAll('.analysis__question'), answers = [];
      for (var qi = 0; qi < qs.length; qi++) {
        var qel = qs[qi], i = qel.getAttribute('data-q');
        var qtextEl = qel.querySelector('.analysis__question-text');
        var picks = [], checked = qel.querySelectorAll('input[name="qans-' + i + '"]:checked');
        for (var ci = 0; ci < checked.length; ci++) picks.push(checked[ci].value);
        var noteEl = qel.querySelector('input[name="qnote-' + i + '"]');
        answers.push({ Q: qtextEl ? qtextEl.textContent : '', Answers: picks, Note: noteEl ? noteEl.value : '' });
      }
      live.dispatchEvent(new CustomEvent('viaupdatedraft', { detail: { draft: ed.getValue(), answers: JSON.stringify(answers) } }));
    });
    // Navigable flag cards: a delegated click on a flagged-span card (also outside
    // the editor shield, re-rendered per analysis) reveals its span in the editor,
    // converting the flag's byte offsets to positions the same way applyDecos does.
    // Delegated on document so it survives the panel's SSE re-renders.
    document.addEventListener('click', function(ev){
      var card = ev.target.closest ? ev.target.closest('.analysis__flag--nav') : null;
      if (!card) return;
      var s = parseInt(card.getAttribute('data-start'), 10), e = parseInt(card.getAttribute('data-end'), 10);
      if (isNaN(s)) return;
      if (isNaN(e) || e < s) e = s;
      var model = ed.getModel(); if (!model) return;
      var ps = model.getPositionAt(s), pe = model.getPositionAt(e);
      var range = new monaco.Range(ps.lineNumber, ps.column, pe.lineNumber, pe.column);
      ed.revealRangeInCenter(range);
      ed.setSelection(range);
      ed.focus();
    });
    // The rewrite payload: when UpdateDraft pushes a new draft, swap the editor to it
    // (only when non-empty and actually changed, so the initial empty payload never
    // wipes the editor). Clearing lastAnalyzed lets the new draft be (re)analyzed.
    function applyRewrite(){
      var el = document.getElementById('authoring-rewrite-data');
      if (!el) return;
      var d = '';
      try { d = (JSON.parse(el.textContent) || {}).draft || ''; } catch (e) { return; }
      if (d && d !== ed.getValue()) { ed.setValue(d); lastAnalyzed = null; }
    }
    var rwEl = document.getElementById('authoring-rewrite-data');
    if (rwEl && window.MutationObserver) {
      new MutationObserver(applyRewrite).observe(rwEl, { childList: true, characterData: true, subtree: true });
    }
    applyDecos();
    ed.focus();
  });
})();`
