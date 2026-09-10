package app

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/go-via/via"
	"github.com/go-via/via/h"

	"github.com/joaomdsg/packets/internal/ledger"
	"github.com/joaomdsg/packets/internal/mutation"
	"github.com/joaomdsg/packets/internal/packet"
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
	// WO, when set (/review?wo=<id>), drills into a filled packet's review: that
	// packet's own surviving mutants (the test-debt the funded work left), not the
	// session's connect-cycle findings — the send→review tie.
	WO string `query:"wo"`
	// File, when set (/review?wo=<id>&file=<path>), is the changed-file-tree leaf the
	// Lead clicked — the file the diff editor shows. Empty = the packet's anchored path.
	// Pure navigation (a GET param), not a signal: the diff island is morph-shielded,
	// so only a fresh navigation re-points it.
	File string `query:"file"`
	// The reviewer's answer submission, set by the editor before posting AnswerQuestion.
	AnswerFile via.SignalStr `via:"answerfile"`
	AnswerLine via.SignalStr `via:"answerline"`
	AnswerTest via.SignalStr `via:"answertest"`
	// AnswerWO scopes an answer to a packet (>0): the re-run uses the PACKET's revs
	// and updates the packet's findings, not the session's. 0/unset = session answer.
	AnswerWO via.SignalStr `via:"answerwo"`
	// AdjFile/AdjLine/AdjText carry an anchored REVIEW ADJUSTMENT: the file and line a
	// comment targets and the comment text, set by the adjustment form. Read by
	// AddAdjustment, which composes the §12.3 review turn and dispatches it to the live
	// harness against session HEAD — the comment→harness round-trip. Per-tab signals.
	AdjFile via.SignalStr `via:"adjfile"`
	AdjLine via.SignalStr `via:"adjline"`
	AdjText via.SignalStr `via:"adjtext"`
	// AdjEndLine is the optional END of a multi-line selection (set by the Monaco
	// diff's selection bridge); empty/0 or equal to AdjLine means a single line.
	// AddAdjustment records it as the annotation's EndLine so a range anchors as
	// the span it covers.
	AdjEndLine via.SignalStr `via:"adjendline"`
	// ConfirmWO carries the packet id ConfirmIntentFidelity confirms (G1's
	// human residual) — set inline by the confirm button's datastar expr.
	ConfirmWO via.SignalStr `via:"confirmwo"`
	// ReplyParent/ReplyText carry a reply to an existing annotation: the id of the
	// annotation being answered (set inline by the reply form's datastar expr) and
	// the reply text (bound to the form input). Read by ReplyToAnnotation, which
	// persists the reply under its parent and re-triggers the harness. Per-tab.
	ReplyParent via.SignalStr `via:"replyparent"`
	ReplyText   via.SignalStr `via:"replytext"`
}

// orderOpenThreads converts a filled packet's cached findings into review
// threads. Empty when the packet is unknown, unfilled, or left no surviving mutants.
func orderOpenThreads(key string, orderID int) []review.Thread {
	e := lookupLiveEntry(key)
	if e == nil {
		return nil
	}
	return review.QuestionThreadsFromMutations(e.orderFindingsFor(orderID))
}

// packetForOrder finds a session's own folded Packet for orderID, ok=false
// when the packet is unknown or the session has no ledger to fold.
func packetForOrder(key string, orderID int) (packet.Packet, bool) {
	for _, p := range sessionPackets(key, 0) {
		if p.ID == orderID {
			return p, true
		}
	}
	return packet.Packet{}, false
}

// orderTarget finds a funded packet's Target (its base/fix revs + anchored path)
// by ID from the session's sends — the revs whose diff IS the edits. Folds ALL
// sends (mirroring packetForOrder's sessionPackets(key, 0)) so a session's oldest
// packet stays inspectable no matter how many later packets it has sent since.
func orderTarget(log *ledger.Log, orderID int) (ledger.Target, bool) {
	if log == nil {
		return ledger.Target{}, false
	}
	views, err := log.RecentSends(0)
	if err != nil {
		return ledger.Target{}, false
	}
	for _, v := range views {
		if v.ID == orderID {
			return v.Target, true
		}
	}
	return ledger.Target{}, false
}

