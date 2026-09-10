package app

import (
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-via/via"
	"github.com/go-via/via/h"
	"github.com/go-via/via/on"

	"github.com/joaomdsg/packets/internal/fabric"
	"github.com/joaomdsg/packets/internal/ledger"
	"github.com/joaomdsg/packets/internal/pipe"
	"github.com/joaomdsg/packets/internal/reanchor"
	"github.com/joaomdsg/packets/internal/surface"
)

// CardRow is one session's line on the fleet board — a calm cross-card tally
// projected purely from that session's own log. It is ACTIVITY, never priority:
// the board orders rows by Queued (work awaiting drain) so the Lead sees where
// motion is, NOT a leverage rank (blocked-downstream is uncomputable today).
type CardRow struct {
	Key              string
	Confirmed        int
	Reinvested       int
	InFlight         int               // claims submitted but not yet minted — peers' pending BETS, never confirmed catches (two-scores)
	Rejected         int               // verified-lost: bets the host verified and found no catch — a RESOLVED loss, distinct from a pending in-flight bet and from a confirmed catch (two-scores)
	Sends            []ledger.SendView // this session's recent funded packets + their caught/missed outcome — honest per-packet round-trip legibility, never a fabricated rank
	Balance          int
	Queued           int
	Running          int
	Done             int
	Caught           int // done packets whose run minted a confirmed catch — the exact ledger.ScoutingReport count (gated on the SAME packet being done), the first-pass-hit numerator
	Misses           int // done packets that minted NOTHING (Done − Caught) — honest losses made visible, not silently discarded
	BacklogRemaining int
	OpenQuestions    int    // the session's latest-cycle open review questions (surviving mutants) — test debt the green verdict hides, made visible across the fleet; a diagnostic, never scored (off the economy)
	Land             string // the session's latest-cycle integration verdict (clean/conflict/checks_red) — surfaced on the board only when BLOCKED, so "Landed ≠ Merged" is visible across the fleet
	LandLifecycle    string // the opened PR's post-land lifecycle (§29.2: landed/merged/bounced) — surfaced on the fleet board only for the TERMINAL merged/bounced outcomes (the board stays calm on the routine landed-not-merged transient)
	Activity         string // the agent's latest live activity beat (e.g. "editing auth.go") while a packet fills — the cross-session "watch the shop" ticker; "" when idle, so an idle row stays calm
	seq              int    // registration ordinal — the deterministic tie-break, not rendered
}

// BoardRows projects one row per registered session by ranging liveReg, reading
// ONLY each session's own log projections (a ledger read failure degrades that
// field to zero, never breaks the board). Rows are ordered by Queued descending
// — the queued-awaiting-drain ACTIVITY signal — tie-broken by registration order
// (seq), so the order is deterministic across renders despite sync.Map's
// nondeterministic Range and the absence of any timestamp to sort by.
func BoardRows() []CardRow {
	var rows []CardRow
	liveReg.Range(func(k, v any) bool {
		e := v.(*liveEntry)
		row := CardRow{Key: k.(string), seq: e.seq}
		if e.log != nil {
			if recs, err := e.log.Records(); err == nil {
				st := ledger.ConfirmedCatches(recs)
				row.Confirmed, row.Reinvested = st.Count, st.Reinvested
			}
			if b, err := e.log.Balance(); err == nil {
				row.Balance = b
			}
			// Claims in flight are peers' pending bets, projected from the claim
			// subtree alone — kept off Confirmed/Balance (two-scores). Degrade to 0 on
			// a read error, like every other field.
			if n, err := e.log.ClaimsInFlight(); err == nil {
				row.InFlight = n
			}
			// Verified-losses (bets the host rejected) are the resolved counterpart
			// to in-flight bets, kept off Confirmed/Balance (two-scores). Degrade to
			// 0 on a read error, like every other field.
			if n, err := e.log.ClaimsRejected(); err == nil {
				row.Rejected = n
			}
			// Recent funded packets + their caught/missed outcome — the
			// round-trip made legible. Degrade to nil on a read error like the rest.
			if ds, err := e.log.RecentSends(5); err == nil {
				row.Sends = ds
			}
			if c, err := e.log.SendStatusCounts(); err == nil {
				row.Queued, row.Running, row.Done = c.Queued, c.Running, c.Done
			}
			// First-pass hits: the EXACT count of done packets whose own run minted a
			// catch (ledger.ScoutingReport gates Caught on the SAME packet being done, so
			// a catch on a still-running packet can't be misattributed). Misses are the
			// rest of the done packets — Caught ≤ Done by construction, so no clamp.
			if sr, err := e.log.ScoutingReport(); err == nil {
				row.Caught = sr.Caught
				row.Misses = row.Done - sr.Caught
			}
			row.BacklogRemaining = len(fundableBacklog(e.cfg, e.log))
		}
		// Open review questions are the session's latest-cycle surviving mutants, read
		// from the in-memory findings cache (not the log) — test debt the green verdict
		// hides, surfaced across the fleet. A diagnostic count, never scored.
		row.OpenQuestions = len(e.openFindings())
		// The latest integration verdict, read from the in-memory cache (not the log)
		// — surfaced on the board only when it blocks a merge (see View).
		row.Land = e.landState()
		// The opened PR's post-land lifecycle (§29.2) — also in-memory; surfaced on the
		// board only for terminal merged/bounced outcomes (see View + boardLifecycle).
		row.LandLifecycle = e.landLifecycleSnapshot()
		// The agent's live activity beat (in-process, read straight from the session's
		// fill buffer) — the cross-session ticker, surfaced only while a packet fills.
		row.Activity = e.activitySnapshot()
		rows = append(rows, row)
		return true
	})
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].Queued != rows[j].Queued {
			return rows[i].Queued > rows[j].Queued // most work awaiting drain first
		}
		return rows[i].seq < rows[j].seq // deterministic tie-break: earlier-registered first
	})
	return rows
}

