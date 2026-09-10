package app

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/go-via/via"
	"github.com/go-via/via/h"
	"github.com/go-via/via/on"

	"github.com/joaomdsg/packets/internal/ledger"
	"github.com/joaomdsg/packets/internal/pipe"
	"github.com/joaomdsg/packets/internal/settle"
)

// openPR is the seam the approve flow opens a PR through (process I/O — verified by
// build + manual run, like runHarness; tests swap it for a scripted reply). It pushes the
// session's branch (leased against expected) and opens a PR, returning the PR URL and the
// SHA it pushed (cached so the next re-land leases against it).
var openPR = realOpenPR

// squashToCommit collapses the whole baseRev→HEAD range into a SINGLE commit object
// with HEAD's tree and baseRev as its sole parent, returning the new commit's SHA. It
// uses `git commit-tree`, which builds a detached commit object — it moves no local ref
// and never touches the working tree, so the session repo is left exactly as it was.
// The opened PR can then push this one squashed commit instead of every session
// revision (DESIGN §16). An empty range (HEAD == baseRev) is benign: it yields a commit
// one ahead of base carrying base's own tree (the land guard refuses empty work
// upstream). A bad baseRev surfaces as an error rather than a bogus commit.
func squashToCommit(ctx context.Context, repoDir, baseRev, message string) (string, error) {
	c := exec.CommandContext(ctx, "git", "-C", repoDir, "commit-tree", "HEAD^{tree}", "-p", baseRev, "-m", message)
	out, err := c.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("commit-tree: %v: %s", err, strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

// pushRefspec builds the args after `git push` for landing a session's squashed commit,
// LEASING the force to an explicit expectation instead of the bare --force-with-lease
// (which, on a per-session branch with no remote-tracking ref, degrades to an unguarded
// clobber). An empty expected leases against "must not exist" (the first push creates the
// branch); a non-empty expected is the SHA we last pushed (a re-land replaces the branch
// only if it's still there — refusing if something else moved it). The explicit-SHA lease
// form needs no remote-tracking ref, so it works against a freshly-cloned session repo.
func pushRefspec(branch, sha, expected string) []string {
	return []string{
		"--force-with-lease=refs/heads/" + branch + ":" + expected,
		"origin",
		sha + ":refs/heads/" + branch,
	}
}

// pushSquash pushes the squashed commit sha to branch on origin under the leased force
// (pushRefspec). A rejected lease (stale expectation, or a branch that already exists on a
// first push) surfaces as an error rather than a silent clobber.
func pushSquash(ctx context.Context, repoDir, sha, branch, expected string) error {
	push := exec.CommandContext(ctx, "git", append([]string{"push"}, pushRefspec(branch, sha, expected)...)...)
	push.Dir = repoDir
	if out, err := push.CombinedOutput(); err != nil {
		return fmt.Errorf("push: %v: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// formatSecretRefusal builds the pre-push gate's refusal message naming EVERY detected
// secret (in scan order) — so a Lead whose push carries several secrets fixes them all in
// one pass instead of one re-land per secret. Empty hits yield "" (the caller only calls
// it with hits, but it stays total).
func formatSecretRefusal(hits []settle.SecretHit) string {
	if len(hits) == 0 {
		return ""
	}
	noun := "secret"
	if len(hits) > 1 {
		noun = "secrets"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "refusing to push: %d %s detected:", len(hits), noun)
	for _, h := range hits {
		fmt.Fprintf(&b, "\n  %s:%d (%s)", h.File, h.Line, h.Rule)
	}
	return b.String()
}

// isPRAlreadyExists reports whether a `gh pr create` failure is the benign re-land signal
// "a PR already exists for this branch" (gh 2.92 emits two shapes — a branch-form message
// and a GraphQL-form one) rather than a real failure (auth, network, validation). It
// requires BOTH "pull request" and "already exists" (case-insensitive) so an unrelated
// "already exists" error (e.g. a label) does not over-match.
func isPRAlreadyExists(ghOutput string) bool {
	out := strings.ToLower(ghOutput)
	return strings.Contains(out, "pull request") && strings.Contains(out, "already exists")
}

// ghPRViewURL returns the URL of the open PR for branch, so a re-land (whose push already
// updated the open PR) can surface the existing PR instead of failing. I/O seam, verified
// by build + manual run like realOpenPR's other gh calls.
func ghPRViewURL(ctx context.Context, repoDir, branch string) (string, error) {
	c := exec.CommandContext(ctx, "gh", "pr", "view", branch, "--json", "url", "-q", ".url")
	c.Dir = repoDir
	out, err := c.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("gh pr view: %v: %s", err, strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

// realOpenPR squashes the session repo's baseRev→HEAD range into one commit, pushes that
// single commit to branch, and opens a PR against the default branch via gh, returning
// the PR URL it prints. v1 pushes a direct branch + PR (the merge-queue path, DESIGN
// §29.2, is deferred). Auth: in production a short-lived token is injected into the
// environment at land time (the agntpr x-access-token pattern) — never a long-lived
// host credential.
func realOpenPR(ctx context.Context, repoDir, baseRev, branch, title, body, expected string) (string, string, error) {
	sha, err := squashToCommit(ctx, repoDir, baseRev, title+"\n\n"+body)
	if err != nil {
		return "", "", err
	}
	// Pre-push secret gate: the squashed commit was built blindly from HEAD's tree and
	// never re-scanned, so this is the last point before the bytes leave the machine.
	// Scan precisely what's pushed (baseRev..sha) and refuse rather than leak (RISKS.md
	// secret-leakage is CRITICAL). Scope is the pushed change only — a secret already in
	// base, or one buried in abandoned history, does not block.
	if hits, err := settle.ScanCommitRange(ctx, repoDir, baseRev, sha); err != nil {
		return "", "", fmt.Errorf("secret scan: %w", err)
	} else if len(hits) > 0 {
		return "", "", fmt.Errorf("%s", formatSecretRefusal(hits))
	}
	// Push under a leased force (pushSquash): expected=="" creates the branch on the first
	// land, a cached SHA lets a re-land replace it only if nothing else moved it.
	if err := pushSquash(ctx, repoDir, sha, branch, expected); err != nil {
		return "", "", err
	}
	pr := exec.CommandContext(ctx, "gh", "pr", "create", "--head", branch, "--title", title, "--body", body)
	pr.Dir = repoDir
	out, err := pr.CombinedOutput()
	if err != nil {
		// A re-land: the push already updated the open PR, so a "already exists" create
		// failure is success — surface the existing PR's URL instead of a spurious error.
		if isPRAlreadyExists(string(out)) {
			url, verr := ghPRViewURL(ctx, repoDir, branch)
			if verr != nil {
				return "", sha, verr // the push landed — return the SHA so the caller caches it
			}
			return url, sha, nil
		}
		// The push already landed sha on the branch; return it (even on this failure) so the
		// caller caches it and the next re-land leases against the SHA actually on the remote.
		return "", sha, fmt.Errorf("gh pr create: %v: %s", err, strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), sha, nil
}

// Approve closes the goal flow: it opens a PR from the session's work (DESIGN §16).
// First it guards (landBlocked): an unmergeable tree (rebase conflict / red checks) or
// open review threads refuses the PR unless the Lead overrides (deliberate, overridable
// friction). Cleared, it derives the branch + PR title/body and opens the PR through
// the openPR seam, caching the URL (or a calm failure / guard message) for the card.
func (c *LiveCard) Approve(ctx *via.Ctx) {
	key := c.Key
	if key == "" {
		key = defaultSessionKey
	}
	cfg, log := readLiveState(key)
	if log == nil {
		return
	}
	e := lookupLiveEntry(key)
	if e == nil {
		return
	}
	// One land at a time per session: a land pushes to the session's shared branch
	// and opens a PR, so a double-clicked "forward →" would race those operations.
	// Drop the duplicate (the in-flight one is already landing).
	if !e.beginLand() {
		return
	}
	defer e.endLand()
	land := pipe.LandState(e.landState())
	override := landOverridden(c.LandOverride.Read(ctx))
	if blocked, reason := landBlocked(len(sessionOpenThreads(key)), land); blocked && !override {
		setLandResult(key, "blocked — "+reason)
		setLandLifecycle(key, "") // no PR → no lifecycle badge
		c.Landed.Write(ctx, "blocked")
		return
	}
	title, body := prTitleAndBody(key, latestSendPrompt(log))
	// Lease the push against the SHA we last pushed for this session (""=first land →
	// must-not-exist), so a re-land updates the branch only if nothing else moved it.
	expected := e.lastPushedSHASnapshot()
	// A prompt-first session has no configured base; derive its origin from the packet
	// history so the squash parents onto where the work actually started (not "").
	pkts, _ := log.Packets()
	baseRev := landBaseRev(cfg.BaseRev, pkts)
	url, pushedSHA, err := openPR(context.Background(), cfg.RepoDir, baseRev, prBranchName(key), title, body, expected)
	// Cache the pushed SHA the moment the push lands (pushedSHA non-empty) — independent of
	// whether the PR-open step then failed. Otherwise a gh failure after a successful push
	// would leave the cache stale while the remote branch advanced, wedging the next
	// re-land's lease (it would name the old SHA against a branch that moved).
	if pushedSHA != "" {
		e.setLastPushedSHA(pushedSHA)
	}
	if err != nil {
		setLandResult(key, "forward failed — "+err.Error())
		setLandLifecycle(key, "") // no opened PR → no lifecycle badge
		c.Landed.Write(ctx, "error")
		return
	}
	setLandResult(key, url)
	// A freshly opened PR is LANDED, not yet merged (DESIGN §29.2) — surface that
	// immediately; CheckMergeState later refreshes it to merged/bounced.
	setLandLifecycle(key, string(lifecycleLanded))
	c.Landed.Write(ctx, "opened")
}

// landBaseRev resolves the base the land squash parents onto. A legacy anchored session
// carries an explicit configured base and lands against THAT. A prompt-first session has
// none (board.go zeroes the revs), so passing it straight to commit-tree would fail
// ("not a valid object name" on an empty parent); instead the base is the session's
// ORIGIN — the earliest packet's recorded base (the repo HEAD before any harness commit,
// an ancestor of the current HEAD) — so the squash collapses ALL the session's work onto
// where it actually started. Empty early bases are skipped; "" means none is derivable
// (the caller then surfaces an honest error rather than fabricating a parentless commit).
func landBaseRev(cfgBaseRev string, pkts []ledger.PacketRecord) string {
	if cfgBaseRev != "" {
		return cfgBaseRev
	}
	for _, o := range pkts {
		if o.Target.BaseRev != "" {
			return o.Target.BaseRev
		}
	}
	return ""
}

// landOverridden reads the override signal tolerantly ("1" or "true").
func landOverridden(v string) bool {
	v = strings.TrimSpace(v)
	return v == "1" || v == "true"
}

// setLandResult caches the approve outcome on the session entry (no-op if unknown).
func setLandResult(key, res string) {
	if e := lookupLiveEntry(key); e != nil {
		e.setLandResult(res)
	}
}

// latestSendPrompt returns the most recent sent packet's prompt — the task
// the PR title/body summarizes. Empty when there are no sends.
func latestSendPrompt(log *ledger.Log) string {
	if log == nil {
		return ""
	}
	views, err := log.RecentSends(1)
	if err != nil || len(views) == 0 {
		return ""
	}
	return views[0].Target.Prompt
}

// landResultKind classifies a cached land-result string into the outcome the land
// control renders — so success (a clickable PR link), a guard block, and a push failure
// are each visually distinct rather than one undifferentiated mono blob.
type landResultKind int

const (
	landResultNone    landResultKind = iota // no result yet — render nothing
	landResultOpened                        // a PR opened — the value is its URL
	landResultBlocked                       // the land guard refused (open threads / red checks)
	landResultError                         // the push/PR failed, or an unrecognized message
)

// classifyLandResult maps a cached land-result string (shapes fixed by setLandResult's
// call sites) to its kind, returning the PR URL only for landOpened. An http(s):// value
// is the opened PR — checked FIRST so a URL whose path contains a keyword is never
// mistaken for a guard. Any unrecognized non-empty value is treated as an error, never a
// clickable success.
func classifyLandResult(res string) (landResultKind, string) {
	switch {
	case res == "":
		return landResultNone, ""
	case strings.HasPrefix(res, "http://"), strings.HasPrefix(res, "https://"):
		return landResultOpened, res
	case strings.HasPrefix(res, "blocked — "):
		return landResultBlocked, ""
	default:
		return landResultError, ""
	}
}

// landLifecycle is where an OPENED PR sits in the post-land lifecycle (DESIGN §29.2:
// "Landed ≠ Merged" — opening a PR is not merging it). Rendered as a badge on the land
// control so the Lead never reads "PR opened" as "merged/done".
type landLifecycle string

const (
	lifecycleLanded  landLifecycle = "landed"  // PR open, NOT yet merged
	lifecycleMerged  landLifecycle = "merged"  // the PR was merged into trunk
	lifecycleBounced landLifecycle = "bounced" // the PR was closed without merging
)

// classifyLandLifecycle maps a gh PR `state` string to the post-open lifecycle, failing
// CLOSED: only a definitive "MERGED" claims Merged; anything we can't confirm (unknown or
// empty) reads as the conservative not-yet-merged Landed — never a false Merged (mirrors
// classifyLandResult's defensive default). Case-insensitive and whitespace-trimmed.
func classifyLandLifecycle(prState string) landLifecycle {
	switch strings.ToUpper(strings.TrimSpace(prState)) {
	case "MERGED":
		return lifecycleMerged
	case "CLOSED":
		return lifecycleBounced
	default: // "OPEN", unknown, or empty → not yet merged
		return lifecycleLanded
	}
}

// mergeState is the seam the merge-lifecycle check reads a PR's state through (process I/O
// — verified by build + manual run, like openPR; tests swap it). It returns the gh PR
// `state` string ("OPEN"/"MERGED"/"CLOSED").
var mergeState = realMergeState

// realMergeState reads the open PR's state for branch via gh, so CheckMergeState can show
// whether it's merged yet. I/O seam, verified by build + manual run like ghPRViewURL.
func realMergeState(ctx context.Context, repoDir, branch string) (string, error) {
	c := exec.CommandContext(ctx, "gh", "pr", "view", branch, "--json", "state", "-q", ".state")
	c.Dir = repoDir
	out, err := c.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("gh pr view state: %v: %s", err, strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

// CheckMergeState refreshes the opened PR's merge lifecycle (DESIGN §29.2) — the Lead
// asking "has it merged yet?". Only meaningful once a PR was opened; a seam error is a
// calm no-op (the prior lifecycle stands). Off the economy ledger.
func (c *LiveCard) CheckMergeState(ctx *via.Ctx) {
	key := c.Key
	if key == "" {
		key = defaultSessionKey
	}
	if kind, _ := classifyLandResult(landResultSnapshot(key)); kind != landResultOpened {
		return // nothing opened to check
	}
	cfg, _ := readLiveState(key)
	state, err := mergeState(context.Background(), cfg.RepoDir, prBranchName(key))
	if err != nil {
		return // transient — leave the prior lifecycle, never a false claim
	}
	if e := lookupLiveEntry(key); e != nil {
		e.setLandLifecycle(string(classifyLandLifecycle(state)))
	}
}

// setLandLifecycle caches the opened PR's lifecycle on the session entry (no-op if unknown).
func setLandLifecycle(key, lc string) {
	if e := lookupLiveEntry(key); e != nil {
		e.setLandLifecycle(lc)
	}
}

// renderLandControl renders the approve-and-open-PR control: a button wired to
// Approve, an override toggle (to push past the guard deliberately), and the last
// approve outcome (the opened PR URL, a guard message, or a failure) when present.
func renderLandControl(c *LiveCard) h.H {
	key := c.Key
	if key == "" {
		key = defaultSessionKey
	}
	parts := []h.H{
		h.Class("land-control"),
		h.Button(on.Click(c.Approve), h.Class("pk-btn land-control__approve"), h.Text("forward →")),
		h.Label(h.Class("land-control__override"),
			h.Input(h.Type("checkbox"), c.LandOverride.Bind()),
			h.Span(h.Text("override guard")),
		),
	}
	res := landResultSnapshot(key)
	switch kind, url := classifyLandResult(res); kind {
	case landResultOpened:
		// The finish line: a clickable link to the PR the Lead just opened.
		parts = append(parts, h.A(h.Href(url), h.Attr("target", "_blank"), h.Rel("noopener"),
			h.Class("land-control__result land-control__result--ok"), h.Text(url)))
	case landResultBlocked:
		parts = append(parts, h.Span(h.Class("land-control__result land-control__result--blocked"),
			h.Text(res)))
	case landResultError:
		parts = append(parts, h.Span(h.Class("land-control__result land-control__result--error"),
			h.Text(res)))
	}
	// Post-open lifecycle badge (DESIGN §29.2: Landed ≠ Merged). Shown only once a PR was
	// opened (lifecycle cache non-empty); a "check merge state" button refreshes it.
	if lc := landLifecycleSnapshot(key); lc != "" {
		var cls, text string
		switch landLifecycle(lc) {
		case lifecycleMerged:
			cls, text = "land-control__lifecycle--merged", "forwarded"
		case lifecycleBounced:
			cls, text = "land-control__lifecycle--bounced", "closed, not forwarded"
		default: // lifecycleLanded
			cls, text = "land-control__lifecycle--landed", "opened — not yet forwarded"
		}
		parts = append(parts,
			h.Span(h.Class("land-control__lifecycle "+cls), h.Text(text)),
			h.Button(on.Click(c.CheckMergeState), h.Class("pk-btn land-control__check-merge"),
				h.Text("check forward status")),
		)
	}
	return h.Div(parts...)
}

// landLifecycleSnapshot returns the cached opened-PR lifecycle for a session ("" if none).
func landLifecycleSnapshot(key string) string {
	if e := lookupLiveEntry(key); e != nil {
		return e.landLifecycleSnapshot()
	}
	return ""
}

// sessionHasSends reports whether the session has at least one sent packet —
// the landable work the approve flow opens a PR for.
func sessionHasSends(log *ledger.Log) bool {
	if log == nil {
		return false
	}
	v, err := log.RecentSends(1)
	return err == nil && len(v) > 0
}

// landResultSnapshot returns the cached approve outcome for a session ("" if none).
func landResultSnapshot(key string) string {
	if e := lookupLiveEntry(key); e != nil {
		return e.landResultSnapshot()
	}
	return ""
}
