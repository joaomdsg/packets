package app

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-via/via"
	"github.com/go-via/via/h"
	"github.com/go-via/via/on"

	"github.com/joaomdsg/packets/internal/bridge"
	"github.com/joaomdsg/packets/internal/catch"
	"github.com/joaomdsg/packets/internal/fabric"
	"github.com/joaomdsg/packets/internal/harness"
	"github.com/joaomdsg/packets/internal/ingest"
	"github.com/joaomdsg/packets/internal/ledger"
	"github.com/joaomdsg/packets/internal/mutation"
	"github.com/joaomdsg/packets/internal/packet"
	"github.com/joaomdsg/packets/internal/pipe"
	"github.com/joaomdsg/packets/internal/reanchor"
	"github.com/joaomdsg/packets/internal/review"
	"github.com/joaomdsg/packets/internal/socket"
	"github.com/joaomdsg/packets/internal/surface"
	"github.com/joaomdsg/packets/internal/tokenstore"
	"github.com/joaomdsg/packets/internal/translate"
)

// LiveConfig is the single catch cycle the live server drives: the two
// revisions, the anchored line, how to run the suite, and the mint-time bits.
type LiveConfig struct {
	RepoDir          string
	BaseRev          string
	FixRev           string
	TipRev           string
	Anchor           reanchor.Anchor
	TestCmd          []string
	LedgerPath       string
	SelfFlagged      bool
	WouldHaveShipped bool
	// MaxConcurrent caps how many catch cycles run at once (each cycle is several
	// full-suite executions — see internal/pipe and the #15 benchmark). Connects
	// beyond the cap QUEUE on a slot, they are never dropped. 0 means unbounded.
	MaxConcurrent int
	// SendBacklog is the ordered supply of DISTINCT work a card's Spends draw
	// down — the rev/anchor triple each funded packet runs. A Spend consumes the next
	// not-yet-funded target head-first; an empty or fully-drawn-down backlog makes a
	// Spend a silent no-op (the honest scarcity signal — no distinct work to buy).
	SendBacklog []ledger.Target
	// UseContainer, when true, runs this session's LIVE packets (Target.Prompt set) in
	// the hardened agent container (harness.RunContainer) instead of the host
	// subprocess (harness.RunProcess). The firewall is unchanged — both produce a
	// revision the host settles; only WHERE the agent runs differs.
	UseContainer bool
	// ListenAddr, when non-empty, binds the shared fabric to an AUTHENTICATED TCP
	// NATS listener (host:port; port 0 picks a free port) so cross-process PEERS
	// submit claims as authenticated clients, each confined by a Grant to its own
	// session's claim subtree. Empty keeps the fabric in-process-only (the default —
	// tests and single-process runs need no socket and no auth).
	ListenAddr string
	// Grants authorizes the cross-process peers allowed on ListenAddr. Each
	// grant's credentials may publish ONLY to its session's claim subtree and can
	// never mint — the in-process host stays the single minter. Ignored when
	// ListenAddr is empty. Build with NewGrant.
	Grants []fabric.Grant
	// ReposRoot is the parent directory under which board-created sessions resolve a
	// picked repo. A browser directory picker yields only the folder NAME (never an
	// absolute path), so CreateSession joins it under this root. Empty means a relative
	// pick resolves against the server's working dir; an absolute pick is always used
	// as-is. See resolveRepoDir.
	ReposRoot string
}

// hasRepo reports whether this config names a repo the session can work — enough to
// be USABLE: the Lead authors prompt packets and the harness fills them against the
// repo. A session with a repo renders the working card (no anchor required); only a
// repo-less config falls back to the "No session configured" landing.
func (c LiveConfig) hasRepo() bool {
	return c.RepoDir != ""
}

// hasAnchor reports whether this config names a primary anchor — a base revision +
// an anchored file — to run the legacy connect catch-cycle on. A repo-only session
// leaves it false, so OnConnect runs no cycle (no phantom Oracle-running spinner).
func (c LiveConfig) hasAnchor() bool {
	return c.BaseRev != "" && c.Anchor.Path != ""
}

// bundleAuthorized reports whether the request carries HTTP Basic credentials
// matching a grant for session key (peer == session key). The password is
// compared in constant time so a prober cannot time-recover it; the user/session
// equality checks are not secret and need no such guard.
func bundleAuthorized(grants []fabric.Grant, key string, r *http.Request) bool {
	user, pass, ok := r.BasicAuth()
	if !ok {
		return false
	}
	for _, g := range grants {
		if g.Session == key && g.User == user &&
			subtle.ConstantTimeCompare([]byte(g.Pass), []byte(pass)) == 1 {
			return true
		}
	}
	return false
}

// NewGrant builds a peer authorization for the live server: the
// credentials may publish claims ONLY to sessionKey's claim subtree, bound to
// the one instance every economy uses (LedgerInstance), and may never mint. It
// is the sanctioned constructor so callers (e.g. cmd/packets) need not know the
// internal instance token.
func NewGrant(sessionKey, user, pass string) fabric.Grant {
	return fabric.Grant{User: user, Pass: pass, Session: sessionKey, Instance: LedgerInstance}
}

// resolveCycle is the seam OnConnect runs the catch cycle through. It defaults to
// the real ResolveStreaming; tests swap it to drive the admission cap
// deterministically without spinning up real oracle work.
var resolveCycle = ResolveStreaming

// runHarness is the seam a LIVE packet runs its agent through. It defaults to
// the real harness.RunProcess (spawns claude, reduces its stream-json into
// settled revisions); tests swap it for a scripted stub so the live-fill routing
// is exercised without a claude binary or API key.
var runHarness = harness.RunProcess

// runHarnessContainer is the seam a UseContainer live packet runs its agent through
// — the hardened agent container (harness.RunContainer), same signature as
// runHarness. Tests swap it.
var runHarnessContainer = harness.RunContainer

// liveEntry is one session's wiring: the cycle config, its ledger, and its
// admission semaphore (a buffered channel of size cfg.MaxConcurrent, or nil when
// uncapped — a send acquires a cycle slot, a receive releases it).
type liveEntry struct {
	cfg LiveConfig
	log *ledger.Log
	sem chan struct{}
	// useContainer is the session's RUNTIME runner mode: when true, live packets run in
	// the hardened agent container (runHarnessContainer) instead of the host
	// subprocess. Initialized from cfg.UseContainer (the -container boot flag) and
	// flipped by the ToggleRunner action, so the Lead can switch at runtime instead of
	// only at boot. Guarded by fillMu (read on the fill path, written by the toggle).
	useContainer bool
	// runMu serializes the per-key packet runner so two concurrent Spends can't both
	// drain (and double-run) the same queued packet. One drainer per session at a time.
	runMu sync.Mutex
	// seq is the registration ordinal — a monotonic stamp assigned when the session
	// is registered. The fleet board orders ties (equal queued counts) by it, since
	// sync.Map.Range is nondeterministic and a CatchRecord carries no timestamp to
	// order by; registration order is the only stable, honest ordinal.
	seq int
	// findingsMu guards findings — the latest connect cycle's open review questions
	// (the fix oracle's surviving/undetermined mutants). It is EPHEMERAL session
	// state, recomputed every connect, deliberately OFF the append-only economy
	// ledger (a diagnostic, never a catch/balance — the two-scores guard). The
	// /review surface reads it; OnConnect writes it when the cycle resolves.
	findingsMu sync.Mutex
	findings   []mutation.Finding
	// resolved holds the "file:line" of questions a reviewer ANSWERED (their test
	// killed the mutant) this session. A killing answer makes the
	// question vanish; since the reviewer's test isn't committed, a later connect
	// cycle would re-find the survivor, so openFindings filters resolved lines out —
	// the answer sticks for the session. Ephemeral, off the economy ledger (a
	// diagnostic, like findings); guarded by findingsMu.
	resolved map[string]bool
	// blockedQ tracks which review-question ids already have a logged attention block
	// this session, so recordQuestionBlocks logs each question's block ONCE — a later
	// connect cycle re-finding the same survivor never re-blocks. Guarded by findingsMu.
	blockedQ map[string]bool
	// land is the latest connect cycle's integration verdict (clean/conflict/
	// checks_red), cached so the fleet board can show which sessions are blocked from
	// merging — ephemeral, recomputed each connect, off the economy ledger. Guarded
	// by findingsMu (written together with findings in OnConnect).
	land string
	// landResult is the outcome of the last Approve (PR URL / guard / error), surfaced
	// on the card. Ephemeral, off-ledger; guarded by findingsMu.
	landResult string
	// adjAnchors records EVERY adjustment's anchor this session (file:line + the commented
	// line's content + the Lead's comment), in order, so a later render can relocate each
	// against the new revision and show whether the agent addressed it (DESIGN §28 thin
	// slice). Upserted by file:line (re-commenting a line replaces, never stacks);
	// ephemeral, off the economy ledger; guarded by findingsMu.
	adjAnchors []adjAnchorRecord
	// landLifecycle is the opened PR's post-land lifecycle ("landed"/"merged"/"bounced",
	// DESIGN §29.2 "Landed ≠ Merged"), surfaced as a badge on the land control. Ephemeral,
	// off the economy ledger; guarded by findingsMu. ""=no opened PR / no lifecycle yet.
	landLifecycle string
	// lastPushedSHA is the squashed commit the last land push put on the PR branch, so a
	// re-land leases its force against it (pushRefspec). Ephemeral, off-ledger; guarded by
	// findingsMu. ""=never pushed.
	lastPushedSHA string
	// orderFindings holds a FILLED packet's review questions (the cycle's
	// surviving mutants) keyed by packet ID — captured when runOneOrder fills the
	// packet, so a funded packet's test-debt is reviewable (the send→review tie).
	// Ephemeral and OFF the economy ledger, like findings (the packet's CATCH mints;
	// its questions are diagnostic). Guarded by findingsMu.
	orderFindings map[int][]mutation.Finding
	// analysis is the latest authoring-assist read of a draft packet (the source's
	// summary/readiness/highlights/questions), cached so the card renders it after the
	// AnalyzeDraft action. Ephemeral and OFF the economy ledger (a diagnostic, like
	// findings — analyzing a draft mints nothing). Guarded by findingsMu.
	analysis *draftAnalysis
	// analysisCancel cancels the latest in-flight authoring-assist run. A new analyze
	// cancels the prior one (beginAnalysis) so a fast-typing Lead's superseded reads
	// are abandoned (the slow model call killed), never left racing to overwrite the
	// cache out of order. Guarded by findingsMu.
	analysisCancel context.CancelFunc
	// rewrite is the latest source-rewritten draft (UpdateDraft folds the Lead's
	// answers into the draft and stashes the new text here), read by composeSurface
	// into the editor's rewrite payload so Monaco swaps to it. Ephemeral, OFF the
	// economy ledger (a diagnostic, like analysis). Guarded by findingsMu.
	rewrite string
	// pendingHandshake is the handshake AuthorHandshake wrote for
	// the NEXT live packet this session places — Send folds its Path/Hash/
	// Strength into the sent Target, then CONSUMES it (clears it back to nil),
	// so a later packet can never silently reuse an earlier one's contract. Ephemeral,
	// off the economy ledger (set at compose time, never by the agent). Guarded by
	// findingsMu.
	pendingHandshake *packet.Handshake
	// composeMessage is the honest inline refusal Send leaves for the Lead when
	// it refuses to send a live packet (e.g. no handshake authored yet) —
	// cleared on a successful placement. Ephemeral, off the economy ledger; guarded
	// by findingsMu.
	composeMessage string
	// harnessSessionID is this packets-session's resumable claude session id — the one
	// the warm-up explores under, REMEMBERED so every later analyze + packet resumes it
	// (warm repo context). harnessWarm gates use: requests resume the id ONLY after the
	// warm-up explore completes (before that they run cold, never resuming a session
	// still being established). Both guarded by findingsMu; "" id = no warm harness.
	harnessSessionID string
	harnessWarm      bool
	// answering is true while an answer re-run is in flight for this session. It
	// serializes answer re-runs (one at a time): a re-run spawns a git worktree +
	// oracle run, so two concurrent re-runs (a double-clicked submit) would race the
	// shared repo's worktree operations. Guarded by findingsMu.
	answering bool
	// landing is true while an Approve (land) call is in flight for this session. It
	// serializes lands (one at a time): a land pushes to the session's shared branch
	// and opens a PR, so two concurrent Approve calls (a double-clicked "forward →",
	// or two tabs) would race the shared repo's push/PR operations. Guarded by
	// findingsMu, mirroring answering/beginAnswer/endAnswer.
	landing bool
	// fillMu + fillingOrder/fillBeats: the live-fill buffer (see startFill). Guarded
	// separately from findingsMu since beats accrue rapidly during a fill.
	fillMu       sync.Mutex
	fillingOrder int
	fillBeats    []string
	// activityBeat is the live agent's LATEST activity line (e.g. "editing auth.go")
	// while a live packet fills — a single updating beat, not a log. Bracketed to the
	// fill lifecycle (reset in startFill, cleared in endFill) and guarded by fillMu.
	activityBeat string
	// activityLog is the accruing TRANSCRIPT of the agent's beats this fill, in stream
	// order — the run made legible (the card scrolls it) rather than only the latest
	// move. Capped at maxActivityLog (oldest dropped) so a long run can't grow it
	// without bound. Bracketed to the fill lifecycle like activityBeat; guarded by fillMu.
	activityLog []string
	// addrOnce/addr cache this session's repo identity (packet.ParseAddr execs
	// git, so it is resolved once per session and reused on every later render,
	// not re-shelled out to on each poll). Guarded by addrOnce, not findingsMu —
	// the value is immutable once set.
	addrOnce sync.Once
	addr     packet.Addr
	// laneMu guards laneCache: a packet's measured Lane, computed via
	// packet.Measure (an exec seam — `go list` + git) at most once per packet
	// id. A fixRev change mints a new packet id (the identity
	// model), so there is no staleness to invalidate at this stage. Separate from
	// findingsMu since it guards an unrelated, exec-derived cache with a much
	// higher per-entry cost — a lock a render might hold across a subprocess
	// call must never also block the cheap off-ledger diagnostic reads above.
	laneMu    sync.Mutex
	laneCache map[int]packet.Lane
	// gauntletMu guards gauntletCache: a packet's computed Gauntlet record,
	// mirroring laneMu/laneCache's exec-derived-cache
	// rationale exactly — G4 (packet.RunBuildVetGate) execs git+go, so this
	// cache must stay off findingsMu (a render must never hold a lock across
	// a subprocess call). intentFidelityConfirmed is guarded by the SAME
	// mutex (a much cheaper map, but keeping one lock for the whole gauntlet
	// record avoids a second mutex for no real contention benefit).
	gauntletMu    sync.Mutex
	gauntletCache map[int]packet.Gauntlet
	// intentFidelityConfirmed holds G1's human confirmation per packet id
	// (ConfirmIntentFidelity) — a real ACTION, not a computed gate, so it is
	// NEVER folded into gauntletCache's stored value: gauntletFor and
	// cachedGauntlet both read this fresh on every call and overlay it onto
	// IntentFidelity, so a confirmation made AFTER a packet's G3/G4 were
	// already cached is still visible on the very next render without
	// invalidating that cache entry. Ephemeral, off the economy ledger —
	// there is no persisted event kind for this yet (a deferral: adding one
	// is out of scope this slice).
	intentFidelityConfirmed map[int]packet.Gate
	// calibMu guards calibDraw: this session's currently-cached calibration
	// sample (the calibration draw) — the drawn packet id, kept STABLE across
	// renders (drawCalibration) until it ages out of the auto-forwarded set.
	// Separate from findingsMu/laneMu: a render-cheap, rarely-changing value
	// with no reason to contend with either's own traffic.
	calibMu   sync.Mutex
	calibDraw int
	// orderCatch holds a FILLED packet's raw catch-cycle outcome and
	// after-revision survivor/inventory counts (settleCatch's Resolution),
	// carried alongside orderFindings — the data G3
	// (packet.GateFromCatchOutcome) folds into a gate without ever
	// re-running the mutation oracle. Guarded by findingsMu, written
	// together with orderFindings in settleCatch.
	orderCatch map[int]packetCatchOutcome
	// watchMu guards watchFires: this session's standing-watch history
	// (standing inspection). Pure in-memory bookkeeping — an
	// append/read over a slice, no exec — so, like calibMu, there is no
	// reason to share findingsMu/laneMu's own traffic. Off the ledger: no
	// persisted event kind for a fire/mark exists yet (a deferral matching
	// intentFidelityConfirmed's precedent).
	watchMu    sync.Mutex
	watchFires []packet.WatchFire
}