// BoardCard is the cross-card FLEET surface: a calm row-per-session tally of the
// whole registry, ordered by queued ACTIVITY (see BoardRows). It re-projects
// liveReg on render and never labels a card by priority or leverage (the Lead sees
// where work is MOVING, not a fabricated importance rank). It also carries the one
// command the fleet view owns: creating a new session.
type BoardCard struct {
	// NewKey holds the key typed into the create-session input (two-way bound), read
	// by CreateSession on submit. Per-tab signal, not authoritative session state.
	NewKey via.SignalStr `via:"newkey"`
	// NewRepo holds the repo dir for the new session, read by CreateSession on submit.
	// It is the FULL absolute path chosen via the server-side directory browser (see
	// SelectRepo) — a browser file input can only ever yield a folder name, never an
	// absolute path, so the picker runs server-side where the real filesystem is
	// reachable. A session is usable with just a repo; empty inherits the server's repo.
	NewRepo via.SignalStr `via:"newrepo"`
	// BrowseDir is the absolute directory the filesystem picker is CURRENTLY showing —
	// tab-scoped server state (each tab browses independently). Empty means the picker
	// is closed; a non-empty value renders the browse panel. Written by OpenBrowser /
	// Browse / SelectRepo / CloseBrowser, read by View to render the panel.
	BrowseDir via.StateTabStr `via:"browsedir"`
	// BrowseTarget carries the directory a panel entry (a child folder, or the up
	// control) wants to move into — set by that entry (on.SetSignal) just before the
	// post, then read by Browse, which moves the picker there only if it's a real dir.
	BrowseTarget via.SignalStr `via:"browsetarget"`
	// RetireKey carries the key of the row whose retire button was clicked — set by
	// that button (on.SetSignal) just before the post, then read by RetireSession.
	RetireKey via.SignalStr `via:"retirekey"`
	// FleetTick is a re-render trigger written by OnConnect's Stream when the fleet
	// fingerprint changes (a session created/retired, counts moved). It is not rendered
	// — writing it marks the card dirty so View re-projects liveReg and the board
	// live-refreshes (see OnConnect).
	FleetTick via.StateTabStr
}

// boardRefreshInterval is how often the board polls the fleet for changes. A calm
// fleet view, not a hot path — a second is responsive without flooding SSE.
const boardRefreshInterval = time.Second

// OnConnect drives the board's live-refresh: it polls the fleet fingerprint each
// interval and, only when it changes, writes FleetTick to re-render — so a session
// another tab creates/retires (or a moving count) appears without a manual reload,
// without re-pushing an unchanged board every tick. The Stream goroutine stops with
// the connection (ctx disposal).
func (c *BoardCard) OnConnect(ctx *via.Ctx) error {
	last := fleetFingerprint(BoardRows())
	via.Stream(ctx, boardRefreshInterval, func(ctx *via.Ctx, _ time.Time) {
		if fp := fleetFingerprint(BoardRows()); fp != last {
			last = fp
			c.FleetTick.Write(ctx, fp)
		}
	})
	return nil
}

