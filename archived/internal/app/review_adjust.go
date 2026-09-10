package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-via/via"
	"github.com/go-via/via/h"
	"github.com/go-via/via/on"

	"github.com/joaomdsg/packets/internal/assist"
	"github.com/joaomdsg/packets/internal/ledger"
)

// ResolveAdjustment clears the adjustment anchored at the AdjFile:AdjLine signals — the
// Lead marking it addressed so it leaves the list (closing the review-thread loop
// symmetrically with the answer-vanish flow). Off the economy ledger, no harness send
// (unlike AddAdjustment): it only forgets the anchor. A missing entry / treeless session /
// blank or non-positive line is a silent no-op.
func (c *ReviewCard) ResolveAdjustment(ctx *via.Ctx) {
	key := c.Key
	if key == "" {
		key = defaultSessionKey
	}
	e := lookupLiveEntry(key)
	if e == nil {
		return
	}
	file := strings.TrimSpace(c.AdjFile.Read(ctx))
	line, _ := strconv.Atoi(strings.TrimSpace(c.AdjLine.Read(ctx)))
	if file == "" || line < 1 {
		return
	}
	e.removeAdjAnchor(file, line)
}

// AddAdjustment is the keystone of the review-thread loop (DESIGN §12.3): leaving an
// anchored comment on a line sends a harness TURN that tells the agent what to
// fix, run against the session's live HEAD — so "leave an adjustment" becomes "watch
// the agent fix it". It is an additive reuse of the live-packet pipe: the comment turn
// is sent and drained exactly like a placed packet (funded by attention
// bandwidth), with the agent's edits folding into the next settled revision. An empty
// comment, a treeless session, or an over-budget meter is a silent no-op.
func (c *ReviewCard) AddAdjustment(ctx *via.Ctx) {
	key := c.Key
	if key == "" {
		key = defaultSessionKey
	}
	cfg, log := readLiveState(key)
	if log == nil {
		return
	}
	text := strings.TrimSpace(c.AdjText.Read(ctx))
	if text == "" {
		return // an empty comment is nothing to address
	}
	head, ok := repoHead(cfg.RepoDir)
	if !ok {
		return // no resolvable tree to route the comment against
	}
	file := strings.TrimSpace(c.AdjFile.Read(ctx))
	line, _ := strconv.Atoi(strings.TrimSpace(c.AdjLine.Read(ctx)))
	// Quote the line under review when we can read it (best-effort; a missing/unreadable
	// line just omits the quote — the file:line and the comment still carry the turn).
	codeLine := readSourceLine(cfg.RepoDir, file, line)
	// Remember the anchor (with the comment) so a later render can relocate it against the
	// settled revision and show whether the agent addressed it (DESIGN §28 thin slice).
	// Only when we could read the line — an unquotable anchor has nothing to relocate by
	// content. Upserted by file:line, so several distinct adjustments are each tracked
	// (not just the last) while re-commenting a line replaces rather than stacks.
	if codeLine != "" {
		if e := lookupLiveEntry(key); e != nil {
			e.addAdjAnchor(file, line, codeLine, text)
		}
	}
	// A multi-line selection (the Monaco diff sets adjendline) anchors the comment
	// to the whole span; an end at/below the start — or absent/non-numeric — is a
	// single line, so EndLine stays 0 rather than a bogus/inverted range.
	endLine, _ := strconv.Atoi(strings.TrimSpace(c.AdjEndLine.Read(ctx)))
	if endLine <= line {
		endLine = 0
	}

	// Persist the anchored comment as a DURABLE annotation before the send, so
	// the comment is recorded on the log (and folds into the review rail) even when
	// the re-trigger below is refused for budget — the human said it, so it stays.
	// The id is sequential over the session's existing annotations, giving each a
	// stable handle a reply can target.
	existing, _ := log.Annotations()
	_ = log.AppendAnnotation(ledger.AnnotationRecord{
		ID:        fmt.Sprintf("ann-%d", len(existing)+1),
		File:      file,
		StartLine: line,
		EndLine:   endLine,
		Author:    "lead",
		Body:      text,
		AtUnixMs:  time.Now().UnixMilli(),
	})

	tgt := ledger.Target{BaseRev: head, Prompt: assist.ReviewTurnPrompt(file, line, codeLine, text)}
	// Funded by attention bandwidth like any UI-authored live packet; an over-budget
	// meter is refused by the ledger and is a silent no-op for the re-trigger.
	if err := log.AppendLiveSend("liveorder", tgt, ownTargetOf(cfg)); err != nil {
		return
	}
	go drainQueuedPackets(key)
}