// packetCatchOutcome is one packet's raw catch-cycle result, cached off the
// economy ledger (like orderFindings) so gauntletFor's G3 can fold it into a
// packet.Gate without re-running the mutation oracle.
type packetCatchOutcome struct {
	outcome    catch.Outcome
	survivors  int
	considered int
}

// setOrderCatchOutcome caches a filled packet's raw catch-cycle outcome —
// called from settleCatch alongside setOrderFindings.
func (e *liveEntry) setOrderCatchOutcome(orderID int, outcome catch.Outcome, survivors, considered int) {
	e.findingsMu.Lock()
	if e.orderCatch == nil {
		e.orderCatch = map[int]packetCatchOutcome{}
	}
	e.orderCatch[orderID] = packetCatchOutcome{outcome: outcome, survivors: survivors, considered: considered}
	e.findingsMu.Unlock()
}

// handshakeTightnessGate derives G3 from orderID's cached catch outcome, or
// the honest NotRun default when no catch cycle has recorded one yet (the
// packet hasn't filled, or filled with a cycle error that settled nothing).
func (e *liveEntry) handshakeTightnessGate(orderID int) packet.Gate {
	e.findingsMu.Lock()
	oc, ok := e.orderCatch[orderID]
	e.findingsMu.Unlock()
	if !ok {
		return packet.Gate{Status: packet.GateNotRun, Detail: "not measured — no catch cycle run yet"}
	}
	return packet.GateFromCatchOutcome(oc.outcome, oc.survivors, oc.considered)
}

// intentFidelityGate reads G1 fresh from intentFidelityConfirmed — see the
// field doc on why this is never part of the cached Gauntlet value.
func (e *liveEntry) intentFidelityGate(orderID int) packet.Gate {
	e.gauntletMu.Lock()
	defer e.gauntletMu.Unlock()
	if g, ok := e.intentFidelityConfirmed[orderID]; ok {
		return g
	}
	return packet.Gate{Status: packet.GateNotRun, Detail: "no data — a human residual, not computable"}
}

// confirmIntentFidelity records that navKey confirmed orderID's intent
// fidelity — the G1 human residual is a real action, never a computed gate.
func (e *liveEntry) confirmIntentFidelity(orderID int, navKey string) {
	e.gauntletMu.Lock()
	if e.intentFidelityConfirmed == nil {
		e.intentFidelityConfirmed = map[int]packet.Gate{}
	}
	e.intentFidelityConfirmed[orderID] = packet.Gate{Status: packet.GatePassed, Detail: "confirmed by " + navKey}
	e.gauntletMu.Unlock()
}

// standingWatchKinds is the three canonical, pre-defined STANDING triggers
// (packet.WatchKind's fixed set) evaluated every render — not an
// author-your-own DSL, per standing inspection's bounded-scope design.
var standingWatchKinds = []packet.WatchKind{
	packet.WatchStrictLane, packet.WatchGateFailure, packet.WatchBlockingHold,
}

// recordWatchFires evaluates every standing watch kind against packets and
// appends a new WatchFire the FIRST time a given (kind, packet id) pair
// matches — idempotent, mirroring recordQuestionBlocks' dedup pattern, so a
// packet sitting in a matching state across many renders logs exactly one
// fire, never one per render (a render is not an event).
func (e *liveEntry) recordWatchFires(packets []packet.Packet) {
	e.watchMu.Lock()
	defer e.watchMu.Unlock()
	seen := make(map[packet.WatchKind]map[int]bool, len(standingWatchKinds))
	for _, f := range e.watchFires {
		if seen[f.Kind] == nil {
			seen[f.Kind] = map[int]bool{}
		}
		seen[f.Kind][f.PacketID] = true
	}
	now := time.Now().UnixMilli()
	for _, p := range packets {
		for _, k := range standingWatchKinds {
			if seen[k][p.ID] {
				continue
			}
			if !packet.EvaluateWatch(k, p) {
				continue
			}
			e.watchFires = append(e.watchFires, packet.WatchFire{Kind: k, PacketID: p.ID, AtUnixMs: now})
			if seen[k] == nil {
				seen[k] = map[int]bool{}
			}
			seen[k][p.ID] = true
		}
	}
}

// watchFireSnapshot returns a COPY of this session's recorded fires — safe
// to range over after the lock is released, and never mutated by the caller
// (that would race the next recordWatchFires/markWatchFire call).
func (e *liveEntry) watchFireSnapshot() []packet.WatchFire {
	e.watchMu.Lock()
	defer e.watchMu.Unlock()
	out := make([]packet.WatchFire, len(e.watchFires))
	copy(out, e.watchFires)
	return out
}

// markWatchFire finds the unmarked fire for (kind, packetID) and records the
// human's usefulness judgment — Precision is computed from real judgment,
// never inferred. The reverse scan is defensive, not load-bearing: recordWatchFires'
// own dedup means at most one fire is ever recorded per (kind, packetID), so there
// is never more than one to find. A (kind, packetID) with no unmarked fire (never
// fired, or already resolved) is a calm no-op.
func (e *liveEntry) markWatchFire(kind packet.WatchKind, packetID int, useful bool) {
	e.watchMu.Lock()
	defer e.watchMu.Unlock()
	for i := len(e.watchFires) - 1; i >= 0; i-- {
		f := &e.watchFires[i]
		if f.Kind == kind && f.PacketID == packetID && f.Useful == nil {
			f.Useful = &useful
			return
		}
	}
}

// notMeasuredNoHandshake is G5's honest default: G5 needs the handshake/
// agent-test split (mutation vs the agent's own tests) that is explicitly
// deferred past this slice — see gauntlet_handshake.go's doc. G2 (handshake
// conformance) became a real gate (handshakeConformanceGate
// below); this sentinel is also G2's OWN answer for a packet with no
// handshake authored at all (p.HandshakePath == "").
var notMeasuredNoHandshake = packet.Gate{Status: packet.GateNotRun, Detail: "not measured — no handshake yet"}

// handshakeConformanceGate computes G2 (one of the gauntlet's six gates): the honest
// no-handshake sentinel when p carries none, otherwise packet.RunHandshakeGate
// overridden to a hard fail when packet.VerifyHandshake finds the LIVE
// handshake file (under repoDir, independent of any particular fix revision)
// no longer matches its authored-time hash — integrity wins over a stale pass
// (the handshake content-hash-before-gates invariant's belt-and-suspenders
// alongside the settle deny-rule). A VerifyHandshake ERROR (file gone/unreadable, distinct from a
// confirmed mismatch) is not treated as a mismatch: the fix revision's own
// test-run result stands, since an infra fact about the live file is not
// itself a fabricated finding about the revision under gate.
//
// Called from BOTH of gauntletFor's branches: the handshake file's identity
// is independent of FixRev, so a revless live packet that already has a
// handshake authored still gets an honest (uncached) answer here rather than
// reverting to "no handshake yet".
func (e *liveEntry) handshakeConformanceGate(ctx context.Context, p packet.Packet) packet.Gate {
	if p.HandshakePath == "" {
		return notMeasuredNoHandshake
	}
	gate := packet.RunHandshakeGate(ctx, e.cfg.RepoDir, p.FixRev, p.HandshakePath)
	if ok, err := packet.VerifyHandshake(packet.Handshake{Path: p.HandshakePath, Hash: p.HandshakeHash}); err == nil && !ok {
		return packet.Gate{Status: packet.GateFailed, Detail: "handshake changed after authoring — content no longer matches its recorded hash"}
	}
	return gate
}

// notMeasuredNoCage is G6's honest default: cage re-derivation exists
// (internal/cage) but is never wired into a locally-sent packet's
// gauntlet yet — a future slice's job, not this one's.
var notMeasuredNoCage = packet.Gate{Status: packet.GateNotRun, Detail: "not measured — cage not wired to local send"}

// independentCheckGate derives G6 (method diversity — cage re-derivation) for
// a filled packet: cage unconfigured (StartCageClaimConsumers never called this
// process) stays the honest notMeasuredNoCage default. Once configured, it
// re-verifies orderID's OWN target through the SAME cage-backed
// ledger.Verifier the claim consumers run, so G6 is a genuine independent
// re-derivation of the fix — not a re-read of the in-process catch outcome G3
// already holds. A permanent ledger.ErrClaimUnverifiable (the target can never
// verify) and any other verifier error (a transient cage/runner failure) both
// stay GateNotRun — the "never fabricate a metric" invariant every other gate
// in this file follows: an infra fact about the cage is not itself a proven
// finding about the revision under gate. A nil record with no error is the
// cage's own honest non-catch (no gap survived to a confirmed catch); it
// reports Held rather than inventing survivor counts the Verifier seam never
// hands back for anything but a genuine catch (see ledger.NewCatchRecord).
func (e *liveEntry) independentCheckGate(target ledger.Target) packet.Gate {
	verify, ok := cageVerifierFor(e.cfg)
	if !ok {
		return notMeasuredNoCage
	}
	rec, err := verify(ledger.ClaimRecord{Target: target})
	if err != nil {
		if errors.Is(err, ledger.ErrClaimUnverifiable) {
			return packet.Gate{Status: packet.GateNotRun, Detail: "not measured — cage could not verify this claim's target"}
		}
		return packet.Gate{Status: packet.GateNotRun, Detail: "not measured — cage verify failed: " + err.Error()}
	}
	if rec == nil {
		return packet.Gate{Status: packet.GateHeld, Detail: "cage re-derivation found no catch to confirm"}
	}
	return packet.GateFromCatchOutcome(rec.Outcome, 0, rec.MutantsConsidered)
}

// cachedGauntletEntry reads gauntletCache directly, reporting a miss
// distinctly from a cached all-NotRun Gauntlet — mirrors cachedLaneEntry.
func (e *liveEntry) cachedGauntletEntry(orderID int) (packet.Gauntlet, bool) {
	e.gauntletMu.Lock()
	defer e.gauntletMu.Unlock()
	g, ok := e.gauntletCache[orderID]
	return g, ok
}

// gauntletFor returns p's gauntlet record (the gauntlet's six gates),
// computing (and caching) G3/G4 on the first call to observe a cache miss
// for this packet id — mirrors laneFor exactly, including the "no fix
// revision yet → answer honestly WITHOUT caching" rule (a live prompt packet
// the harness hasn't produced a revision for), so a later render — once the
// packet fills — gets a real computation instead of a permanently-cached
// miss. G1 is NEVER cached (see intentFidelityConfirmed's doc); G2/G5/G6
// have no real mechanic this slice and are always their honest NotRun
// default.
//
// This execs `git worktree`/`go build`/`go vet` (packet.RunBuildVetGate) and
// may be called ONLY from a render path (View), scoped to the packet(s)
// actually shown in detail — the Inspector's packet-scoped branch. The 100ms
// via.Stream poll (OnConnect) must NEVER call this; it reads cachedGauntlet
// instead, a pure map lookup.
func (e *liveEntry) gauntletFor(ctx context.Context, p packet.Packet) packet.Gauntlet {
	g, ok := e.cachedGauntletEntry(p.ID)
	if !ok {
		if p.FixRev == "" {
			g = packet.Gauntlet{
				HandshakeConformance: e.handshakeConformanceGate(ctx, p),
				TestSensitivity:      notMeasuredNoHandshake,
				IndependentCheck:     notMeasuredNoCage,
			}
			g.IntentFidelity = e.intentFidelityGate(p.ID)
			return g // no fix rev yet — nothing real to cache
		}
		independentCheck := notMeasuredNoCage
		if target, ok := orderTarget(e.log, p.ID); ok {
			independentCheck = e.independentCheckGate(target)
		}
		g = packet.Gauntlet{
			HandshakeConformance: e.handshakeConformanceGate(ctx, p),
			HandshakeTightness:   e.handshakeTightnessGate(p.ID),
			BuildVetLint:         packet.RunBuildVetGate(ctx, e.cfg.RepoDir, p.FixRev),
			TestSensitivity:      notMeasuredNoHandshake,
			IndependentCheck:     independentCheck,
		}
		e.gauntletMu.Lock()
		if e.gauntletCache == nil {
			e.gauntletCache = map[int]packet.Gauntlet{}
		}
		e.gauntletCache[p.ID] = g
		e.gauntletMu.Unlock()
	}
	g.IntentFidelity = e.intentFidelityGate(p.ID)
	return g
}

// cachedGauntlet returns orderID's ALREADY-cached gauntlet (the honest
// all-NotRun zero value on a cache miss), overlaid with G1's fresh
// confirmation state — a pure map read, NEVER a compute. Anything reachable
// from the 100ms Stream poll must use this, never gauntletFor.
func (e *liveEntry) cachedGauntlet(orderID int) packet.Gauntlet {
	g, _ := e.cachedGauntletEntry(orderID)
	g.IntentFidelity = e.intentFidelityGate(orderID)
	return g
}