// resolveSelectedFile decides which file the packet's diff editor opens on: an
// explicit ?file= pick (a clicked tree leaf) wins; else the packet's anchored path;
// else — for an anchorless live/prompt packet — the FIRST changed file in the
// base→fix diff, so the diff pane is never a blank box when there are edits to
// inspect; else "" (no anchor and nothing changed → an honest empty selection).
func resolveSelectedFile(cfg LiveConfig, tgt ledger.Target, requested string) string {
	if requested != "" {
		return requested
	}
	if tgt.Path != "" {
		return tgt.Path
	}
	changed, _ := diffCompute(context.Background(), cfg.RepoDir, tgt.BaseRev, tgt.FixRev)
	if len(changed.Files) > 0 {
		return changed.Files[0].Path
	}
	return ""
}

// orderDiffIsland renders a Monaco DIFF editor of the packet's base→fix edits on the
// SELECTED file — the leaf the Lead clicked in the changed-file tree, defaulting to
// the packet's anchored path when nothing is selected. The diff DATA (base + fix
// source) is the server contract; the editor's rendering is the client island
// (browser-verified). Source unreadable at a rev degrades to an empty side rather
// than breaking the surface.
func orderDiffIsland(cfg LiveConfig, tgt ledger.Target, selected string) h.H {
	if selected == "" {
		selected = tgt.Path
	}
	base, _ := reviewFileReader(context.Background(), cfg.RepoDir, tgt.BaseRev, selected)
	fix, _ := reviewFileReader(context.Background(), cfg.RepoDir, tgt.FixRev, selected)
	payload, _ := json.Marshal(struct {
		Path string `json:"path"`
		Base string `json:"base"`
		Fix  string `json:"fix"`
	}{Path: selected, Base: base, Fix: fix})
	return h.Div(
		h.Class("packet-diff-island"),
		h.DataIgnoreMorph(),
		// Selecting line(s) in the diff dispatches a viaannotate CustomEvent whose
		// detail fills the adjustment anchor signals — so the rail's "leave an
		// adjustment" form pre-fills from the selection (click = one line, drag = a
		// range) instead of the Lead typing file:line by hand. The mirror of the
		// answer editor's viaanswer bridge.
		h.Data("on:viaannotate", "$adjfile=evt.detail.file;$adjline=evt.detail.start;$adjendline=evt.detail.end"),
		h.Script(h.Type("application/json"), h.ID("packet-diff-data"), h.Raw(string(payload))),
		h.Div(h.ID("packet-diff-editor"), h.Class("packet-diff-editor")),
		h.Script(h.Src(monacoLoaderURL)),
		h.Script(h.Raw(orderDiffBootstrapJS)),
	)
}