// adjAnchorState is where an adjustment's commented line ended up after the agent
// settled a new revision — the thin content-relocation slice of DESIGN §28.
type adjAnchorState int

const (
	adjSame     adjAnchorState = iota // still on the line the Lead commented
	adjMoved                          // the same line survived but shifted to a new number
	adjOutdated                       // the line was edited/deleted — the agent addressed it
)

// adjReanchor is the relocation outcome: the state plus the 1-based line the anchor now
// sits on (0 when outdated — there is no surviving line to point at).
type adjReanchor struct {
	State adjAnchorState
	Line  int
}

// adjAnchorRecord is one adjustment's anchor cached on a liveEntry: the file:line the Lead
// commented, the content of that line at comment time (against which a later revision is
// relocated — reanchorAdjustment), and the Lead's comment text (shown beside the status).
type adjAnchorRecord struct {
	file    string
	line    int
	content string
	comment string
}

// adjStatus is one adjustment's relocated state for the surface: its file + the Lead's
// comment, plus where the commented line ended up in the settled revision.
type adjStatus struct {
	File    string
	Comment string
	State   adjAnchorState
	Line    int
}

// upsertAnchor adds an adjustment anchor with last-writer semantics per file:line: if an
// entry for the same file AND line already exists, it is REPLACED in place (latest
// content+comment, position preserved) so re-commenting a line updates its one badge
// rather than stacking a duplicate; otherwise the record is appended. Bounds the list to
// one entry per commented line.
func upsertAnchor(anchors []adjAnchorRecord, rec adjAnchorRecord) []adjAnchorRecord {
	for i, a := range anchors {
		if a.file == rec.file && a.line == rec.line {
			anchors[i] = rec
			return anchors
		}
	}
	return append(anchors, rec)
}

// removeAnchor drops the adjustment matching file AND line (order preserved), returning a
// new slice; a file:line not present is a no-op. The twin of upsertAnchor — used to RESOLVE
// (clear) an addressed adjustment so the list doesn't accumulate for the session.
func removeAnchor(anchors []adjAnchorRecord, file string, line int) []adjAnchorRecord {
	out := make([]adjAnchorRecord, 0, len(anchors))
	for _, a := range anchors {
		if a.file == file && a.line == line {
			continue
		}
		out = append(out, a)
	}
	return out
}

// relocateAdjustments relocates EVERY remembered adjustment against its file's current
// content (read injected — no disk here), preserving order and carrying each anchor's own
// comment through. So the surface shows every adjustment's addressed/moved/outdated state,
// not just the last. An unreadable file (read returns "") relocates to Outdated.
func relocateAdjustments(anchors []adjAnchorRecord, read func(file string) string) []adjStatus {
	out := make([]adjStatus, 0, len(anchors))
	for _, a := range anchors {
		r := reanchorAdjustment(a.content, a.line, read(a.file))
		out = append(out, adjStatus{File: a.file, Comment: a.comment, State: r.State, Line: r.Line})
	}
	return out
}

// reanchorAdjustment relocates an adjustment's commented line against a new revision's
// file content by EXACT line-content match (DESIGN §28, thin slice — no git-hunk rebase
// or rename tracking). An empty anchor has nothing to relocate. The original line still
// at its old number is Same; the same content found elsewhere is Moved to the first such
// line; content gone entirely is Outdated. A trailing "\r" is trimmed on both sides so a
// \r\n revision doesn't spuriously read as edited.
func reanchorAdjustment(origLine string, origLineNum int, newContent string) adjReanchor {
	if origLine == "" {
		return adjReanchor{State: adjOutdated}
	}
	want := strings.TrimSuffix(origLine, "\r")
	lines := strings.Split(newContent, "\n")
	if origLineNum >= 1 && origLineNum <= len(lines) &&
		strings.TrimSuffix(lines[origLineNum-1], "\r") == want {
		return adjReanchor{State: adjSame, Line: origLineNum}
	}
	for i, ln := range lines {
		if strings.TrimSuffix(ln, "\r") == want {
			return adjReanchor{State: adjMoved, Line: i + 1}
		}
	}
	return adjReanchor{State: adjOutdated}
}