// fleetFingerprint folds the board's mutable per-row state into one string, so
// OnConnect can re-render only when the board would actually look different. It
// covers the fields View renders and that change over time (membership, stock,
// balance, the bet lifecycle, send activity, hit/miss, backlog, open questions,
// land, the terminal merge lifecycle, the live activity beat).
func fleetFingerprint(rows []CardRow) string {
	var b strings.Builder
	for _, r := range rows {
		b.WriteString(r.Key)
		for _, n := range []int{r.Confirmed, r.Reinvested, r.InFlight, r.Rejected,
			r.Balance, r.Queued, r.Running, r.Done, r.Caught, r.Misses,
			r.BacklogRemaining, r.OpenQuestions} {
			b.WriteByte(':')
			b.WriteString(strconv.Itoa(n))
		}
		b.WriteByte('|')
		b.WriteString(r.Land)
		b.WriteByte('|')
		// Fold only the DISPLAYED lifecycle state: terminal merged/bounced move the
		// fingerprint (board live-refreshes the badge), the hidden landed/""/unknown
		// transients do not (so an idle board never pushes a look-identical frame).
		if state, _, show := boardLifecycle(r.LandLifecycle); show {
			b.WriteString(state)
		}
		b.WriteByte('|')
		b.WriteString(r.Activity)
		b.WriteByte(';')
	}
	return b.String()
}

// RetireSession removes a session from the fleet view — the honest completion of
// CreateSession, so experiment sessions don't accumulate on the board. It unmounts
// the key from the registry (the ledger's events persist on the fabric; this only
// drops the live entry). The seeded default is NEVER retired — it is the "/" route's
// single-card fallback — and an empty/unknown key is a no-op.
//
// A retired key's durable claim consumer is owned by the session's socket, so
// retiring closes that socket (consumerSpawner.stopConsumer) to stop the
// consumer rather than leave it lingering on the fabric. The durable itself is
// not deleted (ConsumeDurable never deletes on teardown), so re-creating the
// same key resumes from the last ack; POST /claim still gates on liveReg.Load,
// so a retired key 404s and takes no new claims in the meantime.
func (c *BoardCard) RetireSession(ctx *via.Ctx) {
	key := strings.TrimSpace(c.RetireKey.Read(ctx))
	if key == "" || key == defaultSessionKey {
		return // never strand the default fallback
	}
	liveReg.Delete(key)
	consumerSpawner.stopConsumer(key) // stop the retired session's claim consumer, no longer left lingering
}

// CreateSession starts a new session economy from the fleet view: it registers the
// typed key (inheriting the default session's config) so the Lead can work it
// immediately via the in-process card flow — no boot edit, no claim consumer needed
// (consumers serve only the untrusted-peer POST /claim path). An invalid
// subject token or a key that already exists is an honest no-op: a create never
// forges a bad token nor clobbers a live economy's log. (Peer claims for a
// runtime-created session are unsupported in V1 — the card flow works fully.)
func (c *BoardCard) CreateSession(ctx *via.Ctx) {
	key := strings.TrimSpace(c.NewKey.Read(ctx))
	if key == "" || !fabric.ValidToken(key) {
		return // never forge an invalid subject token
	}
	if _, exists := liveReg.Load(key); exists {
		return // never clobber a live economy
	}
	cfg, _ := readLiveState(defaultSessionKey) // inherit the default config (same revs/test cmd)
	// A picked directory points the new session at any tree (a session is usable with
	// just a repo); a blank pick inherits the server's repo. The new session carries
	// no primary anchor — it is a prompt-first session the harness fills.
	// A URL pick is cloned on create (then the session points at the local clone); a
	// local path is used in place; a blank pick inherits the server's repo (§15.2).
	if repo := resolveOrCloneRepo(cfg.ReposRoot, c.NewRepo.Read(ctx)); repo != "" {
		cfg.RepoDir = repo
	}
	cfg.Anchor = reanchor.Anchor{} // prompt-first: no inherited anchor / catch-cycle
	cfg.BaseRev, cfg.FixRev, cfg.TipRev = "", "", ""
	slog, _ := AddSession(key, cfg) // validated above; a bind error leaves the registry unchanged
	// Seed starting attention bandwidth so the Lead can place a prompt packet
	// immediately — a prompt-first session has no anchored catch flow to earn bandwidth
	// from, so without this the place-packet control would never appear (chicken-and-egg).
	if slog != nil {
		seedStartingBandwidth(slog)
	}
	// Start the long-lived harness exploring the repo NOW, so this session's first
	// analyze/packet resumes a warm context (DESIGN §6's resumed-per-session harness).
	if e := lookupLiveEntry(key); e != nil && cfg.RepoDir != "" {
		startWarmHarness(e, cfg.RepoDir)
	}
}