// resolvedAddr returns this session's repo identity (owner/name), computed
// once per session and cached for every later render.
func (e *liveEntry) resolvedAddr() packet.Addr {
	e.addrOnce.Do(func() { e.addr = packet.ParseAddr(e.cfg.RepoDir) })
	return e.addr
}

// cachedLaneEntry reads laneCache directly, reporting a miss distinctly from
// a cached LaneUnmeasured — the distinction laneFor needs to decide whether
// to compute.
func (e *liveEntry) cachedLaneEntry(orderID int) (packet.Lane, bool) {
	e.laneMu.Lock()
	defer e.laneMu.Unlock()
	lane, ok := e.laneCache[orderID]
	return lane, ok
}

// laneFor returns p's measured Lane, computing (and caching) it on the first
// call to observe a cache miss for this packet id. A packet with no fix rev
// yet (still queued — e.g. a live prompt packet the harness hasn't produced a
// revision for) returns LaneUnmeasured WITHOUT computing or caching, so a
// later render — once the revs exist — gets a real measurement instead of a
// permanently-cached miss. Every OTHER outcome, including a Measure error, IS
// cached (as LaneUnmeasured on error) so a doomed measurement is not
// re-shelled-out on every render.
//
// Concurrent misses for the SAME packet (e.g. two tabs opening one Inspector
// at once) are not deduplicated — each runs its own packet.Measure and the
// last write wins the cache slot. Both computed the same real answer, so the
// cache ends up correct either way; the cost is redundant subprocess work,
// never a correctness bug. Not worth a singleflight for the Inspector's
// one-human-at-a-time access pattern.
//
// This execs `go list`/git (packet.Measure) and may be called ONLY from a
// render path (View), scoped to the packet(s) actually shown in detail — the
// Inspector's packet-scoped branch. The 100ms via.Stream poll (OnConnect) must
// NEVER call this; it reads cachedLane instead, a pure map lookup.
func (e *liveEntry) laneFor(ctx context.Context, p packet.Packet) packet.Lane {
	if lane, ok := e.cachedLaneEntry(p.ID); ok {
		return lane
	}
	if p.BaseRev == "" || p.FixRev == "" {
		return packet.LaneUnmeasured
	}
	lane, _ := packet.Measure(ctx, e.cfg.RepoDir, p.BaseRev, p.FixRev)
	e.laneMu.Lock()
	if e.laneCache == nil {
		e.laneCache = map[int]packet.Lane{}
	}
	e.laneCache[p.ID] = lane
	e.laneMu.Unlock()
	return lane
}

// cachedLane returns orderID's ALREADY-cached lane, LaneUnmeasured on a cache
// miss — a pure map read, NEVER a compute. The Console's lane-health grid
// (and anything reachable from the 100ms Stream poll) must use this, never
// laneFor.
func (e *liveEntry) cachedLane(orderID int) packet.Lane {
	lane, _ := e.cachedLaneEntry(orderID)
	return lane
}

// cachedCalibDraw returns this session's currently-cached calibration draw
// id, 0 if none has been drawn yet (0 is never a real packet id — ids start
// at 1).
func (e *liveEntry) cachedCalibDraw() int {
	e.calibMu.Lock()
	defer e.calibMu.Unlock()
	return e.calibDraw
}

// setCalibDraw caches id as this session's calibration draw.
func (e *liveEntry) setCalibDraw(id int) {
	e.calibMu.Lock()
	e.calibDraw = id
	e.calibMu.Unlock()
}

// maxActivityLog bounds the in-flight transcript so a long agent run can't grow the
// per-session buffer without limit; the oldest beats scroll off once it is reached.
const maxActivityLog = 300

// fillMu guards the live-fill buffer: the packet currently being filled by the
// background runner and the cycle beats accrued so far, so the card can show it
// filling LIVE ("watch it fill"). The runner has no request ctx to write the card's
// cells, so it writes this buffer and the card's Stream polls it (like the send
// tally). Ephemeral, off the economy ledger.
func (e *liveEntry) startFill(id int) {
	e.fillMu.Lock()
	e.fillingOrder, e.fillBeats, e.activityBeat, e.activityLog = id, nil, "", nil
	e.fillMu.Unlock()
}

// addActivityBeat records the agent's latest move: it replaces the single
// latest-line AND appends to the transcript (capped at maxActivityLog, oldest
// dropped) so the card can show both the current move and the run's history.
func (e *liveEntry) addActivityBeat(beat string) {
	e.fillMu.Lock()
	e.activityBeat = beat
	e.activityLog = append(e.activityLog, beat)
	if len(e.activityLog) > maxActivityLog {
		e.activityLog = e.activityLog[len(e.activityLog)-maxActivityLog:]
	}
	e.fillMu.Unlock()
}

// activitySnapshot returns the live agent's latest activity line ("" when none).
func (e *liveEntry) activitySnapshot() string {
	e.fillMu.Lock()
	defer e.fillMu.Unlock()
	return e.activityBeat
}

// useContainerMode reports whether this session's live packets run in the hardened
// container (vs the host subprocess) — the runtime runner mode runLivePacket reads.
func (e *liveEntry) useContainerMode() bool {
	e.fillMu.Lock()
	defer e.fillMu.Unlock()
	return e.useContainer
}

// toggleRunner flips the session between host-subprocess and container execution for
// its next live packet (an in-flight packet keeps the runner it started on).
func (e *liveEntry) toggleRunner() {
	e.fillMu.Lock()
	e.useContainer = !e.useContainer
	e.fillMu.Unlock()
}

// activityTranscript returns a copy of the accruing beat transcript for this fill
// (nil when none) — the run history the card scrolls.
func (e *liveEntry) activityTranscript() []string {
	e.fillMu.Lock()
	defer e.fillMu.Unlock()
	return append([]string(nil), e.activityLog...)
}

// addFillBeat appends one cycle beat for the filling packet (the live tempo).
func (e *liveEntry) addFillBeat(kind string) {
	e.fillMu.Lock()
	e.fillBeats = append(e.fillBeats, kind)
	e.fillMu.Unlock()
}

// endFill clears the live-fill buffer when the packet is done — the filling row
// vanishes and the packet's resolved outcome takes over.
func (e *liveEntry) endFill() {
	e.fillMu.Lock()
	e.fillingOrder, e.fillBeats, e.activityBeat, e.activityLog = 0, nil, "", nil
	e.fillMu.Unlock()
}

// fillSnapshot returns the filling packet's id (0 if none) and a copy of its beats.
func (e *liveEntry) fillSnapshot() (int, []string) {
	e.fillMu.Lock()
	defer e.fillMu.Unlock()
	if e.fillingOrder == 0 {
		return 0, nil
	}
	return e.fillingOrder, append([]string(nil), e.fillBeats...)
}

// beginAnswer claims the single in-flight answer slot for the session, returning
// false if a re-run is already running (so the caller drops the duplicate). Pair
// every true with endAnswer.
func (e *liveEntry) beginAnswer() bool {
	e.findingsMu.Lock()
	defer e.findingsMu.Unlock()
	if e.answering {
		return false
	}
	e.answering = true
	return true
}

// endAnswer releases the in-flight answer slot.
func (e *liveEntry) endAnswer() {
	e.findingsMu.Lock()
	e.answering = false
	e.findingsMu.Unlock()
}

// beginLand claims the single in-flight land slot for the session, returning false
// if an Approve is already running (so the caller drops the duplicate). Pair every
// true with endLand.
func (e *liveEntry) beginLand() bool {
	e.findingsMu.Lock()
	defer e.findingsMu.Unlock()
	if e.landing {
		return false
	}
	e.landing = true
	return true
}

// endLand releases the in-flight land slot.
func (e *liveEntry) endLand() {
	e.findingsMu.Lock()
	e.landing = false
	e.findingsMu.Unlock()
}

// setFindings caches the latest cycle's open review questions for the /review
// surface to read. Concurrency-safe vs a concurrent /review read.
func (e *liveEntry) setFindings(fs []mutation.Finding) {
	e.findingsMu.Lock()
	e.findings = fs
	e.findingsMu.Unlock()
}

// openFindings returns the session's latest cached open review questions, with any
// the reviewer has ANSWERED (markResolved) this session filtered out — so a killing
// answer stays vanished even when a later connect cycle re-finds the uncommitted
// survivor (the answered question stays vanished for the session).
func (e *liveEntry) openFindings() []mutation.Finding {
	e.findingsMu.Lock()
	defer e.findingsMu.Unlock()
	if len(e.resolved) == 0 {
		return e.findings
	}
	out := make([]mutation.Finding, 0, len(e.findings))
	for _, f := range e.findings {
		if e.resolved[findingKey(f.File, f.Line)] {
			continue
		}
		out = append(out, f)
	}
	return out
}

// markResolved records that the question at file:line was answered (its mutant
// killed), so openFindings filters it out for the rest of the session.
func (e *liveEntry) markResolved(file string, line int) {
	e.findingsMu.Lock()
	defer e.findingsMu.Unlock()
	if e.resolved == nil {
		e.resolved = map[string]bool{}
	}
	e.resolved[findingKey(file, line)] = true
}

// findingKey is the per-line identity used to match a resolved answer to a finding.
func findingKey(file string, line int) string { return file + ":" + strconv.Itoa(line) }

// recordQuestionBlocks logs an attention BLOCK for each newly-surfaced review
// question — the source needing the Lead's input — starting that question's
// bandwidth interval. It is idempotent per question id: a later connect cycle that
// re-finds the same survivor never re-blocks (blockedQ tracks what is already
// open), so the interval is anchored to when the question FIRST appeared. The
// ledger publish runs outside findingsMu so I/O never holds the lock; a logging
// failure is best-effort (a missed block only forgoes a future award, never breaks
// the cycle).
func (e *liveEntry) recordQuestionBlocks(fs []mutation.Finding) {
	e.findingsMu.Lock()
	if e.blockedQ == nil {
		e.blockedQ = map[string]bool{}
	}
	var fresh []string
	for _, f := range fs {
		id := findingKey(f.File, f.Line)
		if !e.blockedQ[id] {
			e.blockedQ[id] = true
			fresh = append(fresh, id)
		}
	}
	e.findingsMu.Unlock()
	now := time.Now()
	for _, id := range fresh {
		_ = e.log.AppendBlock(id, now)
	}
}

// recordQuestionUnblock logs that the Lead cleared the question at file:line — the
// unblock that closes its bandwidth interval and earns the latency-weighted award.
// Best-effort, off the catch economy (the balance firewall is untouched: an unblock
// moves only the bandwidth meter).
func (e *liveEntry) recordQuestionUnblock(file string, line int) {
	_ = e.log.AppendUnblock(findingKey(file, line), time.Now())
}

// setOrderFindings caches a filled packet's review questions (off-ledger, like
// findings) so the packet's test-debt is reviewable. Empty findings clear the entry.
func (e *liveEntry) setOrderFindings(id int, fs []mutation.Finding) {
	e.findingsMu.Lock()
	defer e.findingsMu.Unlock()
	if len(fs) == 0 {
		delete(e.orderFindings, id)
		return
	}
	if e.orderFindings == nil {
		e.orderFindings = map[int][]mutation.Finding{}
	}
	e.orderFindings[id] = fs
}

// orderQuestionCount returns how many open review questions a filled packet left.
func (e *liveEntry) orderQuestionCount(id int) int {
	e.findingsMu.Lock()
	defer e.findingsMu.Unlock()
	return len(e.orderFindings[id])
}

// sessionPackets folds a session's sends into packets — the Console and
// Inspector's shared read model. n<=0 folds every send
// (an honest total for counts like the hero stat); a positive n caps the
// underlying RecentSends read. Empty (nil) when the session or its
// ledger is unknown — callers treat that as "nothing to show", never an
// error.
func sessionPackets(key string, n int) []packet.Packet {
	e := lookupLiveEntry(key)
	if e == nil || e.log == nil {
		return nil
	}
	views, err := e.log.RecentSends(n)
	if err != nil {
		return nil
	}
	return packet.Fold(views, e.resolvedAddr(), func(id int) int {
		return len(orderOpenThreads(key, id))
	})
}

// sessionAddr returns a session's repo identity, or the zero Addr when the
// session is unknown.
func sessionAddr(key string) packet.Addr {
	if e := lookupLiveEntry(key); e != nil {
		return e.resolvedAddr()
	}
	return packet.Addr{}
}

// orderFindingsFor returns a filled packet's cached review questions (nil if none).
func (e *liveEntry) orderFindingsFor(id int) []mutation.Finding {
	e.findingsMu.Lock()
	defer e.findingsMu.Unlock()
	return e.orderFindings[id]
}

// setLand caches the latest cycle's integration verdict for the fleet board.
func (e *liveEntry) setLand(land string) {
	e.findingsMu.Lock()
	e.land = land
	e.findingsMu.Unlock()
}

// landState returns the session's latest cached integration verdict ("" if none).
func (e *liveEntry) landState() string {
	e.findingsMu.Lock()
	defer e.findingsMu.Unlock()
	return e.land
}

// setLandResult / landResultSnapshot cache the outcome of the last Approve (the opened
// PR URL, a "blocked — …" guard message, or a "PR failed — …" error) so the card can
// surface it. Ephemeral, OFF the economy ledger — a diagnostic, not a catch. Guarded
// by findingsMu.
func (e *liveEntry) setLandResult(res string) {
	e.findingsMu.Lock()
	e.landResult = res
	e.findingsMu.Unlock()
}

func (e *liveEntry) landResultSnapshot() string {
	e.findingsMu.Lock()
	defer e.findingsMu.Unlock()
	return e.landResult
}

// addAdjAnchor UPSERTS one adjustment's anchor by file:line (the file:line the Lead
// commented, the line's content at comment time, and the comment text) — re-commenting a
// line replaces its entry rather than stacking a duplicate — so a later render can
// relocate each against the new revision (relocateAdjustments) and show whether the agent
// addressed it. Ephemeral, off the economy ledger, guarded by findingsMu — like landResult.
func (e *liveEntry) addAdjAnchor(file string, line int, content, comment string) {
	e.findingsMu.Lock()
	e.adjAnchors = upsertAnchor(e.adjAnchors, adjAnchorRecord{file: file, line: line, content: content, comment: comment})
	e.findingsMu.Unlock()
}

// setLandLifecycle records the opened PR's lifecycle (DESIGN §29.2), guarded by findingsMu
// — ephemeral, off the economy ledger, like landResult.
func (e *liveEntry) setLandLifecycle(lc string) {
	e.findingsMu.Lock()
	e.landLifecycle = lc
	e.findingsMu.Unlock()
}