// orderDiffBootstrapJS mounts a read-only Monaco diff editor over the base/fix
// payload — the edits the packet made, side by side. Defensive (guards + try/catch);
// require is loaded by this island's loader.
const orderDiffBootstrapJS = `(function(){
  if (typeof require === 'undefined') return;
  require.config({ paths: { vs: 'https://cdn.jsdelivr.net/npm/monaco-editor@` + monacoVersion + `/min/vs' } });
  require(['vs/editor/editor.main'], function(){
    var el = document.getElementById('packet-diff-editor');
    var dataEl = document.getElementById('packet-diff-data');
    if (!el || !dataEl || el.dataset.mounted) return;
    el.dataset.mounted = '1';
    var d;
    try { d = JSON.parse(dataEl.textContent); } catch (e) { return; }
    var orig = monaco.editor.createModel(d.base || '', 'go', monaco.Uri.file('base/' + (d.path || 'file.go')));
    var mod = monaco.editor.createModel(d.fix || '', 'go', monaco.Uri.file('fix/' + (d.path || 'file.go')));
    var de = monaco.editor.createDiffEditor(el, { readOnly: true, automaticLayout: true, theme: 'vs-dark', renderSideBySide: false, minimap: { enabled: false }, scrollBeyondLastLine: false });
    de.setModel({ original: orig, modified: mod });
    // Selection → adjustment anchor: a click (start==end line) or a click-drag
    // (a span) on the modified side dispatches viaannotate on the island wrapper,
    // whose data-on handler fills $adjfile/$adjline/$adjendline. A zero-length
    // selection sends end==start; the server collapses that to a single line.
    try {
      var wrap = el.closest('.packet-diff-island');
      var mEd = de.getModifiedEditor();
      mEd.onDidChangeCursorSelection(function(ev){
        if (!wrap) return;
        var s = ev.selection;
        wrap.dispatchEvent(new CustomEvent('viaannotate', { detail: {
          file: d.path || '',
          start: String(s.startLineNumber),
          end: String(s.endLineNumber)
        }}));
      });
    } catch (e) {}
  });
})();`

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
	// One re-run at a time per session: a re-run spawns a git worktree + oracle run,
	// so a double-clicked submit would race the shared repo's worktree ops. Drop the
	// duplicate (the in-flight one is already answering).
	if !e.beginAnswer() {
		return
	}
	defer e.endAnswer()
	cfg, log := readLiveState(key)

	// Packet-scoped answer (/review?wo=<id>): re-run against the PACKET's fix revision
	// (the work it did), and update that packet's findings cache — not the session's.
	// The packet cache isn't re-populated by a connect cycle, so a kill sticks without
	// a resolved-set.
	if woID, err := strconv.Atoi(c.AnswerWO.Read(ctx)); err == nil && woID > 0 {
		tgt, ok := orderTarget(log, woID)
		if !ok {
			return
		}
		overlay := map[string]string{filepath.Join(filepath.Dir(file), answerTestFilename): test}
		newFindings, rerr := rerunWithOverlay(context.Background(), cfg.RepoDir, tgt.FixRev, file, line, cfg.TestCmd, overlay)
		if rerr != nil {
			return // transient — leave the packet's question open (flaky-truth fence)
		}
		e.setOrderFindings(woID, newFindings) // off-ledger; a kill empties → question vanishes
		return
	}

	overlay := map[string]string{filepath.Join(filepath.Dir(file), answerTestFilename): test}
	newFindings, err := rerunWithOverlay(context.Background(), cfg.RepoDir, cfg.FixRev, file, line, cfg.TestCmd, overlay)
	if err != nil {
		return // transient — leave the question open, retryable (flaky-truth fence)
	}
	// If the answered line is GONE from the re-run findings, the reviewer's test
	// killed the mutant — mark it resolved so it stays vanished for the session even
	// when a later connect cycle re-finds the (uncommitted) survivor (so the
	// question stays vanished). A still-surviving line is NOT resolved (honest: try again).
	if !findingsHaveLine(newFindings, file, line) {
		e.markResolved(file, line)
		e.recordQuestionUnblock(file, line) // a killing answer clears the block — earns attention bandwidth
	}
	// Wholesale replace is correct because the live card anchors ONE line and
	// mutation.Run is scoped to it, so the cache only ever holds that line's findings.
	// (If multi-line answering is ever added, replace only the answered line's entries.)
	e.setFindings(newFindings) // diagnostic cache only; no ledger touch — FIREWALL
}

// findingsHaveLine reports whether any finding sits on file:line — used to tell a
// killing answer (the line is gone) from a weak one (it remains).
func findingsHaveLine(fs []mutation.Finding, file string, line int) bool {
	for _, f := range fs {
		if f.File == file && f.Line == line {
			return true
		}
	}
	return false
}