// OpenBrowser opens the server-side filesystem picker, landing on a sensible start
// directory (see browseStart) derived from the default session's config. A browser
// file input can only hand back a folder name, never an absolute path — so the Lead
// navigates the real filesystem server-side, where the full path is knowable.
func (c *BoardCard) OpenBrowser(ctx *via.Ctx) {
	cfg, _ := readLiveState(defaultSessionKey)
	c.BrowseDir.Write(ctx, browseStart(cfg))
}

// Browse moves the picker into BrowseTarget (a child folder or the parent via the up
// control) — but ONLY if it is a real, existing directory. A loose file or a
// stale/forged target is ignored, so a bad signal can never strand the picker on a
// non-navigable location; it stays put on the last good directory.
func (c *BoardCard) Browse(ctx *via.Ctx) {
	c.BrowseDir.Write(ctx, nextBrowseDir(c.BrowseDir.Read(ctx), c.BrowseTarget.Read(ctx)))
}

// nextBrowseDir is Browse's pure decision: it returns where the picker should move
// given the directory it is currently showing and the requested target. A target
// that is a real, existing directory wins (cleaned to drop any . / .. segments);
// anything else — a loose file, a stale/forged path, or a blank — leaves the picker
// on the current directory, so a bad target can never strand it on a non-navigable
// location. The returned path is always absolute when the target is (panel entries
// carry absolute child paths), so the eventual selection is a full path, never a
// bare name.
func nextBrowseDir(current, target string) string {
	target = strings.TrimSpace(target)
	if target == "" {
		return current
	}
	if info, err := os.Stat(target); err == nil && info.IsDir() {
		return filepath.Clean(target)
	}
	return current
}

// SelectRepo commits the directory the picker is currently showing as the new
// session's repo — the FULL absolute path lands in NewRepo (the whole point of a
// server-side picker) — and closes the panel. A picker that somehow holds no dir is
// an honest no-op. CreateSession reads NewRepo on submit (resolveRepoDir passes an
// absolute path through as-is).
func (c *BoardCard) SelectRepo(ctx *via.Ctx) {
	dir := strings.TrimSpace(c.BrowseDir.Read(ctx))
	if dir == "" {
		return
	}
	c.NewRepo.Write(ctx, dir)
	c.BrowseDir.Write(ctx, "")
}

// CloseBrowser abandons the picker without choosing anything, so the Lead can back
// out of browsing and fall back to the server's repo (a blank pick) instead of being
// forced to keep whatever directory they were looking at.
func (c *BoardCard) CloseBrowser(ctx *via.Ctx) {
	c.BrowseDir.Write(ctx, "")
}

// repoBrowser renders the server-side filesystem picker panel when BrowseDir is set
// (empty → an empty fragment, so the panel is absent until OpenBrowser). The panel
// shows the directory currently being browsed, a select-this-folder commit, a close,
// an up control (absent only at the filesystem root, where parent == dir), and one
// clickable entry per child directory — each carrying its own absolute path into
// BrowseTarget so Browse descends/ascends to exactly that folder.
func (c *BoardCard) repoBrowser(dir string) h.H {
	if dir == "" {
		return nil // closed: nothing to render — the caller omits the panel
	}
	panel := []h.H{h.Class("board-create__browser"), h.Data("state", "browser"),
		h.Attr("role", "group"), h.Attr("aria-label", "repo directory browser"),
		h.Div(h.Class("board-create__browser-head"),
			h.Span(h.Class("board-create__browser-dir"), h.Text(dir)),
			h.Button(on.Click(c.SelectRepo), h.Class("pk-btn board-create__browser-select"), h.Text("select this folder")),
			h.Button(on.Click(c.CloseBrowser), h.Class("pk-btn pk-btn--quiet board-create__browser-close"), h.Text("close")),
		),
	}
	// The up control ascends to the parent — omitted only at the filesystem root, where
	// filepath.Dir returns the path unchanged (so there is nowhere higher to climb).
	if parent := filepath.Dir(dir); parent != dir {
		panel = append(panel, h.Button(
			on.Click(c.Browse, on.SetSignal(&c.BrowseTarget.Signal, parent)),
			h.Class("board-create__browser-up"), h.Text(".. (up)"),
		))
	}
	for _, name := range browseSubdirs(dir) {
		child := filepath.Join(dir, name)
		panel = append(panel, h.Button(
			on.Click(c.Browse, on.SetSignal(&c.BrowseTarget.Signal, child)),
			h.Class("board-create__browser-entry"), h.Text(name+"/"),
		))
	}
	return h.Div(panel...)
}