// landLifecycleSnapshot returns the opened PR's cached lifecycle ("" if none).
func (e *liveEntry) landLifecycleSnapshot() string {
	e.findingsMu.Lock()
	defer e.findingsMu.Unlock()
	return e.landLifecycle
}

// removeAdjAnchor clears the adjustment anchored at file:line (the Lead resolved it), a
// no-op if absent. Ephemeral, off the economy ledger, guarded by findingsMu — twin of
// addAdjAnchor.
func (e *liveEntry) removeAdjAnchor(file string, line int) {
	e.findingsMu.Lock()
	e.adjAnchors = removeAnchor(e.adjAnchors, file, line)
	e.findingsMu.Unlock()
}

// adjAnchorsSnapshot returns a COPY of this session's adjustment anchors, in order (nil if
// none) — safe to read outside the lock.
func (e *liveEntry) adjAnchorsSnapshot() []adjAnchorRecord {
	e.findingsMu.Lock()
	defer e.findingsMu.Unlock()
	if len(e.adjAnchors) == 0 {
		return nil
	}
	return append([]adjAnchorRecord(nil), e.adjAnchors...)
}

// setLastPushedSHA records the squashed commit the last land push put on the session's PR
// branch, so a re-land can lease its force against it (push only if the remote branch is
// still there — pushRefspec). Ephemeral, off the economy ledger; guarded by findingsMu.
func (e *liveEntry) setLastPushedSHA(sha string) {
	e.findingsMu.Lock()
	e.lastPushedSHA = sha
	e.findingsMu.Unlock()
}

// lastPushedSHASnapshot returns the SHA of the last land push (""=never pushed → the next
// push leases against must-not-exist).
func (e *liveEntry) lastPushedSHASnapshot() string {
	e.findingsMu.Lock()
	defer e.findingsMu.Unlock()
	return e.lastPushedSHA
}

// sessionOpenThreads converts a session's cached open findings into review threads
// (anchored "question:" comments). Empty when the session is unknown or its last
// cycle left no surviving mutants.
func sessionOpenThreads(key string) []review.Thread {
	e := lookupLiveEntry(key)
	if e == nil {
		return nil
	}
	return review.QuestionThreadsFromMutations(e.openFindings())
}

// setAnalysis caches the latest authoring-assist read of a draft (replacing any
// prior one — one draft analysis per session at a time).
func (e *liveEntry) setAnalysis(a *draftAnalysis) {
	e.findingsMu.Lock()
	e.analysis = a
	e.findingsMu.Unlock()
}

// analysisSnapshot returns the cached draft analysis (nil when none yet).
func (e *liveEntry) analysisSnapshot() *draftAnalysis {
	e.findingsMu.Lock()
	defer e.findingsMu.Unlock()
	return e.analysis
}

// setRewrite stashes the source-rewritten draft for the editor to pick up.
func (e *liveEntry) setRewrite(draft string) {
	e.findingsMu.Lock()
	e.rewrite = draft
	e.findingsMu.Unlock()
}

// rewriteSnapshot returns the latest rewritten draft ("" when none yet).
func (e *liveEntry) rewriteSnapshot() string {
	e.findingsMu.Lock()
	defer e.findingsMu.Unlock()
	return e.rewrite
}

// setPendingHandshake caches (or, given nil, CONSUMES) the handshake
// AuthorHandshake wrote for this session's next live packet.
func (e *liveEntry) setPendingHandshake(h *packet.Handshake) {
	e.findingsMu.Lock()
	e.pendingHandshake = h
	e.findingsMu.Unlock()
}

// pendingHandshakeSnapshot returns the currently authored (not yet consumed)
// handshake for this session, or nil when none has been authored.
func (e *liveEntry) pendingHandshakeSnapshot() *packet.Handshake {
	e.findingsMu.Lock()
	defer e.findingsMu.Unlock()
	return e.pendingHandshake
}

// setComposeMessage leaves (or, given "", clears) an honest inline message
// on the compose card — currently only Send's handshake refusal.
func (e *liveEntry) setComposeMessage(msg string) {
	e.findingsMu.Lock()
	e.composeMessage = msg
	e.findingsMu.Unlock()
}

// composeMessageSnapshot returns the current compose-card message ("" when
// none).
func (e *liveEntry) composeMessageSnapshot() string {
	e.findingsMu.Lock()
	defer e.findingsMu.Unlock()
	return e.composeMessage
}

// resumeSessionID returns the warm harness session id to --resume + --fork-session
// from, or "" when this session has no warm harness yet — so a request before the
// warm-up completes (or on a never-warmed session) runs COLD instead of resuming a
// session that is still being established.
func (e *liveEntry) resumeSessionID() string {
	e.findingsMu.Lock()
	defer e.findingsMu.Unlock()
	if e.harnessWarm {
		return e.harnessSessionID
	}
	return ""
}

// markWarm marks the session's harness as explored and ready, so subsequent requests
// resume its warm context. Called when the warm-up explore run completes.
func (e *liveEntry) markWarm() {
	e.findingsMu.Lock()
	e.harnessWarm = true
	e.findingsMu.Unlock()
}

// beginAnalysis starts a new authoring-assist run, CANCELLING any prior in-flight
// run for this session first — so a fast-typing Lead's superseded reads are killed
// immediately (the latest draft wins) instead of racing to overwrite the cache out
// of order. Returns the context the new run must use; the handler must check its
// Err() after the run and skip writing the cache when it was cancelled.
func (e *liveEntry) beginAnalysis() context.Context {
	e.findingsMu.Lock()
	defer e.findingsMu.Unlock()
	if e.analysisCancel != nil {
		e.analysisCancel() // supersede the prior in-flight analysis
	}
	ctx, cancel := context.WithCancel(context.Background())
	e.analysisCancel = cancel
	return ctx
}

// regSeq is the monotonic source of liveEntry.seq — incremented once per session
// registration so the board's tie-break is deterministic across renders.
var regSeq int64

// defaultSessionKey is the one seeded entry. The registry can hold an entry
// per session key so ≥2 distinct cards can coexist; absent a second registered
// session, every connect falls back to this one entry, so the server behaves as
// a single-card demo (one Lead, one card).
const defaultSessionKey = "default"

// liveReg maps a session key → *liveEntry. Via mounts LiveCard by type (zero-value
// per tab, no constructor injection), so the wiring is stashed here and looked up
// by a connect-derived key. A sync.Map is safe for the concurrent reads
// (View/Spend/OnConnect across tabs) and the connect-time write.
var liveReg sync.Map

// registerSession stores one keyed session's wiring (its own cfg, ledger, and
// admission sem) in the registry. Distinct keys get distinct entries with their
// own *ledger.Log, so ≥2 cards served off the one "/" mount are ISOLATED
// economies — a mint or spend on one key never touches another (the
// farm-denial rule, enforced per session: the faucet is the sole credit
// source and a balance is non-transferable across keys).
func registerSession(key string, cfg LiveConfig, log *ledger.Log) {
	var sem chan struct{}
	if cfg.MaxConcurrent > 0 {
		sem = make(chan struct{}, cfg.MaxConcurrent)
	}
	e := &liveEntry{cfg: cfg, log: log, sem: sem, useContainer: cfg.UseContainer, seq: int(atomic.AddInt64(&regSeq, 1))}
	liveReg.Store(key, e)
	// If claim consumers are already running, a session registered now (e.g. a
	// runtime-created session) gets its consumer immediately, not only those present
	// at boot.
	consumerSpawner.onRegister(key, e)
}

func setLiveState(cfg LiveConfig, log *ledger.Log) {
	registerSession(defaultSessionKey, cfg, log)
}

// claimConsumerSpawner gives each session EXACTLY ONE durable claim consumer — for
// the sessions present when consumers start AND for any session registered later
// (runtime-created sessions), so the create flow is not a dead end for the
// peer path. Birth is guarded by `started` so a session is never double-
// consumed. Once active, registerSession spawns a consumer for each new session
// using the latest StartClaimConsumers parameters.
type claimConsumerSpawner struct {
	mu          sync.Mutex
	active      bool
	ctx         context.Context
	verifierFor func(LiveConfig) ledger.Verifier
	ackWait     time.Duration
	adm         *ledger.Admission
	// socks holds one attached (warm) socket per session key — the socket owns
	// that session's claim-consumer lifecycle (Open starts it, Close stops it).
	// A non-nil entry is the "already spawned" guard (formerly a bool set).
	socks map[string]*socket.Socket
	// parked holds sessions whose verify consumer was auto-parked when idle: the
	// resumable ticket plus the cancel for the lightweight arrival watcher that
	// resumes it on the next claim. A key is in exactly one of socks/parked.
	parked map[string]*parkedConsumer
	// lastActive stamps each session's last activity (claim resolve, spend,
	// send, or spawn); parkIdle parks a warm session idle longer than idleAfter.
	// It is guarded by its OWN lock, never mu: noteActivity fires from the verify
	// consumer goroutine (via the resolve hook), which stop/park JOIN while
	// holding mu — so if activity tracking took mu, a claim resolving during a
	// retire/park would deadlock. activityMu is only ever taken alone or nested
	// under mu (mu -> activityMu), never the reverse.
	lastActive map[string]time.Time
	activityMu sync.Mutex
	// idleAfter is the auto-park threshold; 0 disables parking (the default —
	// auto-park is dormant until StartAutoPark sets it).
	idleAfter time.Duration
	// registry durably records which sessions are parked (their non-secret
	// routing identity), so a parked session survives a restart; nil if it
	// could not be opened, in which case persistence is skipped (best-effort).
	registry *socket.ParkedRegistry
	// now is the clock parkIdle reads; nil means time.Now (injected in tests).
	now func() time.Time
}

// parkedConsumer is a session's parked claim consumer: the single-use ticket
// that Resume redeems, and the cancel that stops its arrival watcher.
type parkedConsumer struct {
	ticket *socket.Ticket
	stop   context.CancelFunc
}

var consumerSpawner claimConsumerSpawner

// resetConsumersForTest clears the package-global claim-consumer state: the
// session registry and the spawner. The live server's wiring lives in process
// globals (liveReg + consumerSpawner) that are never torn down in production
// (one server per process). Tests, however, drive NewServer serially in one
// process, so a prior test's stale registry entries (bound to a now-closed
// fabric) and a still-`active` spawner leak forward: a later test's
// StartClaimConsumers would Range over a stale key and mark it `started`,
// starving the same key's fresh entry of a consumer (a real flaky failure).
// Call this at the start of each consumer test's setup to isolate it.
func resetConsumersForTest() {
	consumerSpawner.mu.Lock()
	defer consumerSpawner.mu.Unlock()
	liveReg.Range(func(k, _ any) bool {
		liveReg.Delete(k)
		return true
	})
	// Per-peer bundle guards are server-lifetime in production but must not
	// leak rate/quota state across tests (they key off session, which tests reuse).
	bundleGuards.Range(func(k, _ any) bool {
		bundleGuards.Delete(k)
		return true
	})
	bundleAcctMu.Lock()
	bundleGlobalRetained = 0
	bundleAcctMu.Unlock()
	// Reset the fields in place — never reassign the struct, which would swap out
	// the mutex this call holds (the deferred Unlock would hit a fresh, unlocked
	// one). Zero everything the spawner carries forward between StartClaimConsumers.
	consumerSpawner.stopAllLocked() // stop any sockets/watchers from a prior test's spawner
	consumerSpawner.active = false
	consumerSpawner.ctx = nil
	consumerSpawner.verifierFor = nil
	consumerSpawner.ackWait = 0
	consumerSpawner.adm = nil
	consumerSpawner.socks = nil
	consumerSpawner.parked = nil
	consumerSpawner.resetActivity(nil)
	consumerSpawner.idleAfter = 0
	consumerSpawner.now = nil
	consumerSpawner.registry = nil
	resetCageGauntletForTest()
}

// spawnLocked opens a socket for key/e unless one is already attached. mu held.
// The socket owns the session's claim consumer: its attach runs ConsumeClaims
// under the socket's cancelable context, so Close (on retire) stops it. The
// spawner fields are copied into locals so the attach closure never reads the
// shared struct fields a later StartClaimConsumers might rewrite.
func (s *claimConsumerSpawner) spawnLocked(key string, e *liveEntry) {
	if s.socks[key] != nil {
		return
	}
	ctx, verifierFor, ackWait, adm := s.ctx, s.verifierFor, s.ackWait, s.adm
	cfg, log := e.cfg, e.log
	sock, err := socket.Open(ctx, liveFabric, key, func(c context.Context) error {
		return log.ConsumeClaims(c, verifierFor(cfg), ackWait, adm)
	})
	if err != nil {
		return // no fabric (or invalid key): no consumer, as before the socket wiring
	}
	s.socks[key] = sock
	s.noteActivity(key)
}

// clock is parkIdle's time source: the injected now, or the wall clock.
func (s *claimConsumerSpawner) clock() time.Time {
	if s.now != nil {
		return s.now()
	}
	return time.Now()
}

// noteActivity stamps a session's last-activity time, keeping an actively-used
// session warm against the idle sweep. It takes ONLY activityMu (never mu), so
// it is safe to call from the verify consumer goroutine that a stop/park join
// waits on. Safe (no-op) before consumers start.
func (s *claimConsumerSpawner) noteActivity(key string) {
	t := s.clock()
	s.activityMu.Lock()
	defer s.activityMu.Unlock()
	if s.lastActive == nil {
		return
	}
	s.lastActive[key] = t
}

// activeSince reports whether key's last activity is after cutoff. Takes only
// activityMu; callers may hold mu (order mu -> activityMu).
func (s *claimConsumerSpawner) activeSince(key string, cutoff time.Time) bool {
	s.activityMu.Lock()
	defer s.activityMu.Unlock()
	last, ok := s.lastActive[key]
	return ok && last.After(cutoff)
}

// forgetActivity drops key's activity stamp. Takes only activityMu.
func (s *claimConsumerSpawner) forgetActivity(key string) {
	s.activityMu.Lock()
	defer s.activityMu.Unlock()
	delete(s.lastActive, key)
}

// resetActivity replaces the activity map (fresh set, or nil to disable).
func (s *claimConsumerSpawner) resetActivity(m map[string]time.Time) {
	s.activityMu.Lock()
	defer s.activityMu.Unlock()
	s.lastActive = m
}

// parkIdle parks every warm session whose consumer has been idle longer than
// idleAfter: it stops the verify consumer (socket.Park), moves the session to
// parked, and starts a lightweight arrival watcher that resumes it on the next
// claim. A zero idleAfter disables parking (auto-park dormant).
func (s *claimConsumerSpawner) parkIdle() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.idleAfter <= 0 {
		return
	}
	cutoff := s.clock().Add(-s.idleAfter)
	for key, sock := range s.socks {
		if s.activeSince(key, cutoff) {
			continue // active within the window
		}
		s.parkKeyLocked(key, sock)
	}
}