// View renders the Inspector shell: the identity strip, the
// 3-column grid (changed-files tree | Monaco island + answer form | annotation
// rail), and a timeline footer. The session's open question-threads (or a
// funded packet's own) render as annotation cards on the rail; a calm
// empty state fills it when the oracle left none.
func (c *ReviewCard) View(_ *via.CtxR) h.H {
	navKey := c.Key
	if navKey == "" {
		navKey = defaultSessionKey
	}
	cfg, log := readLiveState(navKey)
	parts := []h.H{h.Class("review"), h.Data("state", "review"), navHeader(navKey, "inspect")}
	// A back-affordance so the review drill-in isn't a dead end: a link to the
	// originating session card (Flow C). The per-packet branch ALSO adds an up-link to
	// the session review, making per-packet↔session nav symmetric.
	parts = append(parts, h.Nav(h.Attr("aria-label", "return"),
		h.P(h.Class("review__return"), cardReturnCrumb(navKey))))

	// Per-packet review (/review?wo=<id>): the filled packet's OWN review questions
	// — the test-debt the funded work left — read from the per-packet findings cache,
	// not the session's connect cycle. Read-only here; packet-scoped answering is a
	// later slice. (The editable answer flow below is session-scoped.)
	if woID, err := strconv.Atoi(c.WO); err == nil && woID > 0 {
		// The per-packet review's UP-link to the session review (drop the wo scope), so a
		// funded packet's test-debt isn't a dead end — the symmetric leg of Flow C.
		parts = append(parts, h.Nav(h.Attr("aria-label", "review nav"),
			h.P(h.Class("review__up"), reviewSessionCrumb(navKey))))

		// "See the edits this packet made": the packet's base→fix diff, in a Monaco diff
		// editor. The diff is STATIC and pre-funded (the fix revision the packet ran) —
		// honest framing, never a faked "live agent typing". A packet whose target can't
		// be resolved (unfilled/unknown id) has no revs to show — the identity strip's
		// rev chip and the tree/diff regions all degrade to their honest empties.
		tgt, hasTarget := orderTarget(log, woID)
		pktName := ""
		lane := packet.LaneUnmeasured
		gauntlet := packet.Gauntlet{}
		if pkt, ok := packetForOrder(navKey, woID); ok {
			pktName = pkt.Name
			// Computed (and cached) HERE, on this render, scoped to the one
			// packet being shown in detail — never on the 100ms Stream poll.
			if e := lookupLiveEntry(navKey); e != nil {
				lane = e.laneFor(context.Background(), pkt)
				gauntlet = e.gauntletFor(context.Background(), pkt)
			}
		}
		parts = append(parts, renderInspectorTitlebar("pkt#"+strconv.Itoa(woID), tgt.BaseRev, tgt.FixRev, sessionAddr(navKey), pktName, lane))

		// The open threads drive both the annotation rail (below) and the per-file
		// badges in the tree, so fetch them once here and tally by file.
		orderThreads := orderOpenThreads(navKey, woID)
		annCounts := annotationCountsByFile(orderThreads)

		// Durable human annotations (and their replies) are folded from the log and
		// scoped to the files this packet actually changed, so the rail shows the
		// conversation on this diff — not another packet's.
		var annThreads []annotationThread
		if anns, err := log.Annotations(); err == nil && len(anns) > 0 {
			changed, _ := diffCompute(context.Background(), cfg.RepoDir, tgt.BaseRev, tgt.FixRev)
			inDiff := make(map[string]bool, len(changed.Files))
			for _, f := range changed.Files {
				inDiff[f.Path] = true
			}
			for _, th := range foldAnnotationThreads(anns) {
				if inDiff[th.Root.File] {
					annThreads = append(annThreads, th)
				}
			}
		}

		left := renderInspectorEmptyTree()
		var main []h.H
		if hasTarget {
			// The full fix tree with the packet's changes highlighted; the clicked leaf
			// (or, by default, the anchor — or the first changed file for an anchorless
			// packet) is what the diff editor shows.
			selected := resolveSelectedFile(cfg, tgt, c.File)
			left = h.Div(
				h.P(h.Class("review__lead"),
					h.Text("Changed files — PKT#"+strconv.Itoa(woID)+" (pick one to inspect):")),
				renderFileTree(cfg, tgt, woID, selected, annCounts),
			)
			main = append(main,
				h.P(h.Class("review__lead"),
					h.Text("The edits PKT#"+strconv.Itoa(woID)+" made — "+selected+":")),
				orderDiffIsland(cfg, tgt, selected),
			)
		}

		parts = append(parts, h.P(h.Class("review__lead"),
			h.Text("Inspecting PKT#"+strconv.Itoa(woID)+" — the packet's surviving mutants:")))

		rail := []h.H{annotationRailHeader(len(orderThreads) + len(annThreads))}
		if len(orderThreads) == 0 && len(annThreads) == 0 {
			rail = append(rail, h.Div(h.Class("review__empty"),
				h.Text("No open questions for this packet — the work left no surviving mutants (or it hasn't filled yet).")))
		} else {
			if len(orderThreads) > 0 {
				rail = append(rail, renderAnnotationCards(orderThreads)...)
				// Answer the packet's questions in-place: the editable pane, scoped to THIS
				// packet ($answerwo) so the re-run uses the packet's revs, not the session's.
				main = append(main, renderAnswerForm(orderThreads[0], woID))
			}
			// Durable human annotations + replies render beneath the oracle findings.
			rail = append(rail, renderAnnotationThreads(c, annThreads)...)
		}
		// The adjustment entry point (DESIGN §12.3, the ✎ you-authored zone): leave an
		// anchored comment and the live harness re-edits in place. Present on every
		// review view so the Lead can always tell the agent what to change.
		rail = append(rail, renderAdjustmentForm(c))

		parts = append(parts, renderInspectorGrid(left, h.Div(main...), rail))
		parts = append(parts, renderInspectorTimeline(navKey, woID, gauntlet))
		return h.Div(parts...)
	}

	threads := sessionOpenThreads(navKey)
	// The session-scoped review has no single packet to measure — a lane is
	// meaningless without one, so this stays the honest LaneUnmeasured rather
	// than computing anything.
	parts = append(parts, renderInspectorTitlebar(navKey, cfg.BaseRev, cfg.FixRev, sessionAddr(navKey), "", packet.LaneUnmeasured))

	var main []h.H
	if len(threads) > 0 {
		parts = append(parts, h.P(h.Class("review__lead"),
			h.Text(strconv.Itoa(len(threads))+" open — surviving mutants the tests didn't catch:")))
		// The editor island: a DOM subtree the client-side Monaco review editor (a later
		// slice) mounts into, plus the SAME threads as a machine-readable JSON payload so
		// the editor reads structured data, not the human text above. data-ignore-morph
		// shields the editor's own DOM from being clobbered by an SSE re-render. Emitted
		// only when there ARE questions — nothing to scaffold over an empty set.
		main = append(main, reviewEditorIsland(cfg, threads))
		// The answer affordance: write the killing test in an editable Monaco pane and
		// submit. Following the maplibre plugin's client→server pattern (the only one
		// that survives morphs cleanly): the editor + submit live in ONE data-ignore-morph
		// wrapper, the submit dispatches a CustomEvent carrying the editor's value, and the
		// wrapper's data-on:viaanswer ASSIGNS the answer signals from evt.detail INLINE,
		// then @posts AnswerQuestion. Assigning in the datastar expression (not data-bind)
		// is what makes the signal reliably present at post time. AnswerQuestion re-runs
		// the oracle with the test injected — a kill makes the question vanish
		// (diagnostic, off-economy).
		main = append(main, renderAnswerForm(threads[0], 0))
	}

	rail := []h.H{annotationRailHeader(len(threads))}
	if len(threads) == 0 {
		rail = append(rail, h.Div(h.Class("review__empty"),
			h.Text("No open questions — every mutant it tried was caught (or this addr hasn't run a cycle yet).")))
	} else {
		rail = append(rail, renderAnnotationCards(threads)...)
	}
	// The adjustment entry point, present on every review view (see the per-packet
	// branch above for the full rationale).
	rail = append(rail, renderAdjustmentForm(c))

	parts = append(parts, renderInspectorGrid(renderInspectorEmptyTree(), h.Div(main...), rail))
	// The session-scoped review has no single packet to gauntlet — the honest
	// zero-value Gauntlet (every gate NotRun), same pattern as the lane chip
	// just above staying LaneUnmeasured. orderID 0 also omits the confirm
	// affordance: there is no packet id to confirm against here.
	parts = append(parts, renderInspectorTimeline(navKey, 0, packet.Gauntlet{}))
	return h.Div(parts...)
}