// resolveRepoDir turns the create form's repo pick into a session repo dir. A
// browser directory picker yields only the picked folder's NAME (never an absolute
// path), so a relative pick is joined under reposRoot — and only its final segment
// is used (filepath.Base), so a crafted "../foo" can never escape the root. A pick
// whose final segment is a dot-only traversal token (".", "..") names no folder —
// joining it would land ON the root or climb ABOVE it, so it is treated as blank.
// An empty pick returns "" (the caller inherits the server's repo); an absolute path
// (a power-user / programmatic value) is used as-is.
func resolveRepoDir(reposRoot, pick string) string {
	pick = strings.TrimSpace(pick)
	if pick == "" {
		return ""
	}
	if filepath.IsAbs(pick) {
		return pick
	}
	base := filepath.Base(pick)
	if base == "." || base == ".." {
		return "" // dot-only token names no folder — would land on or above the root
	}
	return filepath.Join(reposRoot, base)
}

// browseSubdirs lists the immediate child DIRECTORIES of dir, sorted by name — the
// navigable rungs of the create-form's filesystem picker. A repo is always a folder,
// so loose files are excluded (they can't be a session's tree). Entries are
// classified by their RESOLVED target (os.Stat follows symlinks), so a symlink
// pointing to a directory is listed as navigable — Leads commonly keep repos as
// symlinks — and this stays consistent with nextBrowseDir, which also follows
// symlinks when navigating. A symlink to a file or a broken link resolves to nothing
// navigable and is skipped. A missing/unreadable dir lists as empty rather than
// erroring: the panel simply shows nothing to descend into, so a stale or
// permission-denied path is a calm dead-end, never a crash.
func browseSubdirs(dir string) []string {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var dirs []string
	for _, e := range ents {
		// Stat (not e.IsDir) so a symlink is judged by its target, matching nextBrowseDir.
		if info, err := os.Stat(filepath.Join(dir, e.Name())); err == nil && info.IsDir() {
			dirs = append(dirs, e.Name())
		}
	}
	sort.Strings(dirs)
	return dirs
}

// browseStart picks the absolute directory the filesystem picker opens ON. The
// configured repos root is the natural home for board-created sessions, so it wins
// when set; failing that, an absolute server repo dir puts the Lead near the tree
// the server already works. A relative repo dir (the default "." config) is no
// meaningful filesystem anchor, so the start falls back to the Lead's home dir (or
// "/" if even that is unknown) — the picker must always open on an absolute path it
// can ascend out of, never a bare ".".
func browseStart(cfg LiveConfig) string {
	if cfg.ReposRoot != "" {
		return cfg.ReposRoot
	}
	if filepath.IsAbs(cfg.RepoDir) {
		return cfg.RepoDir
	}
	if home, err := os.UserHomeDir(); err == nil {
		return home
	}
	return "/"
}

// startingBandwidthSeeds is how many cleared attention intervals a new session is
// seeded with. Each interval clears instantly (latency ≈ 0 → the fast-clear bonus),
// so the meter starts well above the live-send cost and a few prompt packets can
// be placed before the Lead earns more by answering review questions.
const startingBandwidthSeeds = 3

// seedStartingBandwidth credits a new session with startingBandwidthSeeds cleared
// intervals (block→unblock pairs) so a prompt packet is fundable on first load —
// using only the public ledger primitives, so the events are real cleared-attention
// facts, just pre-seeded. A write error is best-effort: a session with no seeded
// bandwidth simply shows no place control until the Lead earns some.
func seedStartingBandwidth(log *ledger.Log) {
	now := time.Now()
	for i := 0; i < startingBandwidthSeeds; i++ {
		id := "seed-bandwidth-" + strconv.Itoa(i)
		if log.AppendBlock(id, now) != nil {
			return
		}
		if log.AppendUnblock(id, now) != nil {
			return
		}
	}
}