// parkKeyLocked parks one warm session: it arms the arrival watcher, stops the
// verify consumer, records the park durably, and hands the watcher the channel.
// The subscription is established BEFORE the park so no claim can slip through
// the gap. mu held. Used by both the idle sweep and boot restore.
func (s *claimConsumerSpawner) parkKeyLocked(key string, sock *socket.Socket) {
	watchCtx, stop := context.WithCancel(s.ctx)
	filter := fabric.EventSubject(key, LedgerInstance, fabric.StatusClaim, ">")
	ch, err := liveFabric.SubscribeNew(watchCtx, filter)
	if err != nil {
		stop()
		return // can't watch: leave it warm rather than park it blind
	}
	ticket, err := sock.Park()
	if err != nil {
		stop()
		return // already closed/parked; drop the just-made subscription
	}
	delete(s.socks, key)
	s.parked[key] = &parkedConsumer{ticket: ticket, stop: stop}
	if s.registry != nil {
		_ = s.registry.Put(socket.ParkedEntry{Addr: key, Session: key, Instance: LedgerInstance})
	}
	go s.watch(watchCtx, key, ch)
}

// watch waits for the first claim on a parked session's subtree (or ctx
// cancellation from retire/reset/shutdown) and resumes the session. The
// subscription is created synchronously in parkIdle, so by the time this runs
// no arriving claim can be missed.
func (s *claimConsumerSpawner) watch(ctx context.Context, key string, ch <-chan fabric.Event) {
	select {
	case <-ctx.Done():
		return
	case _, ok := <-ch:
		if !ok {
			return
		}
		s.resume(key)
	}
}

// resume re-attaches a parked session's consumer on the shared fabric. Called
// by the watcher on a claim arrival; a no-op if the session was retired or
// already resumed while the claim was in flight.
func (s *claimConsumerSpawner) resume(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p := s.parked[key]
	if p == nil {
		return
	}
	delete(s.parked, key)
	p.stop() // tear down this watcher; the resumed consumer takes over
	sock, err := socket.Resume(s.ctx, p.ticket)
	if err != nil {
		return
	}
	s.socks[key] = sock
	s.noteActivity(key)
	if s.registry != nil {
		_ = s.registry.Delete(key) // no longer parked
	}
}

// stopConsumer stops and forgets a session's claim consumer — warm (close the
// socket) or parked (stop the watcher and drop the ticket so it can't resume).
// A key with neither is a no-op. Used when a session is retired.
func (s *claimConsumerSpawner) stopConsumer(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if sock := s.socks[key]; sock != nil {
		_ = sock.Close()
		delete(s.socks, key)
	}
	if p := s.parked[key]; p != nil {
		p.stop()
		delete(s.parked, key)
	}
	s.forgetActivity(key)
	if s.registry != nil {
		_ = s.registry.Delete(key) // retired: drop any persisted parked record
	}
}

// stopAllLocked stops every consumer — warm sockets and parked watchers — and
// empties the sets. mu held.
func (s *claimConsumerSpawner) stopAllLocked() {
	for k, sock := range s.socks {
		_ = sock.Close()
		delete(s.socks, k)
	}
	for k, p := range s.parked {
		p.stop()
		delete(s.parked, k)
	}
}

// onRegister is called after a session is stored in liveReg. If consumers are
// active, the new session gets one immediately — the runtime-create path.
func (s *claimConsumerSpawner) onRegister(key string, e *liveEntry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.active {
		s.spawnLocked(key, e)
	}
}

// StartClaimConsumers activates per-session claim consumers: it spawns one for
// every currently-registered session and arms registerSession to spawn one for
// each session created later (so a runtime-created session is never left without a
// consumer). Idempotent-friendly: each call refreshes the verifier/ackWait/adm and
// re-spawns for any session that does not yet have a consumer under this call.
func StartClaimConsumers(ctx context.Context, verifierFor func(LiveConfig) ledger.Verifier, ackWait time.Duration, adm *ledger.Admission) {
	consumerSpawner.mu.Lock()
	defer consumerSpawner.mu.Unlock()
	consumerSpawner.active = true
	consumerSpawner.ctx = ctx
	consumerSpawner.verifierFor = verifierFor
	consumerSpawner.ackWait = ackWait
	consumerSpawner.adm = adm
	consumerSpawner.socks = map[string]*socket.Socket{} // this call owns a fresh consumer set
	consumerSpawner.parked = map[string]*parkedConsumer{}
	consumerSpawner.resetActivity(map[string]time.Time{})
	if r, err := socket.OpenParkedRegistry(liveFabric); err == nil {
		consumerSpawner.registry = r
	}
	liveReg.Range(func(k, v any) bool {
		consumerSpawner.spawnLocked(k.(string), v.(*liveEntry))
		return true
	})
	consumerSpawner.restoreParkedLocked()
}

// restoreParkedLocked reconciles the durable parked registry against the just-
// spawned warm sessions: a session recorded as parked before a restart comes
// back PARKED (not warm), and a record for a session no longer registered is
// pruned. mu held. Makes the persisted record load-bearing rather than
// write-only. Best-effort: a read error leaves warm sessions as-is.
func (s *claimConsumerSpawner) restoreParkedLocked() {
	if s.registry == nil {
		return
	}
	entries, err := s.registry.List()
	if err != nil {
		return
	}
	for _, e := range entries {
		sock := s.socks[e.Addr]
		if sock == nil {
			if s.parked[e.Addr] == nil {
				_ = s.registry.Delete(e.Addr) // stale: session no longer registered
			}
			continue
		}
		// A claim may have been on the stream unprocessed when the process died.
		// The park-arrival watcher only wakes on NEW claims, so restoring such a
		// session to parked would strand that backlog. Leave it warm to drain
		// (the idle sweep re-parks it once quiet) and drop the parked record.
		if v, ok := liveReg.Load(e.Addr); ok {
			// Keep the session warm if it has a backlog OR if we cannot measure
			// one: parking a session with unprocessed (or unmeasurable) claims
			// would strand them behind the new-claims-only arrival watcher. The
			// idle sweep re-parks it once it is genuinely quiet.
			if n, err := v.(*liveEntry).log.ClaimsInFlight(); err != nil || n > 0 {
				_ = s.registry.Delete(e.Addr)
				continue
			}
		}
		s.parkKeyLocked(e.Addr, sock) // quiet before the restart: restore it parked
	}
}

// StartAutoPark arms the idle sweep: every interval it parks sessions whose
// claim consumer has been idle longer than idleAfter, freeing them until a
// claim wakes them (via the arrival watcher). It is NOT wired into boot yet —
// production stays warm-always until this is called. ctx bounds the sweep and
// every watcher it starts.
func StartAutoPark(ctx context.Context, idleAfter, interval time.Duration) {
	consumerSpawner.mu.Lock()
	consumerSpawner.idleAfter = idleAfter
	consumerSpawner.mu.Unlock()
	go func() {
		t := time.NewTicker(interval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				consumerSpawner.parkIdle()
			}
		}
	}()
}

// LedgerInstance is the subject instance token every session's economy binds to.
// There is one economy per session, so the session key alone demuxes them; the
// instance is a fixed token completing the canonical subject. Exported so any
// out-of-process writer (e.g. a host CLI ACKing a delivery) binds under the
// identical token a running server's own sessions use — a caller-invented
// instance string writes to a wire subject no server projection ever folds,
// silently stranding the write.
const LedgerInstance = "ledger"

// liveFabric is the one embedded JetStream the server's sessions share — the
// single authoritative economy substrate. NewServer
// starts it and gives the primary Log ownership of its lifecycle; AddSession
// binds further sessions to it under their own session token, so each session is
// an ISOLATED economy on the one stream. Set once per server; the live tests
// drive NewServer serially (they share this and liveReg), so it is not guarded.
var liveFabric *fabric.Fabric

// startLiveFabric stands up the shared economy fabric — the host-owned network —
// rooting its durable store beside the configured ledger path (a dedicated dir
// per server, so two servers in one process never share a store). An empty path
// falls back to a temp store. The fabric's lifecycle is the host's.
func startLiveFabric(ledgerPath, listenAddr string, grants []fabric.Grant) (*fabric.Fabric, error) {
	dir := ledgerPath + "-fabric"
	if ledgerPath == "" {
		d, err := os.MkdirTemp("", "packets-fabric-*")
		if err != nil {
			return nil, fmt.Errorf("app: fabric store dir: %v", err)
		}
		dir = d
	} else if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("app: fabric store dir: %v", err)
	}
	// A configured listen address stands the fabric up with an authenticated NATS
	// ingress (the host stays in-process via NoAuthUser; peers authenticate and are
	// grant-confined). Absent it, the fabric is in-process-only — no auth surface.
	// Repo endpoints attach onto this fabric later, at registration, via
	// internal/socket — the fabric is the network, owned here, not by any socket.
	if listenAddr != "" {
		return fabric.StartListening(context.Background(), dir, listenAddr, grants...)
	}
	return fabric.Start(context.Background(), dir)
}

// AddSession binds a session's economy to the shared fabric and registers it
// under key, so the one "/" mount also serves /?key=<key> with its OWN isolated
// economy (its own session subtree on the stream + admission sem). The returned
// Log does not own the fabric, so its Close is a no-op; the fabric's lifecycle
// belongs to the primary Log from NewServer. This is the wiring entry the command
// uses to stand up a SECOND review target beyond the default card; the core keyed
// registration + cross-session isolation is registerSession, exercised by the
// live tests.
func AddSession(key string, cfg LiveConfig) (*ledger.Log, error) {
	if liveFabric == nil {
		return nil, fmt.Errorf("app: AddSession before NewServer started the fabric")
	}
	if !fabric.ValidToken(key) {
		return nil, fmt.Errorf("app: session key %q is not a valid subject token", key)
	}
	log := ledger.Bind(liveFabric, key, LedgerInstance)
	registerSession(key, cfg, log)
	return log, nil
}

// lookupLiveEntry resolves a session key to its entry, falling back to the default
// session when the key isn't registered — so a connect whose key has no dedicated
// entry still drives the one seeded session (behavior-preserving while only
// defaultSessionKey is seeded). Returns nil only if nothing is registered at all.
func lookupLiveEntry(key string) *liveEntry {
	if v, ok := liveReg.Load(key); ok {
		return v.(*liveEntry)
	}
	if v, ok := liveReg.Load(defaultSessionKey); ok {
		return v.(*liveEntry)
	}
	return nil
}

func readLiveState(key string) (LiveConfig, *ledger.Log) {
	if e := lookupLiveEntry(key); e != nil {
		return e.cfg, e.log
	}
	return LiveConfig{}, nil
}

func cycleSem(key string) chan struct{} {
	if e := lookupLiveEntry(key); e != nil {
		return e.sem
	}
	return nil
}

// LiveCard is the served review card. On connect it renders the in-flight state
// immediately, runs the catch cycle in the background, and resolves the card in
// place over SSE when the verdict lands — so a human watches one verdict go
// in-flight → resolved, with the catch (if any) appended to the ledger.
type LiveCard struct {
	// Key selects the session this card drives — its registry entry (cfg, ledger,
	// sem). It is decoded from the ?key= query slot into the per-connection
	// instance (Via persists it per tab and re-decodes it on action POSTs). An
	// empty Key (the "/" route, no ?key) falls back to defaultSessionKey via the
	// registry lookup — so the single-card "/" wire is byte-identical.
	Key     string `query:"key"`
	Verdict via.StateTabStr
	Land    via.StateTabStr
	Beats   via.StateTabStr
	// Questions broadcasts the count of open review questions — the fix oracle's
	// surviving/undetermined mutants — so the card shows a gated "N open questions"
	// badge when the verdict's green hides unkilled mutants. Written by OnConnect
	// after the cycle; the full anchored threads live on the /review surface.
	Questions via.StateTabStr
	// FundTarget carries the path:line of the bench item the Lead clicked to fund —
	// set by that item's on.SetSignal just before the post, then read by FundChosen
	// to send the CHOSEN target instead of the FIFO head.
	FundTarget via.SignalStr `via:"fundtarget"`
	// Draft carries the free-form task the Lead authored in the compose control,
	// read by Send to fund a prompt-carrying live packet (vs drawing a pre-baked
	// backlog target). Per-tab signal, not authoritative session state.
	Draft via.SignalStr `via:"draft"`
	// HandshakeDraft/HandshakeStrengthPick carry the handshake the
	// Lead composes BEFORE placing a live packet — a runnable contract authored
	// independently of the agent's own code, written under the protected handshake/
	// directory (internal/settle's deny-rule then refuses any later agent turn that
	// touches it). The strength is SELF-DECLARED, never scored: AuthorHandshake reads
	// these verbatim and refuses (writes nothing) on a blank draft or an
	// unrecognized/blank strength pick — it never guesses or defaults one. Per-tab
	// signals, not authoritative session state.
	HandshakeDraft        via.SignalStr `via:"handshakedraft"`
	HandshakeStrengthPick via.SignalStr `via:"handshakestrengthpick"`
	// HandshakeAuthored is the broadcast trigger for a successfully authored
	// handshake: View re-reads the pending handshake from the session entry (not
	// reactive), so AuthorHandshake writes here to fan out a re-render to every
	// connected tab, mirroring Analysis/Rewrite's pattern.
	HandshakeAuthored via.StateTabStr
	// LandOverride ("1"/"true") lets Approve open a PR despite a guard block (open
	// threads / red checks) — deliberate, overridable friction (DESIGN §16).
	LandOverride via.SignalStr `via:"landoverride"`
	// Landed is the Approve broadcast trigger: View re-reads the land result from the
	// session entry (not reactive), so Approve writes here to fan out the re-render.
	Landed via.StateTabStr
	// RefineTarget/RefineKind/RefineText carry a bench card's SHARPEN inputs: the
	// path:line being refined, the kind (criteria | convention), and the free text
	// (criteria one-per-line, or the convention note). Read by RefineChosen, which
	// appends a worefine fact for that target. Per-tab signals, not session state.
	RefineTarget via.SignalStr `via:"refinetarget"`
	RefineKind   via.SignalStr `via:"refinekind"`
	RefineText   via.SignalStr `via:"refinetext"`
	// Balance is the spend broadcast trigger: the balance ROW value is re-read
	// from the ledger in View (the source of truth), but the ledger is not
	// reactive — so Spend writes the new balance here to make the live SSE stream
	// re-render (a cell Write fans out a re-render; an action's auto-render only
	// returns in the action's own response).
	Balance via.StateTabStr
	// Sends is the same broadcast trigger for the sent-work tally: the
	// count is re-read from the ledger in View, but a Spend writes the new count
	// here so the sends row rises over the live SSE stream in the SAME render as
	// the balance drains. It carries no authoritative value — View is the source.
	Sends via.StateTabStr
	// BandwidthMeter is the spend broadcast trigger for the attention-bandwidth row,
	// mirroring Balance: View re-reads the meter from the ledger (the source of truth),
	// but the ledger is not reactive — so Send writes the new bandwidth here to
	// fan out a live SSE re-render as the meter drains.
	BandwidthMeter via.StateTabStr
	// Analysis is the broadcast trigger for the authoring assist: View re-reads the
	// cached draft analysis from the session entry, but that cache is not reactive — so
	// AnalyzeDraft writes here to fan out a re-render once the source's read lands.
	Analysis via.StateTabStr
	// DraftAnswers carries the Lead's answers to the analysis questions — a JSON array
	// of {Q, Answers, Note} the Update-draft control gathers from the answer form, read
	// by UpdateDraft to build the source's rewrite prompt. Per-tab signal.
	DraftAnswers via.SignalStr `via:"draftanswers"`
	// Rewrite is the broadcast trigger for a draft rewrite: View re-reads the rewritten
	// draft from the session entry (not reactive), so UpdateDraft writes here to fan out
	// the re-render that swaps the new draft into the editor's rewrite payload.
	Rewrite via.StateTabStr
	// FillBeats is a re-render trigger written by the Stream when the live-fill buffer
	// (a currently-filling packet's accruing beats) changes, so the card shows the
	// packet filling live. View reads the buffer; this cell only nudges the re-render.
	FillBeats via.StateTabStr
	// Bench is the re-render trigger for a sharpening: RefineChosen writes the current
	// refinement count here so the bench re-renders (a split re-folds fundableBacklog;
	// criteria/convention re-render the card body). It carries no authoritative value
	// — View re-reads the refinements — and changes on every append (the count rises),
	// so the frame always fans out even when the fundable target list is unchanged.
	Bench via.StateTabStr
	// MarkWatchKind/MarkWatchWO/MarkUseful carry a standing watch's mark prompt
	// (standing inspection): which WatchKind (its int value, as a
	// string), which packet id, and the human's usefulness judgment ("true"/
	// "false"). Read by MarkWatchFire, which finds that (kind, packet)'s
	// unmarked fire and records the judgment — Precision is computed from real
	// human marks, never inferred. Per-tab signals, not authoritative state.
	MarkWatchKind via.SignalStr `via:"markwatchkind"`
	MarkWatchWO   via.SignalStr `via:"markwatchwo"`
	MarkUseful    via.SignalStr `via:"markuseful"`
}