// renderAdjustmentForm renders the review surface's adjustment entry point: file +
// line + comment inputs (bound to the adjustment signals) and a button wired to
// AddAdjustment. This is the UI half of the comment→harness round-trip — the Lead
// tells the agent what to change and the live harness re-edits in place.
func renderAdjustmentForm(c *ReviewCard) h.H {
	parts := []h.H{
		h.Class("review-adjust"),
		h.P(h.Class("review-adjust__label"), h.Text("Leave an adjustment — tell the agent what to change:")),
		h.Input(h.Type("text"), c.AdjFile.Bind(), h.Class("pk-input review-adjust__file"), h.Placeholder("file (e.g. main.go)")),
		h.Input(h.Type("text"), c.AdjLine.Bind(), h.Class("pk-input review-adjust__line"), h.Placeholder("line"), h.Attr("aria-label", "line number")),
		h.Input(h.Type("text"), c.AdjText.Bind(), h.Class("pk-input review-adjust__text"), h.Placeholder("what should change?")),
		h.Button(on.Click(c.AddAdjustment), h.Class("pk-btn review-adjust__submit"), h.Text("Leave adjustment")),
	}
	parts = append(parts, renderAdjustmentStatuses(c)...)
	return h.Div(parts...)
}

// renderAdjustmentStatuses shows whether EACH adjustment this session was addressed: it
// relocates every cached anchor against its file's CURRENT content (the settled revision
// on disk) and renders one badge per adjustment — the Lead's comment plus still-here /
// moved / line-edited — so "leave adjustmentS → watch them addressed" has a visible
// payoff (DESIGN §28 thin slice). Empty when none were left this session.
func renderAdjustmentStatuses(c *ReviewCard) []h.H {
	key := c.Key
	if key == "" {
		key = defaultSessionKey
	}
	e := lookupLiveEntry(key)
	if e == nil {
		return nil
	}
	anchors := e.adjAnchorsSnapshot()
	if len(anchors) == 0 {
		return nil
	}
	cfg, _ := readLiveState(key)
	read := func(file string) string {
		if data, err := os.ReadFile(filepath.Join(cfg.RepoDir, file)); err == nil {
			return string(data)
		}
		return ""
	}
	statuses := relocateAdjustments(anchors, read)
	var badges []h.H
	for i, s := range statuses {
		var cls, status string
		switch s.State {
		case adjSame:
			cls, status = "review-adjust__status--same", "still on line "+strconv.Itoa(s.Line)
		case adjMoved:
			cls, status = "review-adjust__status--moved", "addressed — moved to line "+strconv.Itoa(s.Line)
		default: // adjOutdated
			cls, status = "review-adjust__status--outdated", "addressed — line edited"
		}
		// Resolve button keyed on the ORIGINAL anchor file:line (anchors is 1:1 with
		// statuses by index). The datastar inline-assign sets the adj signals to THIS
		// badge's file:line then posts ResolveAdjustment — the maplibre/answer-flow bridge,
		// so the shared per-tab signals carry the right anchor regardless of form state.
		a := anchors[i]
		resolveExpr := "$adjfile=" + jsStr(a.file) + ";$adjline=" + jsStr(strconv.Itoa(a.line)) + ";@post('/_action/ResolveAdjustment')"
		badges = append(badges, h.Span(h.Class("review-adjust__status "+cls),
			h.Text(s.Comment+" — "+status),
			h.Button(h.Type("button"), h.Class("pk-btn review-adjust__resolve"),
				h.Data("on:click", resolveExpr), h.Text("resolve")),
		))
	}
	return badges
}

// jsStr renders s as a double-quoted JS string literal for a datastar expression, escaping
// the characters that would break out of the quotes (so a path with a quote/backslash
// can't corrupt the inline-assign).
func jsStr(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`, "\r", `\r`)
	return `"` + r.Replace(s) + `"`
}

// readSourceLine returns the 1-indexed content of file's line within repoDir, so the
// review turn can quote the line under comment. Best-effort: a missing file, an
// unreadable tree, or an out-of-range/non-positive line degrades to "" (no quote)
// rather than erroring — the adjustment still sends, it just isn't quoted.
func readSourceLine(repoDir, file string, line int) string {
	if file == "" || line < 1 {
		return ""
	}
	data, err := os.ReadFile(filepath.Join(repoDir, file))
	if err != nil {
		return ""
	}
	lines := strings.Split(string(data), "\n")
	if line > len(lines) {
		return ""
	}
	return strings.TrimRight(lines[line-1], "\r")
}