// hitRateLabel is the card's standing — the ONE honest progression number: Caught
// (packets whose own run minted a confirmed catch, the exact ledger.ScoutingReport
// count) over Done (resolved sent packets). A pure COUNT ratio of logged
// events, never an inferred probability or forecast, so it redeems against the
// mint/miss the Lead actually earned. Done==0 reads a calm "hit-rate 0/0" — a
// string ratio, never a divide-by-zero.
//
// No clamp is needed: ScoutingReport gates a hit on the SAME packet being done, so
// Caught ≤ Done by construction (a "wo:" catch on a still-running packet is not
// counted — the misattribution the old Reinvested-stock heuristic could leak).
func hitRateLabel(r CardRow) string {
	return "hit-rate " + strconv.Itoa(r.Caught) + "/" + strconv.Itoa(r.Done)
}

// View renders one row per registered session: its confirmed/reinvested stock,
// the peers' bet lifecycle (in-flight bets and verified-losses, each its own
// span, never folded into the confirmed stock — two-scores), spendable balance,
// queued/running/done activity, the distinct work still awaiting a spend, and the
// hit-rate standing. Calm spans in the stock idiom — no gauges, no priority, no
// forecast.
func (c *BoardCard) View(ctx *via.CtxR) h.H {
	// The fleet content is the page's main landmark (named for screen-reader
	// navigation), and a LIVE region: OnConnect re-renders it over SSE when the fleet
	// changes, so aria-live="polite" lets assistive tech announce a new/retired session
	// without the user hunting for it. The nav is a sibling landmark (added in the final
	// wrap), never nested inside main.
	// The fleet view's one command: start a new session economy. A calm input + a
	// repo picker + button, in the surface idiom — no modal, no menu. The picker panel
	// is appended only when open (BrowseDir set), so a nil panel is never embedded.
	createKids := []h.H{h.Class("board-create"),
		h.Input(h.Type("text"), c.NewKey.Bind(), h.Class("pk-input board-create__key"), h.Placeholder("new session key")),
		// The repo pick: the full absolute path chosen via the server-side directory
		// browser (a browser file input can only ever yield a folder NAME, never an
		// absolute path). The selected path shows here; blank inherits the server's
		// repo. The Browse control opens the picker (OpenBrowser → c.repoBrowser).
		h.Div(h.Class("board-create__repo"),
			h.Span(h.Class("pk-section-label"), h.Text("repo: ")),
			h.Span(h.Class("board-create__selected"), c.NewRepo.Text()),
			h.Button(on.Click(c.OpenBrowser), h.Class("pk-btn pk-btn--quiet board-create__browse"),
				h.Attr("aria-label", "browse for a repo directory (blank = server's repo)"), h.Text("Browse…")),
		),
	}
	if panel := c.repoBrowser(c.BrowseDir.Read(ctx)); panel != nil {
		createKids = append(createKids, panel)
	}
	createKids = append(createKids,
		h.Button(on.Click(c.CreateSession), h.Class("pk-btn board-create__btn"), h.Text("Track repo")))

	parts := []h.H{h.Class("board"), h.Data("state", "board"),
		h.Role("main"), h.Attr("aria-label", "fleet"), h.Attr("aria-live", "polite"),
		h.Div(createKids...),
	}
	rows := BoardRows()
	// A fleet-level merge-readiness roll-up: how many sessions are blocked from
	// landing, out of the whole fleet — a calm count of the same honest per-session
	// land verdicts, so a Lead sees fleet-wide merge friction without scanning every
	// row. Surfaced ONLY when ≥1 session is blocked (mirroring the per-row precedent),
	// so a fully-mergeable fleet stays calm. A diagnostic projection, off the economy;
	// never a fabricated rank.
	if blocked := blockedLandCount(rows); blocked > 0 {
		parts = append(parts, h.Span(
			h.Class("board__land-summary"),
			h.Text(strconv.Itoa(blocked)+" of "+strconv.Itoa(len(rows))+" addrs held from forwarding"),
		))
	}
	for _, r := range rows {
		row := []h.H{
			h.Class("pk-card board-row"),
			h.Data("key", r.Key),
			// The row key DRILLS into that session's card — the fleet board is not a
			// dead end. The default row links to /?key=default (explicit + honest). The
			// key is URL-escaped: fabric.ValidToken admits query metacharacters ('&',
			// '=', '#', '+'), which interpolated raw would split or truncate the query
			// and target the WRONG session — QueryEscape makes the link round-trip.
			h.A(h.Href("/?key="+url.QueryEscape(r.Key)), h.Class("board-row__key"), h.Text(r.Key)),
			h.Span(h.Class("board-row__stock"), h.Text(strconv.Itoa(r.Confirmed)+" confirmed, "+strconv.Itoa(r.Reinvested)+" reinvested")),
			// The peers' BET lifecycle, sealed into one explicitly-labelled
			// cluster so a pending/lost bet can't blend into the confirmed stock at a
			// glance — the two-scores separation carried by STRUCTURE, not by hoping a
			// reader parses each label. The inner spans keep their class hooks so a
			// future stylesheet can color bets muted-vs-solid with no server change.
			h.Div(h.Class("board-row__bets"),
				h.Span(h.Class("pk-section-label board-row__bets-label"), h.Text("claims:")),
				h.Span(h.Class("board-row__inflight"), h.Text(strconv.Itoa(r.InFlight)+" in flight")),
				h.Span(h.Class("board-row__rejected"), h.Text(strconv.Itoa(r.Rejected)+" verified-lost")),
			),
			h.Span(h.Class("board-row__balance"), h.Text("catches "+strconv.Itoa(r.Balance))),
			h.Span(h.Class("board-row__activity"), h.Text("queued "+strconv.Itoa(r.Queued)+", running "+strconv.Itoa(r.Running)+", done "+strconv.Itoa(r.Done))),
			h.Span(h.Class("board-row__misses"), h.Text(strconv.Itoa(r.Misses)+" misses")),
			h.Span(h.Class("board-row__hitrate"), h.Text(hitRateLabel(r))),
			h.Span(h.Class("board-row__backlog"), h.Text(strconv.Itoa(r.BacklogRemaining)+" awaiting")),
		}
		// Open review questions (surviving mutants) — surfaced only when there ARE any,
		// so a session carrying test debt the green verdict hides stands out at a glance
		// without nagging the clean ones. Links into that session's /review surface.
		if r.OpenQuestions > 0 {
			row = append(row, h.A(
				h.Href("/review?key="+url.QueryEscape(r.Key)),
				h.Class("board-row__questions"),
				h.Text(strconv.Itoa(r.OpenQuestions)+" open questions"),
			))
		}
		// Integration verdict — surfaced ONLY when it BLOCKS a merge (conflict /
		// checks-red), so a session that can't land stands out across the fleet
		// ("Landed ≠ Merged"). A clean or not-yet-resolved verdict shows nothing
		// (the board stays calm). Honest color via the data-state hook (the honest state palette).
		if state, label, blocked := boardLand(r.Land); blocked {
			row = append(row, h.Span(
				h.Class("board-row__land"),
				h.Data("state", state),
				h.Text(label),
			))
		}
		// Post-open lifecycle across the fleet (§29.2: Landed ≠ Merged) — surfaced only for
		// the terminal merged/bounced outcomes, so a finished or closed-unmerged PR stands
		// out while the routine landed-not-merged transient keeps the board calm.
		if state, label, show := boardLifecycle(r.LandLifecycle); show {
			row = append(row, h.Span(
				h.Class("board-row__lifecycle"),
				h.Data("state", state),
				h.Text(label),
			))
		}
		// What the agent is doing RIGHT NOW — a calm dim ticker, surfaced only while a
		// packet fills (a beat exists), so the Lead watches the shop across sessions
		// without opening each card. Absent on an idle row (no dead "·").
		if r.Activity != "" {
			row = append(row, h.Span(h.Class("board-row__activity-beat"), h.Text("· "+r.Activity)))
		}
		// The funded packet round-trip made legible: recent sends with their
		// caught/missed outcome, in their own cluster (omitted when there are none).
		// Honest per-packet outcomes, never a fabricated rank.
		if d := renderSends(r.Key, r.Sends); d != nil {
			row = append(row, d)
		}
		// A retire control on every NON-default row — the default is the "/" route's
		// fallback and is never retirable. The button sets retirekey to THIS row's key
		// (on.SetSignal) just before the post, so RetireSession removes the right one.
		if r.Key != defaultSessionKey {
			row = append(row, h.Button(
				on.Click(c.RetireSession, on.SetSignal(&c.RetireKey.Signal, r.Key)),
				h.Class("pk-btn pk-btn--quiet board-row__retire"), h.Text("retire"),
			))
		}
		parts = append(parts, h.Div(row...))
	}
	// nav landmark first, then the main fleet region — distinct sibling landmarks.
	return h.Div(navHeader("", "console"), h.Div(parts...))
}