// MarkWatchFire records a human's usefulness judgment on the unmarked fire
// for the given standing-watch kind + packet. A malformed kind/packet id
// or an unknown session is a calm no-op, mirroring ConfirmIntentFidelity's
// handling of bad input.
func (c *LiveCard) MarkWatchFire(ctx *via.Ctx) {
	e := lookupLiveEntry(c.Key)
	if e == nil {
		return
	}
	kindInt, err := strconv.Atoi(c.MarkWatchKind.Read(ctx))
	if err != nil {
		return
	}
	packetID, err := strconv.Atoi(c.MarkWatchWO.Read(ctx))
	if err != nil || packetID <= 0 {
		return
	}
	e.markWatchFire(packet.WatchKind(kindInt), packetID, c.MarkUseful.Read(ctx) == "true")
}

// View renders the card's rows via the shared surface rendering: the retrospective
// confirmed-catch STOCK (re-derived read-only from the ledger on every render — the
// economy finally SHOWN, not just logged), the streamed beat row (the felt tempo),
// the oracle verdict row, and the integration (Land) row. One row never speaks for
// another. The stock is read-only: a ledger read failure degrades to an empty
// stock, never breaks the card.
func (c *LiveCard) View(ctx *via.CtxR) h.H {
	cfg, log := readLiveState(c.Key)
	// Nothing to work at this key — neither a repo (prompt-authoring) nor an anchor
	// (catch-cycle): render a calm landing pointing at the fleet board. A repo-only OR
	// an anchored session is usable and renders the working card below.
	if log == nil || (!cfg.hasRepo() && !cfg.hasAnchor()) {
		return h.Div(navHeader("", "console"),
			h.Div(h.Role("main"), h.Attr("aria-label", "no addr"),
				h.Div(h.Class("pk-card onboarding"), h.Data("state", "empty"),
					h.P(h.Class("onboarding__lead"), h.Text("No addr configured.")),
					h.P(h.Class("onboarding__step"),
						h.Text("Open the "),
						h.A(h.Href("/board"), h.Text("fleet")),
						h.Text(" to create or pick one.")),
				),
			),
		)
	}
	var stock ledger.Stock
	balance := 0
	bandwidth := 0
	var sends []ledger.SendView
	if log != nil {
		if recs, err := log.Records(); err == nil {
			stock = ledger.ConfirmedCatches(recs)
		}
		if b, err := log.Balance(); err == nil {
			balance = b
		}
		if bw, err := log.Bandwidth(); err == nil {
			bandwidth = bw
		}
		// This session's recent funded packets + their caught/missed outcome —
		// the round-trip the Lead watches after a Spend, on the same card they act on.
		if ds, err := log.RecentSends(5); err == nil {
			sends = ds
			// Enrich each with its open-question count (the packet's reviewable
			// test-debt) from the per-packet findings cache — off-ledger diagnostic,
			// so it's filled here, not projected.
			if e := lookupLiveEntry(c.Key); e != nil {
				for i := range sends {
					sends[i].Questions = e.orderQuestionCount(sends[i].ID)
				}
			}
		}
	}
	// The "/" card with no ?key IS the default session — name it honestly in the
	// breadcrumb rather than leave the crumb keyless.
	navKey := c.Key
	if navKey == "" {
		navKey = defaultSessionKey
	}
	// The economy region (everything below the nav) is the page's main content and a
	// LIVE region: this card re-renders over SSE on every catch/balance/send
	// change, so role="main" + aria-live="polite" lets assistive tech announce those
	// changes without the user hunting for them. The nav is a sibling landmark (added
	// in the final wrap), never nested inside main.
	parts := []h.H{
		h.Role("main"),
		h.Attr("aria-live", "polite"),
		h.Attr("aria-label", "packet activity"),
	}
	// A brand-new session gets a calm onboarding affordance ahead of the (all-zero)
	// economy rows, so a first-run Lead sees the next action, not a dead screen.
	if hint := onboardingHint(stock, cfg.hasAnchor()); hint != nil {
		parts = append(parts, hint)
	}
	// The card splits into two sub-landmarks INSIDE main (Flow A): the ACT-NOW region
	// gathers the moves the Lead makes right now (fund work, author + place a packet);
	// the STATE/HISTORY region carries the retrospective economy (stock, balance,
	// sends, beats, verdict, land). The split is carried by the section headings
	// + landmark roles, NOT a new background layer: --pk-surface-3 is SKIPPED (gate,
	// §1) — the per-row .pk-card elevation plus the labelled regions already separate
	// the two without a third elevation token without another consumer.
	var actNow []h.H
	// Flow B: spend (balance hue) and place-packet (bandwidth hue) both FUND work, so
	// they sit under one "fund work" group with a two-currency explainer, never read
	// as unrelated controls.
	if fund := renderFundWork(c, cfg, log, balance, bandwidth); fund != nil {
		actNow = append(actNow, fund)
		// The agent-runner control sits with the funding controls — it governs how a
		// PLACED live packet runs (host vs container). Gated on fund-work being present
		// so a fresh, no-currency session keeps its act-now omitted (onboarding shown).
		if e := lookupLiveEntry(c.Key); e != nil {
			actNow = append(actNow, renderRunnerControl(c, e.useContainerMode()))
		}
	}
	// The prep bench: the fundable work on deck, so the Lead sees (and, in a later
	// slice, curates) what a Spend funds rather than a blind auto-pick. Omitted when
	// there is no fundable work; guarded on log (fundableBacklog reads it).
	if log != nil {
		if bench := renderBench(c, fundableBacklog(cfg, log), benchAnnotations(log)); bench != nil {
			actNow = append(actNow, bench)
		}
	}
	// Approve & open a PR: the land control closes the goal flow (review -> PR). Shown
	// only when there is landable work (a sent packet) or a prior land result to
	// surface — so a fresh, empty session keeps its act-now omitted (onboarding shown).
	if log != nil {
		lk := c.Key
		if lk == "" {
			lk = defaultSessionKey
		}
		if landResultSnapshot(lk) != "" || sessionHasSends(log) {
			actNow = append(actNow, renderLandControl(c))
		}
	}
	if len(actNow) > 0 {
		section := []h.H{
			h.Attr("aria-labelledby", "act-now-label"),
			h.Span(h.Class("pk-section-label"), h.ID("act-now-label"), h.Text("act now")),
		}
		parts = append(parts, h.Section(append(section, actNow...)...))
	}
	// The economy meter rows (stock/balance/bandwidth/send) are RETIRED from
	// the UI (the vocabulary map) — the underlying ledger
	// reads above still feed renderFundWork/onboarding (their reframe is a
	// separate effort), but nothing renders them as a row here anymore.
	state := []h.H{
		h.Attr("aria-labelledby", "state-history-label"),
		h.Span(h.Class("pk-section-label"), h.ID("state-history-label"), h.Text("state & history")),
	}
	// WATCH IT FILL: when the background runner is mid-fill on a packet, show it live
	// — the packet id + the cycle beats accruing as the oracle works (re-rendered each
	// Stream tick via the FillBeats poll). Omitted when nothing is filling.
	if e := lookupLiveEntry(c.Key); e != nil {
		if id, fb := e.fillSnapshot(); id > 0 {
			state = append(state, h.Div(
				h.Class("packet-filling"),
				h.Data("state", "beats"),
				h.Text("filling PKT#"+strconv.Itoa(id)+" — "+strings.Join(fb, " → ")),
			))
			// The live agent's LATEST move (a single updating line) while it works —
			// distinct from the oracle's cycle beats above. Absent on dead-air (no beat
			// yet) so silence stays honest, no spinner.
			if act := e.activitySnapshot(); act != "" {
				state = append(state, h.Div(
					h.Class("order-activity"),
					h.Data("state", "activity"),
					h.Text("· "+act),
				))
			}
			// The scrolling TRANSCRIPT: every beat so far, in order, so the Lead watches
			// the run unfold rather than only its latest move. Re-rendered each Stream
			// tick (same poll as the latest-line); the CSS bounds its height and scrolls
			// it. Omitted until there is a beat — no empty pane.
			if tr := e.activityTranscript(); len(tr) > 0 {
				lines := []h.H{h.Class("packet-transcript"), h.Data("state", "transcript"),
					h.Attr("aria-label", "agent transcript")}
				for _, line := range tr {
					lines = append(lines, h.Div(h.Class("packet-transcript__line"), h.Text(line)))
				}
				state = append(state, h.Div(lines...))
			}
		}
	}
	// Below the aggregate counts, the per-packet round-trip: each recent packet
	// with its caught/missed outcome, so the Lead watches the packet they funded
	// resolve in place (omitted when there are none — same helper the board uses).
	if d := renderSends(navKey, sends); d != nil {
		state = append(state, d)
	}
	// The catch-cycle surface (beats, verdict, review-questions badge, land verdict)
	// renders ONLY for an anchored session — it is the OnConnect cycle's output. A
	// repo-only session runs no cycle, so showing the verdict here would mean a phantom
	// "Oracle running…" spinner with nothing behind it.
	if cfg.hasAnchor() {
		state = append(state,
			surface.RenderBeats(c.Beats.Read(ctx)),
			surface.RenderVerdict(c.Verdict.Read(ctx)),
		)
		// A gated, calm badge: when the oracle left surviving mutants, the verdict's
		// green hides honest test gaps — show the open-question count (the full anchored
		// threads live on /review). Omitted when there are none.
		if b := reviewQuestionsBadge(c.Questions.Read(ctx), navKey); b != nil {
			state = append(state, b)
		}
		state = append(state, surface.RenderLand(pipe.LandState(c.Land.Read(ctx))))
	}
	parts = append(parts, h.Section(state...))
	// nav landmark first, then the Console grid: a needs-you
	// rail of this session's HELD packets, the untouched center content (behind
	// a hero stat + in-flight strip), and a settled+watches rail — every count
	// folded from the SAME packet slice, never a second source of truth.
	packets := sessionPackets(c.Key, 0)
	reconcileHolds(c.Key, packets)
	if e := lookupLiveEntry(c.Key); e != nil {
		e.recordWatchFires(packets)
	}
	return h.Div(navHeader(navKey, "console"),
		renderConsole(c, navKey, packets, sessionAddr(c.Key), h.Div(parts...)))
}

// reconcileHolds attaches each packet's ALREADY-cached Lane/Gauntlet (pure
// map reads — cachedLane/cachedGauntlet, never the exec'ing laneFor/
// gauntletFor) and reconciles its hold via packet.ReconcileHold,
// in place. This is the ONE call site that hands a packets slice
// into renderConsole, which is the only render path reading p.Hold/
// p.HoldReason (the needs-you and settled rails) — so this is also the only
// place that needs to run it. It must stay safe on the 100ms via.Stream poll
// (View is re-invoked on every tick): both cache reads are pure lookups, so
// this adds no exec to the poll path, only cheap in-memory composition. An
// unknown session (e == nil) leaves packets exactly as Fold set them.
func reconcileHolds(key string, packets []packet.Packet) {
	e := lookupLiveEntry(key)
	if e == nil {
		return
	}
	for i, p := range packets {
		p.Lane = e.cachedLane(p.ID)
		p.Gauntlet = e.cachedGauntlet(p.ID)
		packets[i] = packet.ReconcileHold(p)
	}
}

// Spend funds one unit of sent work against the balance — the Lead's first
// ACTION on the stock, and the moment a catch finally BUYS something. It debits
// one catch AND fuels exactly one queued packet in a single atomic ledger
// fact (AppendSend). An over-budget spend (balance already 0) is refused by
// the ledger and the action is a silent no-op (no broadcast). On success it
// writes BOTH the drained balance and the risen send count to their trigger
// cells, whose Writes fan out a single re-render to the live SSE stream so the
// balance drains and the send row rises together — the spend is visibly
// converted into work, not just a vanishing number.
func (c *LiveCard) Spend(ctx *via.Ctx) {
	cfg, log := readLiveState(c.Key)
	if log == nil {
		return
	}
	tgt, ok := nextUnconsumedTarget(cfg, log)
	if !ok {
		return // backlog exhausted / empty: no distinct work to buy — a silent no-op
	}
	if err := log.AppendSend("dispatch", tgt, ownTargetOf(cfg)); err != nil {
		return // over-budget / nothing to spend / own work: a no-op, never an error to the Lead
	}
	if b, err := log.Balance(); err == nil {
		c.Balance.Write(ctx, strconv.Itoa(b)) // announce the drain
	}
	if d, err := log.PendingSends(); err == nil {
		c.Sends.Write(ctx, strconv.Itoa(d)) // announce the funded packet so the sends row rises in the same render
	}
	go drainQueuedPackets(c.Key) // the packet RUNS in the background — spend-to-earn
}