// renderAnswerForm renders the editable Monaco answer pane + submit, wired to
// AnswerQuestion via the maplibre-style data-on bridge. woID>0 scopes the answer to
// a packet (the re-run uses the packet's revs) by setting $answerwo inline; woID
// 0 is the session review.
func renderAnswerForm(anchor review.Thread, woID int) h.H {
	expr := "$answerfile=evt.detail.file;$answerline=evt.detail.line;$answertest=evt.detail.test;@post('/_action/AnswerQuestion')"
	if woID > 0 {
		expr = "$answerwo=" + strconv.Itoa(woID) + ";" + expr
	}
	return h.Div(
		h.Class("review-answer"),
		h.P(h.Class("review-answer__label"),
			h.Text("Answer: write a test that kills the mutant on "+anchor.File+":"+strconv.Itoa(anchor.StartLine)+" (⌘/Ctrl+Enter to submit)")),
		h.Div(
			h.Class("review-answer__input"),
			h.DataIgnoreMorph(),
			// datastar catches the editor's submit CustomEvent, lifts its detail into
			// the answer signals, and posts the action — the maplibre-proven bridge.
			h.Data("on:viaanswer", expr),
			// while that post is in flight (the oracle re-run takes seconds), the
			// "answering" signal is true so the running line below reveals itself.
			h.Attr("data-indicator", "answering"),
			h.Div(h.ID("answer-editor"), h.Class("review-answer__editor")),
			h.Button(h.Type("button"), h.Class("pk-btn review-answer__submit"),
				h.Text("Submit answer — recheck the gate")),
			h.Script(h.Raw(answerEditorJS(anchor.File, anchor.StartLine))),
		),
		// Shown ONLY while the re-run is in flight (data-show on the indicator signal):
		// a calm status, not dead-air, for the seconds the oracle takes.
		h.Div(
			h.Attr("data-show", "$answering"),
			h.Class("review-answer__running"),
			h.Text("rechecking the gate — checking if your test kills the mutant…"),
		),
	)
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
// server-rendered text threads as the fallback. Read-only (editable
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