// boardLand maps a session's raw integration verdict to the board's (data-state,
// label, blocked) — blocked is true only for verdicts that STOP a merge (conflict /
// checks-red), so the board surfaces them and stays silent on clean/pending. The
// data-state mirrors the surface land-states so the honest state palette colors it.
func boardLand(land string) (state, label string, blocked bool) {
	switch pipe.LandState(land) {
	case pipe.LandConflict:
		return "land-conflict", "held: rebase", true
	case pipe.LandChecksRed:
		return "land-checks-red", "held: checks red", true
	default: // clean / pending / unknown — nothing blocking to surface
		return "", "", false
	}
}

// boardLifecycle maps a session's opened-PR lifecycle (§29.2 "Landed ≠ Merged") to the
// board's (data-state, label, show). It surfaces only TERMINAL outcomes — merged or
// bounced — so the fleet view flags a finished (or closed-unmerged) PR while staying calm
// on the routine "landed, not yet merged" transient and on anything unrecognized. The
// data-state hooks the honest state palette (confirmed-green merged / miss-hue bounced).
func boardLifecycle(lc string) (state, label string, show bool) {
	switch landLifecycle(lc) {
	case lifecycleMerged:
		return "merged", "forwarded", true
	case lifecycleBounced:
		return "bounced", "closed, not forwarded", true
	default: // landed (routine transient) / empty / unknown — nothing terminal to surface
		return "", "", false
	}
}