// FundChosen is the prep bench's payoff: it funds the CHOSEN bench target (set by
// that item's on.SetSignal into FundTarget) instead of the FIFO head, turning
// sending from a blind auto-pick into the Lead's management-sim decision. The
// chosen target is VALIDATED to be in the fundable set (chosenFundable), so a click
// can never fund the card's own cycle, an already-consumed target, or an arbitrary
// one — the distinct-work / two-scores rules hold. Otherwise it mirrors Spend: one
// atomic AppendSend debit, then announce the drain + risen send over SSE and
// run the packet. An off-bench key or over-budget balance is a silent no-op.
func (c *LiveCard) FundChosen(ctx *via.Ctx) {
	cfg, log := readLiveState(c.Key)
	if log == nil {
		return
	}
	tgt, ok := chosenFundable(cfg, log, strings.TrimSpace(c.FundTarget.Read(ctx)))
	if !ok {
		return // not on the bench (unknown / consumed / own cycle): a no-op
	}
	if err := log.AppendSend("dispatch", tgt, ownTargetOf(cfg)); err != nil {
		return // over-budget / nothing to spend: a no-op, never an error to the Lead
	}
	if b, err := log.Balance(); err == nil {
		c.Balance.Write(ctx, strconv.Itoa(b))
	}
	if d, err := log.PendingSends(); err == nil {
		c.Sends.Write(ctx, strconv.Itoa(d))
	}
	go drainQueuedPackets(c.Key)
}

// Send authors a LIVE packet from the card: it funds a prompt-carrying target
// (the Lead's free-form task) against the balance and sends it, so the live
// harness runs the authored work — the UI counterpart of the -live CLI flag, but
// composed at runtime instead of baked at boot. The base is the repo's CURRENT
// HEAD, so the agent works the live tree. An empty prompt, an unconfigured repo, or
// an over-budget balance is a silent no-op (never a funded packet with no task, no
// tree, or no catch to spend). A live packet additionally REQUIRES a
// handshake authored first (AuthorHandshake) — the contract is authored
// independently of, and before, the agent's own code, so a live
// packet with none is refused (funding nothing) and leaves an honest inline message
// for the Lead, rather than sending an agent turn with no contract to gate it. A
// PRE-FUNDED packet (FundChosen, Prompt=="") predates this concept and is untouched.
// On success the handshake is folded into the Target and CONSUMED (a later packet
// must author its own), and it mirrors Spend: announce the drained balance + risen
// send over SSE, then run the packet in the background.
func (c *LiveCard) Send(ctx *via.Ctx) {
	cfg, log := readLiveState(c.Key)
	if log == nil {
		return
	}
	prompt := strings.TrimSpace(c.Draft.Read(ctx))
	if prompt == "" {
		return // an empty prompt is not a packet
	}
	e := lookupLiveEntry(c.Key)
	var hs *packet.Handshake
	if e != nil {
		hs = e.pendingHandshakeSnapshot()
	}
	if hs == nil {
		if e != nil {
			e.setComposeMessage("author a handshake before sending")
		}
		return // no handshake authored yet — refuse rather than send an ungated live packet
	}
	head, ok := repoHead(cfg.RepoDir)
	if !ok {
		return // no resolvable tree to run the agent against — never send a treeless live packet
	}
	tgt := ledger.Target{
		BaseRev: head, Prompt: prompt,
		HandshakePath: hs.Path, HandshakeHash: hs.Hash, HandshakeStrength: int(hs.Strength),
	}
	// A UI-authored live packet is funded by ATTENTION bandwidth, not a catch — the
	// responsiveness the Lead earned by unblocking work funds the autonomous work
	// they send (the two meters, both used). An over-budget meter is refused by
	// the ledger and is a silent no-op.
	if err := log.AppendLiveSend("liveorder", tgt, ownTargetOf(cfg)); err != nil {
		return
	}
	// The handshake is CONSUMED by the packet it was authored for — a later packet
	// must author its own, never silently reuse this one.
	e.setPendingHandshake(nil)
	e.setComposeMessage("")
	if bw, err := log.Bandwidth(); err == nil {
		c.BandwidthMeter.Write(ctx, strconv.Itoa(bw)) // announce the drained meter
	}
	if d, err := log.PendingSends(); err == nil {
		c.Sends.Write(ctx, strconv.Itoa(d))
	}
	go drainQueuedPackets(c.Key)
}

// ToggleRunner switches this session's live-packet runner between the host subprocess
// (default) and the hardened agent container — surfacing the built RunContainer path
// (previously reachable only via the -container boot flag) as a runtime choice. The
// NEXT live packet uses the new mode; an in-flight one is unaffected. The card
// auto-renders the new mode in the action's response.
func (c *LiveCard) ToggleRunner(ctx *via.Ctx) {
	if e := lookupLiveEntry(c.Key); e != nil {
		e.toggleRunner()
	}
}

// renderRunnerControl shows the session's live-packet runner mode (host / container)
// and a toggle — a calm act-now control surfacing the built container runner without
// requiring a boot flag. Stripped-CSS legible (the mode is plain words).
func renderRunnerControl(c *LiveCard, useContainer bool) h.H {
	mode, toggleTo := "host", "run in container"
	if useContainer {
		mode, toggleTo = "container", "run on host"
	}
	return h.Div(h.Class("live-runner"),
		h.Span(h.Class("live-runner__mode"), h.Text("agent runner: "+mode)),
		h.Button(on.Click(c.ToggleRunner), h.Class("pk-btn--quiet live-runner__toggle"), h.Text(toggleTo)),
	)
}

// repoHead resolves repoDir's current HEAD, the base a UI-authored live packet runs
// from. An empty dir or an unresolvable HEAD (no repo / no commit) reports false so
// Send refuses rather than send a treeless packet.
func repoHead(repoDir string) (string, bool) {
	if repoDir == "" {
		return "", false
	}
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = repoDir
	out, err := cmd.Output()
	if err != nil {
		return "", false
	}
	head := strings.TrimSpace(string(out))
	return head, head != ""
}

// chosenFundable resolves a "path:line" bench key to the matching fundable target,
// validating membership: only a target currently in fundableBacklog can be chosen,
// so the own-cycle target and already-consumed work are never fundable by key.
func chosenFundable(cfg LiveConfig, log *ledger.Log, key string) (ledger.Target, bool) {
	if key == "" {
		return ledger.Target{}, false
	}
	for _, t := range fundableBacklog(cfg, log) {
		if t.Path+":"+strconv.Itoa(t.Line) == key {
			return t, true
		}
	}
	return ledger.Target{}, false
}

// maxPacketAttempts bounds how many times the runner will pick a single queued
// packet before giving up on it. A status write that fails permanently (e.g. a
// closed ledger handle) would otherwise leave a packet forever queued and spin
// the suite-exec loop without end; the cap turns that into a bounded, abandoned
// packet instead of an unbounded #15-multiplier burn.
const maxPacketAttempts = 3

// drainQueuedPackets runs every queued packet for a session to completion — the
// second in-process source. It serializes per session (runMu) so two concurrent
// Spends never double-run a packet. Each packet: mark running, run its DISTINCT
// target through the catch cycle under the admission sem (bounding the suite-exec
// cost), route any Catch through the idempotent Append stamped with the packet's
// source (a re-run that reproduces a seen identity mints nothing — an honest
// loss), then mark done. The mint is the only thing logged; intermediate beats
// stay off-ledger. A packet whose status can never advance is retried at most
// maxPacketAttempts times then GIVEN UP (a best-effort terminal "failed" line, so
// it leaves the queued set when the log is writable), guaranteeing the drain
// always returns.
func drainQueuedPackets(key string) {
	consumerSpawner.noteActivity(key) // a spend/send/adjust keeps the session warm against the idle sweep
	e := lookupLiveEntry(key)
	if e == nil || e.log == nil {
		return
	}
	e.runMu.Lock()
	defer e.runMu.Unlock()
	attempts := map[int]int{}
	givenUp := map[int]bool{}
	for {
		queued, err := e.log.QueuedPackets()
		if err != nil {
			return
		}
		var pkt *ledger.PacketRecord
		for i := range queued {
			if !givenUp[queued[i].ID] {
				pkt = &queued[i]
				break
			}
		}
		if pkt == nil {
			return // nothing left that hasn't been given up
		}
		attempts[pkt.ID]++
		if attempts[pkt.ID] > maxPacketAttempts {
			givenUp[pkt.ID] = true
			_ = e.log.AppendStatus(pkt.ID, "failed") // best-effort terminal line; if this too fails, givenUp still bounds the loop
			continue
		}
		if pkt.Target.Prompt != "" {
			runLivePacket(e, *pkt)
		} else {
			runOneOrder(e, *pkt)
		}
	}
}

// runLivePacket fills a LIVE packet: a real Claude Code harness runs the
// packet's task prompt and PRODUCES the fix revision in the repo (vs the
// pre-funded base→fix diff runOneOrder replays). It mints NO catch — the
// oracle/catch step on the produced revision is a later slice; this settles only
// the agent's git revision, keeping the catch economy untouched (the firewall).
// A terminal status is always reached ("done" on success, "failed" on a harness
// error) so the packet never lingers mid-flight: once it leaves "queued" the
// drain's attempts cap no longer sees it, so the terminal write must happen here.
func runLivePacket(e *liveEntry, pkt ledger.PacketRecord) {
	if err := e.log.AppendStatus(pkt.ID, "running"); err != nil {
		return // could not advance status — the packet stays queued; the drain retries under the attempts cap
	}
	if e.sem != nil {
		e.sem <- struct{}{}
		defer func() { <-e.sem }()
	}
	e.startFill(pkt.ID)
	defer e.endFill()
	// Bound the agent run so a runaway harness can't burn the budget without limit
	// (the cost-gate — the only token cap a live packet has).
	hctx, cancel := context.WithTimeout(context.Background(), liveHarnessTimeout)
	// Resume the session's WARM explored harness (forking a branch) so the fill works
	// with the repo context the warm-up built — the remembered session, same as the
	// analyze reads. "" until the warm-up completes (then the fill cold-starts).
	hctx = harness.WithResume(hctx, e.resumeSessionID())
	runner := runHarness // host subprocess by default; the container when the session opts in
	if e.useContainerMode() {
		runner = runHarnessContainer
	}
	turns, err := runner(hctx, e.cfg.RepoDir, pkt.Target.Prompt, func(evs []translate.UIEvent) {
		if len(evs) > 0 {
			e.addActivityBeat(formatActivity(evs[len(evs)-1])) // the latest event = the agent's current move
		}
	})
	cancel()
	if err != nil {
		_ = e.log.AppendStatus(pkt.ID, "failed") // the live run failed — terminal, not a completed fill
		return
	}
	// Run the catch cycle on the agent-PRODUCED revision, against the packet's
	// PRE-SPECIFIED anchor (Target.Path/Line) — never an anchor derived from the
	// agent's own diff, which would let it farm confirmed-catches (the
	// anti-farming firewall).
	if liveHead, ok := lastMintedSHA(turns); ok {
		beats := make(chan pipe.TraceEvent, 64)
		go func() {
			for ev := range beats {
				e.addFillBeat(ev.Kind)
			}
		}()
		res, cerr := resolveCycle(context.Background(), e.cfg.RepoDir,
			pkt.Target.BaseRev, liveHead, liveHead,
			anchorFromTarget(pkt.Target), e.cfg.TestCmd, false, false, beats)
		close(beats)
		settleCatch(e, pkt.ID, res, cerr)
	}
	_ = e.log.AppendStatus(pkt.ID, "done")
}

// liveHarnessTimeout bounds one live agent run — the runaway-token cost-gate.
const liveHarnessTimeout = 10 * time.Minute

// formatActivity renders one agent activity event as a human-legible line for the
// card's "latest activity" indicator — "thinking", "editing <file>", "running
// <cmd>" — falling back to the detail (or kind) for an unrecognized beat.
func formatActivity(e translate.UIEvent) string {
	switch e.Kind {
	case "thinking":
		return "thinking"
	case "editing":
		return "editing " + e.Detail
	case "tool":
		return "running " + e.Detail
	default:
		if e.Detail != "" {
			return e.Detail
		}
		return e.Kind
	}
}

func runOneOrder(e *liveEntry, pkt ledger.PacketRecord) {
	if err := e.log.AppendStatus(pkt.ID, "running"); err != nil {
		return // could not advance the packet's status — don't run; the drain loop retries under the attempts cap
	}
	if e.sem != nil {
		e.sem <- struct{}{}
		defer func() { <-e.sem }()
	}
	// Accrue the cycle's beats into the live-fill buffer so the card can show this
	// packet filling LIVE ("watch it fill"). The runner has no request ctx to write
	// the card's cells; it writes the buffer and the card's Stream polls it.
	e.startFill(pkt.ID)
	beats := make(chan pipe.TraceEvent, 64)
	go func() {
		for ev := range beats {
			e.addFillBeat(ev.Kind)
		}
	}()
	res, err := resolveCycle(context.Background(), e.cfg.RepoDir,
		pkt.Target.BaseRev, pkt.Target.FixRev, pkt.Target.TipRev,
		anchorFromTarget(pkt.Target), e.cfg.TestCmd, false, false, beats)
	close(beats) // the cycle only SENDS on beats; the caller owns the close, so the accrue goroutine exits (mirrors OnConnect)
	settleCatch(e, pkt.ID, res, err)
	_ = e.log.AppendStatus(pkt.ID, "done")
	e.endFill() // the packet is done — clear the live filling row; its outcome takes over
}

// settleCatch persists a catch cycle's result for a packet: the minted catch (the
// only economy write — attributed to wo:<id>, deduped on a re-run of a seen
// identity), the oracle's verdict (diagnostic — the WHY behind a catch or miss),
// and the surviving-mutant findings (diagnostic — the send→review tie). The
// verdict and findings are OFF the two-scores economy; only res.Record mints. A
// cycle error settles nothing.
func settleCatch(e *liveEntry, orderID int, res Resolution, err error) {
	if err != nil {
		return
	}
	if res.Record != nil {
		res.Record.Source = "wo:" + strconv.Itoa(orderID)
		_ = e.log.Append(*res.Record)
	}
	_ = e.log.AppendPacketVerdict(orderID, res.Verdict)
	e.setOrderFindings(orderID, res.Findings)
	e.setOrderCatchOutcome(orderID, res.Outcome, res.AfterSurvivors, res.AfterConsidered)
}