// blockedLandCount counts how many rows carry a land verdict that blocks a merge
// (conflict / checks-red) — the numerator of the fleet's merge-readiness summary,
// using the SAME boardLand classification the per-row spans use so the roll-up and
// the rows can't disagree.
func blockedLandCount(rows []CardRow) int {
	n := 0
	for _, r := range rows {
		if _, _, blocked := boardLand(r.Land); blocked {
			n++
		}
	}
	return n
}

// renderSends renders a session's recent packets as a calm cluster —
// one span per packet: "PKT#<id> <path>:<line> <status>[ caught|missed]". The
// caught/missed outcome is shown only for a done packet (a queued/running packet
// has no outcome yet). Returns nil when there are none, so the cluster is omitted.
func renderSends(key string, views []ledger.SendView) h.H {
	if len(views) == 0 {
		return nil
	}
	spans := []h.H{h.Class("board-row__sends"), h.Span(h.Class("pk-section-label board-row__sends-label"), h.Text("sends:"))}
	for _, v := range views {
		text := "PKT#" + strconv.Itoa(v.ID) + " " + v.Target.Path + ":" + strconv.Itoa(v.Target.Line) + " " + v.Status
		span := []h.H{h.Class("pk-chip board-row__send")}
		// A resolved packet carries its outcome as a hook so the calm palette can
		// color caught vs missed at a glance (a queued/running packet has no outcome
		// yet, so no hook — it stays neutral).
		if v.Status == "done" {
			if v.Caught {
				text += " caught"
				span = append(span, h.Data("outcome", "caught"))
			} else {
				text += " missed"
				span = append(span, h.Data("outcome", "missed"))
			}
		}
		span = append(span, h.Text(text))
		// The oracle's verdict for a resolved packet — the WHY behind a catch/miss
		// (no-catch vs lost-via-rename vs no-oracle-signal …) — as a calm secondary
		// detail. Omitted when none is persisted (never an empty "why" tag).
		if v.Status == "done" && v.Verdict != "" {
			span = append(span, h.Span(h.Class("board-row__send-why"), h.Text(" "+surface.VerdictLabel(v.Verdict))))
		}
		// A DRILL link into the packet's review (/review?wo=<id>), the send→review
		// tie. A filled packet that left surviving mutants frames it as its open-question
		// count (the reviewable test-debt). A SETTLED packet that left none still links —
		// to inspect the peer's edits (the base→fix diff) — so a clean fill (caught
		// or missed) is never a dead end. Queued/running packets have produced no diff
		// yet, so they carry no link. A calm accent, never an alarm.
		href := "/review?key=" + url.QueryEscape(key) + "&wo=" + strconv.Itoa(v.ID)
		switch {
		case v.Questions > 0:
			span = append(span, h.A(
				h.Href(href),
				h.Class("board-row__send-questions"),
				h.Text(" • "+strconv.Itoa(v.Questions)+" open questions"),
			))
		case v.Status == "done":
			span = append(span, h.A(
				h.Href(href),
				h.Class("board-row__send-inspect"),
				h.Text(" • inspect diffs"),
			))
		}
		spans = append(spans, h.Span(span...))
	}
	return h.Div(spans...)
}