// lastMintedSHA returns the SHA of the last turn that minted a revision — the
// live packet's "fix revision" — or ok=false when the agent committed nothing
// (so the caller skips the catch cycle: there is no revision to check).
func lastMintedSHA(turns []harness.Turn) (string, bool) {
	for i := len(turns) - 1; i >= 0; i-- {
		if turns[i].Outcome.Minted {
			return turns[i].Outcome.SHA, true
		}
	}
	return "", false
}

// anchorFromTarget reconstructs the re-anchor anchor a funded packet runs against
// from the target's persisted rev/anchor fields.
func anchorFromTarget(t ledger.Target) reanchor.Anchor {
	return reanchor.Anchor{Path: t.Path, Start: t.Line, End: t.Line, LineHash: t.LineHash}
}

// OnConnect kicks off the catch cycle and streams its beats live: each pipe
// transition (settle-base → oracle-base → … → catch → land) is flushed to the
// beat row as it happens, and the verdict + Land rows resolve only when the cycle
// completes. So the human feels the loop's tempo over the seconds of real oracle +
// rebase work, instead of watching a spinner snap to a verdict. The beats channel
// is buffered past the beat count so the cycle never blocks on a slow/gone client.
func (c *LiveCard) OnConnect(ctx *via.Ctx) error {
	cfg, log := readLiveState(c.Key)
	// A session with no primary anchor (flag-less / repo-only boot) has no catch cycle
	// to run — skip it so the working card (View) never sits behind a phantom
	// Oracle-running spinner.
	if log == nil || !cfg.hasAnchor() {
		return nil
	}
	sem := cycleSem(c.Key)
	type resolved struct{ verdict, land, questions string }
	beats := make(chan pipe.TraceEvent, 16)
	result := make(chan resolved, 1)
	go func() {
		// Acquire a cycle slot (when capped): connects beyond MaxConcurrent block
		// here until a running cycle frees a slot — queued, never dropped. The
		// release covers every exit path (cycle error included), so a slot can't leak.
		if sem != nil {
			sem <- struct{}{}
			defer func() { <-sem }()
		}
		res, err := resolveCycle(context.Background(), cfg.RepoDir, cfg.BaseRev, cfg.FixRev, cfg.TipRev,
			cfg.Anchor, cfg.TestCmd, cfg.SelfFlagged, cfg.WouldHaveShipped, beats)
		close(beats)
		if err != nil {
			result <- resolved{} // leave the card in-flight on a cycle error
			return
		}
		if res.Record != nil && log != nil {
			res.Record.Source = "connect" // provenance: the connect-cycle source, demuxed from a sent run's "wo:<id>"
			_ = log.Append(*res.Record)   // best-effort; a logging failure must not hang the card
		}
		// Cache this cycle's open questions (the fix oracle's surviving mutants) for
		// the /review surface — ephemeral diagnostic state, off the economy ledger.
		if e := lookupLiveEntry(c.Key); e != nil {
			e.setFindings(res.Findings)
			e.recordQuestionBlocks(res.Findings) // each surfaced question opens an attention-bandwidth interval
			e.setLand(string(res.Land))          // cache the integration verdict for the fleet board
		}
		result <- resolved{verdict: res.Verdict, land: string(res.Land), questions: strconv.Itoa(len(res.Findings))}
	}()
	var accrued []string
	lastSendSig := -1
	lastFill := "0:0"
	via.Stream(ctx, 100*time.Millisecond, func(ctx *via.Ctx, _ time.Time) {
		for { // drain every beat available this tick, flushing the growing row
			select {
			case ev, ok := <-beats:
				if !ok {
					beats = nil // closed: stop selecting on it (a nil channel never fires)
					break
				}
				accrued = append(accrued, ev.Kind)
				c.Beats.Write(ctx, strings.Join(accrued, ","))
				continue
			default:
			}
			break
		}
		// Poll the send tally so a BACKGROUND packet runner (drainQueuedPackets has
		// no request ctx, cannot write cells) still surfaces over SSE: when the
		// per-status counts change, write the Sends cell to re-render, so the Lead
		// watches the packet move queued→running→done live. Keyed on a cheap signature
		// so an unchanged tally writes nothing (no spurious frames).
		if log != nil {
			// ONE ledger projection per tick (RecentSends), same cost as the
			// SendStatusCounts poll it replaces — the per-status tally, the
			// Caught tally, and the open-thread count all derive from it without a
			// second fold.
			if views, err := log.RecentSends(0); err == nil {
				// The needs-you rail's open-thread count has no cell of
				// its own — findings can change off the connect-cycle path (a sent
				// packet's own findings, a /review answer resolving one) with nothing else
				// to notice while this session's "/" stream is open. Folding it into the
				// SAME cheap signature the send tally already uses re-renders the rail
				// live without standing up a whole new SSE channel/cell.
				threads := len(sessionOpenThreads(c.Key))
				// A packet's Caught bit can flip WITHOUT any status/question change (a
				// catch crediting a packet whose status already settled to "done") — the
				// hero/settled fold is Caught-sensitive, so the caught tally folds into
				// the SAME signature too, or that transition would sit invisible behind
				// an open SSE connection until reload.
				var queued, running, done, caught int
				for _, v := range views {
					switch v.Status {
					case "queued":
						queued++
					case "running":
						running++
					case "done":
						done++
					}
					if v.Caught {
						caught++
					}
				}
				// A new interrupt (a review question surfacing) is an off-ledger fact
				// independent of the send tally above — recordQuestionBlocks logs it
				// straight to the ledger with no accompanying status/question/caught
				// change, so without folding used in here the interrupt KPI could sit
				// stale behind this open SSE connection until reload.
				used, _ := weeklyInterrupts(c.Key)
				if sig := used*1_000_000_000_000_000 + caught*1_000_000_000_000 + threads*1_000_000_000 + queued*1_000_000 + running*1_000 + done; sig != lastSendSig {
					lastSendSig = sig
					c.Sends.Write(ctx, strconv.Itoa(sig))
				}
			}
		}
		// Poll the live-fill buffer too: the packet the background runner is currently
		// filling + its accruing cycle beats, so the Lead WATCHES the work happen, not
		// just the queued→running→done counts. Keyed on (id, beat-count) so an
		// unchanged buffer writes nothing.
		if e := lookupLiveEntry(c.Key); e != nil {
			id, fb := e.fillSnapshot()
			if sig := strconv.Itoa(id) + ":" + strconv.Itoa(len(fb)) + ":" + e.activitySnapshot(); sig != lastFill {
				lastFill = sig
				c.FillBeats.Write(ctx, sig)
			}
		}
		select {
		case r := <-result:
			c.Verdict.Write(ctx, r.verdict)
			c.Land.Write(ctx, r.land)
			c.Questions.Write(ctx, r.questions)
		default:
		}
	})
	return nil
}

// NewServer wires the live review server: it starts the shared economy fabric,
// binds the default session's ledger (which OWNS the fabric's lifecycle), stashes
// the cycle config, mounts the LiveCard, and returns the Via app (an
// http.Handler) plus the ledger handle for the caller to close (closing it tears
// the fabric down). Extra Via options are passed through; tests wrap the returned
// app with httptest.NewServer(app) to drive it over HTTP.
func NewServer(cfg LiveConfig, opts ...via.Option) (*via.App, *ledger.Log, error) {
	f, err := startLiveFabric(cfg.LedgerPath, cfg.ListenAddr, cfg.Grants)
	if err != nil {
		return nil, nil, err
	}
	liveFabric = f
	log := ledger.BindOwning(f, defaultSessionKey, LedgerInstance)
	// Register the default session when it has a repo — enough to be usable
	// (prompt-authoring; an anchor adds the catch-cycle but is derived from the repo, so
	// a real boot with an anchor always has one). A repo-less flag-less boot registers
	// no default — the board + settings still serve, and "/" is a calm landing (see
	// View). The owning log still binds the fabric so the returned handle's Close tears
	// the substrate down.
	if cfg.hasRepo() {
		setLiveState(cfg, log)
	}
	// The Anthropic key lives beside the ledger (one server, one key). Bind the store
	// and inject any saved key into the env before mounting, so a restart keeps the
	// harness runnable without a re-entry and the settings card reflects it.
	tokenStore = tokenstore.New(tokenConfigPath(cfg.LedgerPath))
	loadStoredTokenIntoEnv()
	app := via.New(opts...)
	// Attach the base stylesheet (the calm visual language) to every page's head
	// before mounting — boot-time, so it never races a render. It targets the
	// class hooks the card/board markup already emit; no markup changes here.
	app.AppendToHead(styleHead())
	via.Mount[LiveCard](app, "/")
	via.Mount[BoardCard](app, "/board")       // the cross-card fleet view (read-only projection of liveReg)
	via.Mount[ReviewCard](app, "/review")     // the Inspector: the gate's open "question:" threads
	via.Mount[SettingsCard](app, "/settings") // the setup surface: configure the Anthropic API key the harness runs with
	// The raw SSE bridge over the authoritative stream: a plain text/event-stream
	// endpoint a browser (or any cross-process consumer) tails, distinct from the
	// in-process Via reactivity above. ?key=<session> selects which session's
	// economy to stream (the default when absent). The key MUST be a registered
	// session: an unregistered or wildcard ('*'/'>') key is refused, so it can
	// neither inject a fleet-wide subject filter nor stream a phantom economy. The
	// method-qualified pattern keeps it a more-specific path under the same method
	// as Via's "GET /" mount, avoiding a ServeMux precedence conflict.
	app.HandleFunc("GET /stream", func(w http.ResponseWriter, r *http.Request) {
		key := r.URL.Query().Get("key")
		if key == "" {
			key = defaultSessionKey
		}
		// Only a registered session is served. Registration validates the key as a
		// subject token (AddSession / validateSessions), so a metacharacter or
		// wildcard key can never be in the registry — a registry miss refuses it.
		if _, ok := liveReg.Load(key); !ok {
			http.NotFound(w, r)
			return
		}
		bridge.Handler(f, key, LedgerInstance)(w, r)
	})
	// Claim submission is RETIRED from the HTTP surface: a peer
	// submits a claim ONLY through the authenticated NATS ingress
	// (fabric.StartListening + a Grant), publishing to its own grant-confined
	// claim subtree. The host's claim consumer drains it there. This removes the
	// unauthenticated HTTP edge so a claim can no longer be injected by anyone who
	// can reach the port; the per-message size bound is now NATS's max-payload, and
	// the cage verifier remains the fail-closed check that the revs resolve.
	// A cross-process peer uploads a git bundle of its commits here BEFORE
	// submitting a claim. The host validates + namespace-confines it OFFLINE
	// (ingest unbundles only into refs/producers/<key>/* of the session's repo — legacy wire name),
	// so a later claim's SHAs resolve against that repo WITHOUT the host ever
	// fetching a peer-controlled URL — no egress, no SSRF. The
	// peer id is the session key (the peer identity per one-session-per-
	// peer); a key that is not a safe ref segment is refused by ingest (400).
	// Mirrors POST /claim's session-key gate + body cap (this is the live server's
	// HTTP peer surface; if that ever moves to the NATS Grant path,
	// the bundle channel moves with it).
	const maxBundleBytes = 32 << 20 // 32 MiB — a commit bundle is small; a hard ceiling on abuse
	app.HandleFunc("POST /bundle", func(w http.ResponseWriter, r *http.Request) {
		key := r.URL.Query().Get("key")
		if key == "" {
			key = defaultSessionKey
		}
		// When peers are configured, the bundle blob (the one peer channel
		// that stays on HTTP — git bundles are ill-suited to NATS messaging) is
		// authenticated against the SAME grant table as the NATS claim ingress:
		// Basic credentials must match a grant for this session key (peer ==
		// session key). Checked BEFORE the registry lookup so an unauthenticated
		// prober learns nothing about which keys exist. With no grants configured
		// (in-process/single-user runs) the endpoint stays open, as before.
		if len(cfg.Grants) > 0 && !bundleAuthorized(cfg.Grants, key, r) {
			w.Header().Set("WWW-Authenticate", `Basic realm="packets-bundle"`)
			http.Error(w, "bundle: unauthorized", http.StatusUnauthorized)
			return
		}
		if _, ok := liveReg.Load(key); !ok {
			http.NotFound(w, r)
			return
		}
		cfg, _ := readLiveState(key)
		// A session with no configured repo must refuse, not pass "" to ingest: an
		// empty store makes git run in the server process cwd, so an upload would
		// silently land the peer's commits in refs/producers/<key>/* of whatever
		// repo the server was launched from. Reject before reading the body.
		if cfg.RepoDir == "" {
			http.Error(w, "bundle: session has no repository", http.StatusBadRequest)
			return
		}
		// Per-peer flood-defenses: throttle the upload RATE so a peer
		// can't flood the ingest path, then (after reading) bound the aggregate bytes
		// it RETAINS so it can't fill the host store. Both key off the authenticated
		// peer identity (== session key) the boundary now guarantees.
		guard := bundleGuardFor(key)
		if !guard.allowUpload() {
			http.Error(w, "bundle: rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxBundleBytes)
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "bundle: too large or unreadable", http.StatusBadRequest)
			return
		}
		// Reserve this upload's bytes against the peer's quota AND the global
		// ceiling BEFORE ingesting; an over-limit upload is refused without doing the
		// work. GC-by-resolved frees both when the peer's objects are
		// reclaimed. A per-peer overflow is 413 (this peer's fault); a global
		// overflow is 503 (the host is at capacity, not this peer's fault).
		if ok, global := guard.reserve(int64(len(body))); !ok {
			if global {
				http.Error(w, "bundle: host storage at capacity", http.StatusServiceUnavailable)
			} else {
				http.Error(w, "bundle: peer storage quota exceeded", http.StatusRequestEntityTooLarge)
			}
			return
		}
		// A bad peer id, an invalid bundle, or one past the cap is a client
		// error; keep the message generic — the typed reasons live in ingest, and
		// leaking git internals would aid a prober. A failed ingest releases the
		// reserved bytes so a rejected upload never permanently consumes quota.
		if err := ingest.IngestPeerObjects(r.Context(), cfg.RepoDir, key, body, maxBundleBytes); err != nil {
			guard.release(int64(len(body)))
			http.Error(w, "bundle: rejected", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusAccepted)
	})
	// The cross-session fleet board, streamed off the same authoritative stream:
	// one ordered SSE frame of per-session rows per committed event, across every
	// session. Additive to the in-process Via BoardCard at "/board".
	app.Handle("GET /fleet", bridge.FleetHandler(f))
	return app, log, nil
}
