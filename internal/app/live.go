package app

import (
	"context"
	"crypto/subtle"
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
	"github.com/joaomdsg/packets/internal/fabric"
	"github.com/joaomdsg/packets/internal/harness"
	"github.com/joaomdsg/packets/internal/ingest"
	"github.com/joaomdsg/packets/internal/ledger"
	"github.com/joaomdsg/packets/internal/mutation"
	"github.com/joaomdsg/packets/internal/pipe"
	"github.com/joaomdsg/packets/internal/reanchor"
	"github.com/joaomdsg/packets/internal/review"
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
	// DispatchBacklog is the ordered supply of DISTINCT work a card's Spends draw
	// down — the rev/anchor triple each funded order runs. A Spend consumes the next
	// not-yet-funded target head-first; an empty or fully-drawn-down backlog makes a
	// Spend a silent no-op (the honest scarcity signal — no distinct work to buy).
	DispatchBacklog []ledger.Target
	// UseContainer, when true, runs this session's LIVE orders (Target.Prompt set) in
	// the hardened agent container (harness.RunContainer) instead of the host
	// subprocess (harness.RunProcess). The firewall is unchanged — both produce a
	// revision the host settles; only WHERE the agent runs differs.
	UseContainer bool
	// ListenAddr, when non-empty, binds the shared fabric to an AUTHENTICATED TCP
	// NATS listener (host:port; port 0 picks a free port) so cross-process PRODUCERS
	// submit claims as authenticated clients, each confined by a Grant to its own
	// session's claim subtree. Empty keeps the fabric in-process-only (the default —
	// tests and single-process runs need no socket and no auth).
	ListenAddr string
	// Grants authorizes the cross-process producers allowed on ListenAddr. Each
	// grant's credentials may publish ONLY to its session's claim subtree and can
	// never mint — the in-process host stays the single minter. Ignored when
	// ListenAddr is empty. Build with NewProducerGrant.
	Grants []fabric.ProducerGrant
}

// bundleAuthorized reports whether the request carries HTTP Basic credentials
// matching a grant for session key (producer == session key). The password is
// compared in constant time so a prober cannot time-recover it; the user/session
// equality checks are not secret and need no such guard.
func bundleAuthorized(grants []fabric.ProducerGrant, key string, r *http.Request) bool {
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

// NewProducerGrant builds a producer authorization for the live server: the
// credentials may publish claims ONLY to sessionKey's claim subtree, bound to
// the one instance every economy uses (ledgerInstance), and may never mint. It
// is the sanctioned constructor so callers (e.g. cmd/packets) need not know the
// internal instance token.
func NewProducerGrant(sessionKey, user, pass string) fabric.ProducerGrant {
	return fabric.ProducerGrant{User: user, Pass: pass, Session: sessionKey, Instance: ledgerInstance}
}

// resolveCycle is the seam OnConnect runs the catch cycle through. It defaults to
// the real ResolveStreaming; tests swap it to drive the admission cap
// deterministically without spinning up real oracle work.
var resolveCycle = ResolveStreaming

// runHarness is the seam a LIVE work order runs its agent through. It defaults to
// the real harness.RunProcess (spawns claude, reduces its stream-json into
// settled revisions); tests swap it for a scripted stub so the live-fill routing
// is exercised without a claude binary or API key.
var runHarness = harness.RunProcess

// runHarnessContainer is the seam a UseContainer live order runs its agent through
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
	// runMu serializes the per-key order runner so two concurrent Spends can't both
	// drain (and double-run) the same queued order. One drainer per session at a time.
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
	// killed the mutant) this session. R63 settled that a killing answer makes the
	// question vanish; since the reviewer's test isn't committed, a later connect
	// cycle would re-find the survivor, so openFindings filters resolved lines out —
	// the answer sticks for the session. Ephemeral, off the economy ledger (a
	// diagnostic, like findings); guarded by findingsMu.
	resolved map[string]bool
	// land is the latest connect cycle's integration verdict (clean/conflict/
	// checks_red), cached so the fleet board can show which sessions are blocked from
	// merging — ephemeral, recomputed each connect, off the economy ledger. Guarded
	// by findingsMu (written together with findings in OnConnect).
	land string
	// orderFindings holds a FILLED work-order's review questions (the cycle's
	// surviving mutants) keyed by order ID — captured when runOneOrder fills the
	// order, so a funded order's test-debt is reviewable (the dispatch→review tie).
	// Ephemeral and OFF the economy ledger, like findings (the order's CATCH mints;
	// its questions are diagnostic). Guarded by findingsMu.
	orderFindings map[int][]mutation.Finding
	// answering is true while an answer re-run is in flight for this session. It
	// serializes answer re-runs (one at a time): a re-run spawns a git worktree +
	// oracle run, so two concurrent re-runs (a double-clicked submit) would race the
	// shared repo's worktree operations. Guarded by findingsMu.
	answering bool
	// fillMu + fillingOrder/fillBeats: the live-fill buffer (see startFill). Guarded
	// separately from findingsMu since beats accrue rapidly during a fill.
	fillMu       sync.Mutex
	fillingOrder int
	fillBeats    []string
	// activityBeat is the live agent's LATEST activity line (e.g. "editing auth.go")
	// while a live order fills — a single updating beat, not a log. Bracketed to the
	// fill lifecycle (reset in startFill, cleared in endFill) and guarded by fillMu.
	activityBeat string
	// activityLog is the accruing TRANSCRIPT of the agent's beats this fill, in stream
	// order — the run made legible (the card scrolls it) rather than only the latest
	// move. Capped at maxActivityLog (oldest dropped) so a long run can't grow it
	// without bound. Bracketed to the fill lifecycle like activityBeat; guarded by fillMu.
	activityLog []string
}

// maxActivityLog bounds the in-flight transcript so a long agent run can't grow the
// per-session buffer without limit; the oldest beats scroll off once it is reached.
const maxActivityLog = 300

// fillMu guards the live-fill buffer: the work-order currently being filled by the
// background runner and the cycle beats accrued so far, so the card can show it
// filling LIVE ("watch it fill"). The runner has no request ctx to write the card's
// cells, so it writes this buffer and the card's Stream polls it (like the dispatch
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

// activityTranscript returns a copy of the accruing beat transcript for this fill
// (nil when none) — the run history the card scrolls.
func (e *liveEntry) activityTranscript() []string {
	e.fillMu.Lock()
	defer e.fillMu.Unlock()
	return append([]string(nil), e.activityLog...)
}

// addFillBeat appends one cycle beat for the filling order (the live tempo).
func (e *liveEntry) addFillBeat(kind string) {
	e.fillMu.Lock()
	e.fillBeats = append(e.fillBeats, kind)
	e.fillMu.Unlock()
}

// endFill clears the live-fill buffer when the order is done — the filling row
// vanishes and the order's resolved outcome takes over.
func (e *liveEntry) endFill() {
	e.fillMu.Lock()
	e.fillingOrder, e.fillBeats, e.activityBeat, e.activityLog = 0, nil, "", nil
	e.fillMu.Unlock()
}

// fillSnapshot returns the filling order's id (0 if none) and a copy of its beats.
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
// survivor (R63's "the question vanishes").
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

// setOrderFindings caches a filled work-order's review questions (off-ledger, like
// findings) so the order's test-debt is reviewable. Empty findings clear the entry.
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

// orderQuestionCount returns how many open review questions a filled order left.
func (e *liveEntry) orderQuestionCount(id int) int {
	e.findingsMu.Lock()
	defer e.findingsMu.Unlock()
	return len(e.orderFindings[id])
}

// orderFindingsFor returns a filled order's cached review questions (nil if none).
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
// economies — a mint or spend on one key never touches another (the R18
// farm-denial verdict, enforced per session: the faucet is the sole credit
// source and a balance is non-transferable across keys).
func registerSession(key string, cfg LiveConfig, log *ledger.Log) {
	var sem chan struct{}
	if cfg.MaxConcurrent > 0 {
		sem = make(chan struct{}, cfg.MaxConcurrent)
	}
	e := &liveEntry{cfg: cfg, log: log, sem: sem, seq: int(atomic.AddInt64(&regSeq, 1))}
	liveReg.Store(key, e)
	// If claim consumers are already running, a session registered now (e.g. an R53
	// runtime-created session) gets its consumer immediately, not only those present
	// at boot.
	consumerSpawner.onRegister(key, e)
}

func setLiveState(cfg LiveConfig, log *ledger.Log) {
	registerSession(defaultSessionKey, cfg, log)
}

// claimConsumerSpawner gives each session EXACTLY ONE durable claim consumer — for
// the sessions present when consumers start AND for any session registered later
// (R53 runtime-created sessions), so the create flow is not a dead end for the
// producer path. Birth is guarded by `started` so a session is never double-
// consumed. Once active, registerSession spawns a consumer for each new session
// using the latest StartClaimConsumers parameters.
type claimConsumerSpawner struct {
	mu          sync.Mutex
	active      bool
	ctx         context.Context
	verifierFor func(LiveConfig) ledger.Verifier
	ackWait     time.Duration
	adm         *ledger.Admission
	started     map[string]bool
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
	// Per-producer bundle guards are server-lifetime in production but must not
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
	consumerSpawner.active = false
	consumerSpawner.ctx = nil
	consumerSpawner.verifierFor = nil
	consumerSpawner.ackWait = 0
	consumerSpawner.adm = nil
	consumerSpawner.started = nil
}

// spawnLocked starts a consumer for key/e unless one is already running. mu held.
// The spawner fields are copied into locals UNDER the lock and the goroutine closes
// over those locals — never the shared struct fields — so a later StartClaimConsumers
// call writing s.ctx/s.verifierFor/etc. can't race the running goroutine's reads.
func (s *claimConsumerSpawner) spawnLocked(key string, e *liveEntry) {
	if s.started[key] {
		return
	}
	s.started[key] = true
	ctx, verifierFor, ackWait, adm := s.ctx, s.verifierFor, s.ackWait, s.adm
	go func() { _ = e.log.ConsumeClaims(ctx, verifierFor(e.cfg), ackWait, adm) }()
}

// onRegister is called after a session is stored in liveReg. If consumers are
// active, the new session gets one immediately — the R53 runtime-create path.
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
	consumerSpawner.started = map[string]bool{} // this call owns a fresh consumer set
	liveReg.Range(func(k, v any) bool {
		consumerSpawner.spawnLocked(k.(string), v.(*liveEntry))
		return true
	})
}

// ledgerInstance is the subject instance token every session's economy binds to.
// There is one economy per session, so the session key alone demuxes them; the
// instance is a fixed token completing the canonical subject.
const ledgerInstance = "ledger"

// liveFabric is the one embedded JetStream the server's sessions share — the
// single authoritative economy substrate. NewServer
// starts it and gives the primary Log ownership of its lifecycle; AddSession
// binds further sessions to it under their own session token, so each session is
// an ISOLATED economy on the one stream. Set once per server; the live tests
// drive NewServer serially (they share this and liveReg), so it is not guarded.
var liveFabric *fabric.Fabric

// startLiveFabric stands up the shared economy fabric, rooting its durable store
// beside the configured ledger path (a dedicated dir per server, so two servers
// in one process never share a store). An empty path falls back to a temp store.
func startLiveFabric(ledgerPath, listenAddr string, grants []fabric.ProducerGrant) (*fabric.Fabric, error) {
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
	// A configured listen address binds an authenticated socket (the host stays
	// in-process via NoAuthUser; producers authenticate and are grant-confined).
	// Absent it, the fabric is in-process-only — no socket, no auth surface.
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
	log := ledger.Bind(liveFabric, key, ledgerInstance)
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
	// to dispatch the CHOSEN target instead of the FIFO head.
	FundTarget via.SignalStr `via:"fundtarget"`
	// OrderPrompt carries the free-form task the Lead authored in the compose control,
	// read by PlaceOrder to fund a prompt-carrying live order (vs drawing a pre-baked
	// backlog target). Per-tab signal, not authoritative session state.
	OrderPrompt via.SignalStr `via:"orderprompt"`
	// Balance is the spend broadcast trigger: the balance ROW value is re-read
	// from the ledger in View (the source of truth), but the ledger is not
	// reactive — so Spend writes the new balance here to make the live SSE stream
	// re-render (a cell Write fans out a re-render; an action's auto-render only
	// returns in the action's own response).
	Balance via.StateTabStr
	// Dispatch is the same broadcast trigger for the dispatched-work tally: the
	// count is re-read from the ledger in View, but a Spend writes the new count
	// here so the dispatch row rises over the live SSE stream in the SAME render as
	// the balance drains. It carries no authoritative value — View is the source.
	Dispatch via.StateTabStr
	// FillBeats is a re-render trigger written by the Stream when the live-fill buffer
	// (a currently-filling order's accruing beats) changes, so the card shows the
	// order filling live. View reads the buffer; this cell only nudges the re-render.
	FillBeats via.StateTabStr
}

// View renders the card's rows via the shared surface rendering: the retrospective
// confirmed-catch STOCK (re-derived read-only from the ledger on every render — the
// economy finally SHOWN, not just logged), the streamed beat row (the felt tempo),
// the oracle verdict row, and the integration (Land) row. One row never speaks for
// another. The stock is read-only: a ledger read failure degrades to an empty
// stock, never breaks the card.
func (c *LiveCard) View(ctx *via.CtxR) h.H {
	cfg, log := readLiveState(c.Key)
	var stock ledger.Stock
	balance := 0
	bandwidth := 0
	var dispatch ledger.DispatchCounts
	var dispatches []ledger.DispatchView
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
		if c, err := log.DispatchStatusCounts(); err == nil {
			dispatch = c
		}
		// This session's recent funded work-orders + their caught/missed outcome —
		// the round-trip the Lead watches after a Spend, on the same card they act on.
		if ds, err := log.RecentDispatches(5); err == nil {
			dispatches = ds
			// Enrich each with its open-question count (the order's reviewable
			// test-debt) from the per-order findings cache — off-ledger diagnostic,
			// so it's filled here, not projected.
			if e := lookupLiveEntry(c.Key); e != nil {
				for i := range dispatches {
					dispatches[i].Questions = e.orderQuestionCount(dispatches[i].ID)
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
	// LIVE region: this card re-renders over SSE on every catch/balance/dispatch
	// change, so role="main" + aria-live="polite" lets assistive tech announce those
	// changes without the user hunting for them. The nav is a sibling landmark (added
	// in the final wrap), never nested inside main.
	parts := []h.H{
		h.Role("main"),
		h.Attr("aria-live", "polite"),
		h.Attr("aria-label", "session economy"),
	}
	// A brand-new session gets a calm onboarding affordance ahead of the (all-zero)
	// economy rows, so a first-run Lead sees the next action, not a dead screen.
	if hint := onboardingHint(stock); hint != nil {
		parts = append(parts, hint)
	}
	parts = append(parts, surface.RenderStock(stock), surface.RenderBalance(balance),
		surface.RenderBandwidth(bandwidth))
	// The Spend action — turning a confirmed catch into a funded work-order — is the
	// Lead's core economic move. Render its trigger right under the balance it spends,
	// but ONLY when there is balance to spend: offering a Spend control with nothing
	// to spend is dishonest (the click would be a silent no-op).
	if balance > 0 {
		parts = append(parts, h.Button(
			on.Click(c.Spend),
			h.Class("spend-action"),
			h.Text(spendButtonLabel(cfg, log)),
		))
	}
	// AUTHOR a live order: when there is balance to fund one, render the compose
	// control so the Lead can type a task and place it (the runtime counterpart of
	// the -live CLI flag). Below it, a calm note when no API key is configured — the
	// order would fail without one — linking to the setup surface rather than letting
	// the Lead place an order that can't run.
	if balance > 0 {
		parts = append(parts, renderCompose(c))
	}
	// The prep bench: the fundable work on deck, so the Lead sees (and, in a later
	// slice, curates) what a Spend funds rather than a blind auto-pick. Omitted when
	// there is no fundable work; guarded on log (fundableBacklog reads it).
	if log != nil {
		if bench := renderBench(c, fundableBacklog(cfg, log)); bench != nil {
			parts = append(parts, bench)
		}
	}
	parts = append(parts, surface.RenderDispatch(dispatch))
	// WATCH IT FILL: when the background runner is mid-fill on an order, show it live
	// — the order id + the cycle beats accruing as the oracle works (re-rendered each
	// Stream tick via the FillBeats poll). Omitted when nothing is filling.
	if e := lookupLiveEntry(c.Key); e != nil {
		if id, fb := e.fillSnapshot(); id > 0 {
			parts = append(parts, h.Div(
				h.Class("order-filling"),
				h.Data("state", "beats"),
				h.Text("filling WO#"+strconv.Itoa(id)+" — "+strings.Join(fb, " → ")),
			))
			// The live agent's LATEST move (a single updating line) while it works —
			// distinct from the oracle's cycle beats above. Absent on dead-air (no beat
			// yet) so silence stays honest, no spinner.
			if act := e.activitySnapshot(); act != "" {
				parts = append(parts, h.Div(
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
				lines := []h.H{h.Class("order-transcript"), h.Data("state", "transcript"),
					h.Attr("aria-label", "agent transcript")}
				for _, line := range tr {
					lines = append(lines, h.Div(h.Class("order-transcript__line"), h.Text(line)))
				}
				parts = append(parts, h.Div(lines...))
			}
		}
	}
	// Below the aggregate counts, the per-order round-trip: each recent work-order
	// with its caught/missed outcome, so the Lead watches the order they funded
	// resolve in place (omitted when there are none — same helper the board uses).
	if d := renderDispatches(navKey, dispatches); d != nil {
		parts = append(parts, d)
	}
	parts = append(parts,
		surface.RenderBeats(c.Beats.Read(ctx)),
		surface.RenderVerdict(c.Verdict.Read(ctx)),
	)
	// A gated, calm badge: when the oracle left surviving mutants, the verdict's
	// green hides honest test gaps — show the open-question count (the full anchored
	// threads live on /review). Omitted when there are none.
	if b := reviewQuestionsBadge(c.Questions.Read(ctx), navKey); b != nil {
		parts = append(parts, b)
	}
	parts = append(parts, surface.RenderLand(pipe.LandState(c.Land.Read(ctx))))
	// nav landmark first, then the main economy region — distinct sibling landmarks.
	return h.Div(navHeader(navKey), h.Div(parts...))
}

// Spend funds one unit of dispatched work against the balance — the Lead's first
// ACTION on the stock, and the moment a catch finally BUYS something. It debits
// one catch AND fuels exactly one queued work-order in a single atomic ledger
// fact (AppendDispatch). An over-budget spend (balance already 0) is refused by
// the ledger and the action is a silent no-op (no broadcast). On success it
// writes BOTH the drained balance and the risen dispatch count to their trigger
// cells, whose Writes fan out a single re-render to the live SSE stream so the
// balance drains and the dispatch row rises together — the spend is visibly
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
	if err := log.AppendDispatch("dispatch", tgt, ownTargetOf(cfg)); err != nil {
		return // over-budget / nothing to spend / own work: a no-op, never an error to the Lead
	}
	if b, err := log.Balance(); err == nil {
		c.Balance.Write(ctx, strconv.Itoa(b)) // announce the drain
	}
	if d, err := log.PendingDispatches(); err == nil {
		c.Dispatch.Write(ctx, strconv.Itoa(d)) // announce the funded work-order so the dispatch row rises in the same render
	}
	go drainQueuedOrders(c.Key) // the order RUNS in the background — spend-to-earn
}

// FundChosen is the prep bench's payoff: it funds the CHOSEN bench target (set by
// that item's on.SetSignal into FundTarget) instead of the FIFO head, turning
// dispatch from a blind auto-pick into the Lead's management-sim decision. The
// chosen target is VALIDATED to be in the fundable set (chosenFundable), so a click
// can never fund the card's own cycle, an already-consumed target, or an arbitrary
// one — the distinct-work / two-scores rules hold. Otherwise it mirrors Spend: one
// atomic AppendDispatch debit, then announce the drain + risen dispatch over SSE and
// run the order. An off-bench key or over-budget balance is a silent no-op.
func (c *LiveCard) FundChosen(ctx *via.Ctx) {
	cfg, log := readLiveState(c.Key)
	if log == nil {
		return
	}
	tgt, ok := chosenFundable(cfg, log, strings.TrimSpace(c.FundTarget.Read(ctx)))
	if !ok {
		return // not on the bench (unknown / consumed / own cycle): a no-op
	}
	if err := log.AppendDispatch("dispatch", tgt, ownTargetOf(cfg)); err != nil {
		return // over-budget / nothing to spend: a no-op, never an error to the Lead
	}
	if b, err := log.Balance(); err == nil {
		c.Balance.Write(ctx, strconv.Itoa(b))
	}
	if d, err := log.PendingDispatches(); err == nil {
		c.Dispatch.Write(ctx, strconv.Itoa(d))
	}
	go drainQueuedOrders(c.Key)
}

// PlaceOrder authors a LIVE order from the card: it funds a prompt-carrying target
// (the Lead's free-form task) against the balance and dispatches it, so the live
// harness runs the authored work — the UI counterpart of the -live CLI flag, but
// composed at runtime instead of baked at boot. The base is the repo's CURRENT
// HEAD, so the agent works the live tree. An empty prompt, an unconfigured repo, or
// an over-budget balance is a silent no-op (never a funded order with no task, no
// tree, or no catch to spend). On success it mirrors Spend: announce the drained
// balance + risen dispatch over SSE, then run the order in the background.
func (c *LiveCard) PlaceOrder(ctx *via.Ctx) {
	cfg, log := readLiveState(c.Key)
	if log == nil {
		return
	}
	prompt := strings.TrimSpace(c.OrderPrompt.Read(ctx))
	if prompt == "" {
		return // an empty prompt is not an order
	}
	head, ok := repoHead(cfg.RepoDir)
	if !ok {
		return // no resolvable tree to run the agent against — never dispatch a treeless live order
	}
	tgt := ledger.Target{BaseRev: head, Prompt: prompt}
	if err := log.AppendDispatch("liveorder", tgt, ownTargetOf(cfg)); err != nil {
		return // over-budget / nothing to spend: a no-op, never an error to the Lead
	}
	if b, err := log.Balance(); err == nil {
		c.Balance.Write(ctx, strconv.Itoa(b))
	}
	if d, err := log.PendingDispatches(); err == nil {
		c.Dispatch.Write(ctx, strconv.Itoa(d))
	}
	go drainQueuedOrders(c.Key)
}

// repoHead resolves repoDir's current HEAD, the base a UI-authored live order runs
// from. An empty dir or an unresolvable HEAD (no repo / no commit) reports false so
// PlaceOrder refuses rather than dispatch a treeless order.
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

// maxOrderAttempts bounds how many times the runner will pick a single queued
// order before giving up on it. A status write that fails permanently (e.g. a
// closed ledger handle) would otherwise leave an order forever queued and spin
// the suite-exec loop without end; the cap turns that into a bounded, abandoned
// order instead of an unbounded #15-multiplier burn.
const maxOrderAttempts = 3

// drainQueuedOrders runs every queued work-order for a session to completion — the
// second in-process producer. It serializes per session (runMu) so two concurrent
// Spends never double-run an order. Each order: mark running, run its DISTINCT
// target through the catch cycle under the admission sem (bounding the suite-exec
// cost), route any Catch through the idempotent Append stamped with the order's
// producer (a re-run that reproduces a seen identity mints nothing — an honest
// loss), then mark done. The mint is the only thing logged; intermediate beats
// stay off-ledger. An order whose status can never advance is retried at most
// maxOrderAttempts times then GIVEN UP (a best-effort terminal "failed" line, so
// it leaves the queued set when the log is writable), guaranteeing the drain
// always returns.
func drainQueuedOrders(key string) {
	e := lookupLiveEntry(key)
	if e == nil || e.log == nil {
		return
	}
	e.runMu.Lock()
	defer e.runMu.Unlock()
	attempts := map[int]int{}
	givenUp := map[int]bool{}
	for {
		queued, err := e.log.QueuedWorkOrders()
		if err != nil {
			return
		}
		var order *ledger.WorkOrderRecord
		for i := range queued {
			if !givenUp[queued[i].ID] {
				order = &queued[i]
				break
			}
		}
		if order == nil {
			return // nothing left that hasn't been given up
		}
		attempts[order.ID]++
		if attempts[order.ID] > maxOrderAttempts {
			givenUp[order.ID] = true
			_ = e.log.AppendStatus(order.ID, "failed") // best-effort terminal line; if this too fails, givenUp still bounds the loop
			continue
		}
		if order.Target.Prompt != "" {
			runLiveOrder(e, *order)
		} else {
			runOneOrder(e, *order)
		}
	}
}

// runLiveOrder fills a LIVE work order: a real Claude Code harness runs the
// order's task prompt and PRODUCES the fix revision in the repo (vs the
// pre-funded base→fix diff runOneOrder replays). It mints NO catch — the
// oracle/catch step on the produced revision is a later slice; this settles only
// the agent's git revision, keeping the catch economy untouched (the firewall).
// A terminal status is always reached ("done" on success, "failed" on a harness
// error) so the order never lingers mid-flight: once it leaves "queued" the
// drain's attempts cap no longer sees it, so the terminal write must happen here.
func runLiveOrder(e *liveEntry, order ledger.WorkOrderRecord) {
	if err := e.log.AppendStatus(order.ID, "running"); err != nil {
		return // could not advance status — the order stays queued; the drain retries under the attempts cap
	}
	if e.sem != nil {
		e.sem <- struct{}{}
		defer func() { <-e.sem }()
	}
	e.startFill(order.ID)
	defer e.endFill()
	// Bound the agent run so a runaway harness can't burn the budget without limit
	// (the cost-gate — the only token cap a live order has; council R69/R70).
	hctx, cancel := context.WithTimeout(context.Background(), liveHarnessTimeout)
	runner := runHarness // host subprocess by default; the container when the session opts in
	if e.cfg.UseContainer {
		runner = runHarnessContainer
	}
	turns, err := runner(hctx, e.cfg.RepoDir, order.Target.Prompt, func(evs []translate.UIEvent) {
		if len(evs) > 0 {
			e.addActivityBeat(formatActivity(evs[len(evs)-1])) // the latest event = the agent's current move
		}
	})
	cancel()
	if err != nil {
		_ = e.log.AppendStatus(order.ID, "failed") // the live run failed — terminal, not a completed fill
		return
	}
	// Run the catch cycle on the agent-PRODUCED revision, against the order's
	// PRE-SPECIFIED anchor (Target.Path/Line) — never an anchor derived from the
	// agent's own diff, which would let it farm confirmed-catches (council R70).
	if liveHead, ok := lastMintedSHA(turns); ok {
		beats := make(chan pipe.TraceEvent, 64)
		go func() {
			for ev := range beats {
				e.addFillBeat(ev.Kind)
			}
		}()
		res, cerr := resolveCycle(context.Background(), e.cfg.RepoDir,
			order.Target.BaseRev, liveHead, liveHead,
			anchorFromTarget(order.Target), e.cfg.TestCmd, false, false, beats)
		close(beats)
		settleCatch(e, order.ID, res, cerr)
	}
	_ = e.log.AppendStatus(order.ID, "done")
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

func runOneOrder(e *liveEntry, order ledger.WorkOrderRecord) {
	if err := e.log.AppendStatus(order.ID, "running"); err != nil {
		return // could not advance the order's status — don't run; the drain loop retries under the attempts cap
	}
	if e.sem != nil {
		e.sem <- struct{}{}
		defer func() { <-e.sem }()
	}
	// Accrue the cycle's beats into the live-fill buffer so the card can show this
	// order filling LIVE ("watch it fill"). The runner has no request ctx to write
	// the card's cells; it writes the buffer and the card's Stream polls it.
	e.startFill(order.ID)
	beats := make(chan pipe.TraceEvent, 64)
	go func() {
		for ev := range beats {
			e.addFillBeat(ev.Kind)
		}
	}()
	res, err := resolveCycle(context.Background(), e.cfg.RepoDir,
		order.Target.BaseRev, order.Target.FixRev, order.Target.TipRev,
		anchorFromTarget(order.Target), e.cfg.TestCmd, false, false, beats)
	close(beats) // the cycle only SENDS on beats; the caller owns the close, so the accrue goroutine exits (mirrors OnConnect)
	settleCatch(e, order.ID, res, err)
	_ = e.log.AppendStatus(order.ID, "done")
	e.endFill() // the order is done — clear the live filling row; its outcome takes over
}

// settleCatch persists a catch cycle's result for an order: the minted catch (the
// only economy write — attributed to wo:<id>, deduped on a re-run of a seen
// identity), the oracle's verdict (diagnostic — the WHY behind a catch or miss),
// and the surviving-mutant findings (diagnostic — the dispatch→review tie). The
// verdict and findings are OFF the two-scores economy; only res.Record mints. A
// cycle error settles nothing.
func settleCatch(e *liveEntry, orderID int, res Resolution, err error) {
	if err != nil {
		return
	}
	if res.Record != nil {
		res.Record.Producer = "wo:" + strconv.Itoa(orderID)
		_ = e.log.Append(*res.Record)
	}
	_ = e.log.AppendWorkOrderVerdict(orderID, res.Verdict)
	e.setOrderFindings(orderID, res.Findings)
}

// lastMintedSHA returns the SHA of the last turn that minted a revision — the
// live order's "fix revision" — or ok=false when the agent committed nothing
// (so the caller skips the catch cycle: there is no revision to check).
func lastMintedSHA(turns []harness.Turn) (string, bool) {
	for i := len(turns) - 1; i >= 0; i-- {
		if turns[i].Outcome.Minted {
			return turns[i].Outcome.SHA, true
		}
	}
	return "", false
}

// anchorFromTarget reconstructs the re-anchor anchor a funded order runs against
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
			res.Record.Producer = "connect" // provenance: the connect-cycle producer, demuxed from a dispatched run's "wo:<id>"
			_ = log.Append(*res.Record)     // best-effort; a logging failure must not hang the card
		}
		// Cache this cycle's open questions (the fix oracle's surviving mutants) for
		// the /review surface — ephemeral diagnostic state, off the economy ledger.
		if e := lookupLiveEntry(c.Key); e != nil {
			e.setFindings(res.Findings)
			e.setLand(string(res.Land)) // cache the integration verdict for the fleet board
		}
		result <- resolved{verdict: res.Verdict, land: string(res.Land), questions: strconv.Itoa(len(res.Findings))}
	}()
	var accrued []string
	lastDispatch := -1
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
		// Poll the dispatch tally so a BACKGROUND order runner (drainQueuedOrders has
		// no request ctx, cannot write cells) still surfaces over SSE: when the
		// per-status counts change, write the Dispatch cell to re-render, so the Lead
		// watches the order move queued→running→done live. Keyed on a cheap signature
		// so an unchanged tally writes nothing (no spurious frames).
		if log != nil {
			if cnt, err := log.DispatchStatusCounts(); err == nil {
				if sig := cnt.Queued*1_000_000 + cnt.Running*1_000 + cnt.Done; sig != lastDispatch {
					lastDispatch = sig
					c.Dispatch.Write(ctx, strconv.Itoa(sig))
				}
			}
		}
		// Poll the live-fill buffer too: the order the background runner is currently
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
// the fabric down). Extra Via options (e.g. via.WithTestServer) are passed through.
func NewServer(cfg LiveConfig, opts ...via.Option) (*via.App, *ledger.Log, error) {
	f, err := startLiveFabric(cfg.LedgerPath, cfg.ListenAddr, cfg.Grants)
	if err != nil {
		return nil, nil, err
	}
	liveFabric = f
	log := ledger.BindOwning(f, defaultSessionKey, ledgerInstance)
	setLiveState(cfg, log)
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
	via.Mount[ReviewCard](app, "/review")     // the per-session review surface: the oracle's open "question:" threads
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
		bridge.Handler(f, key, ledgerInstance)(w, r)
	})
	// Claim submission is RETIRED from the HTTP surface (council R82): a producer
	// submits a claim ONLY through the authenticated NATS ingress
	// (fabric.StartListening + a ProducerGrant), publishing to its own grant-confined
	// claim subtree. The host's claim consumer drains it there. This removes the
	// unauthenticated HTTP edge so a claim can no longer be injected by anyone who
	// can reach the port; the per-message size bound is now NATS's max-payload, and
	// the cage verifier remains the fail-closed check that the revs resolve.
	// A cross-process producer uploads a git bundle of its commits here BEFORE
	// submitting a claim. The host validates + namespace-confines it OFFLINE
	// (ingest unbundles only into refs/producers/<key>/* of the session's repo),
	// so a later claim's SHAs resolve against that repo WITHOUT the host ever
	// fetching a producer-controlled URL — no egress, no SSRF (council R38). The
	// producer id is the session key (the producer identity per one-session-per-
	// producer); a key that is not a safe ref segment is refused by ingest (400).
	// Mirrors POST /claim's session-key gate + body cap (this is the live server's
	// HTTP producer surface; if that ever moves to the NATS ProducerGrant path,
	// the bundle channel moves with it).
	const maxBundleBytes = 32 << 20 // 32 MiB — a commit bundle is small; a hard ceiling on abuse
	app.HandleFunc("POST /bundle", func(w http.ResponseWriter, r *http.Request) {
		key := r.URL.Query().Get("key")
		if key == "" {
			key = defaultSessionKey
		}
		// When producers are configured, the bundle blob (the one producer channel
		// that stays on HTTP — git bundles are ill-suited to NATS messaging) is
		// authenticated against the SAME grant table as the NATS claim ingress:
		// Basic credentials must match a grant for this session key (producer ==
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
		// silently land the producer's commits in refs/producers/<key>/* of whatever
		// repo the server was launched from. Reject before reading the body.
		if cfg.RepoDir == "" {
			http.Error(w, "bundle: session has no repository", http.StatusBadRequest)
			return
		}
		// Per-producer flood-defenses (R85): throttle the upload RATE so a producer
		// can't flood the ingest path, then (after reading) bound the aggregate bytes
		// it RETAINS so it can't fill the host store. Both key off the authenticated
		// producer identity (== session key) the boundary now guarantees.
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
		// Reserve this upload's bytes against the producer's quota AND the global
		// ceiling BEFORE ingesting; an over-limit upload is refused without doing the
		// work. GC-by-resolved (R84) frees both when the producer's objects are
		// reclaimed. A per-producer overflow is 413 (this producer's fault); a global
		// overflow is 503 (the host is at capacity, not this producer's fault).
		if ok, global := guard.reserve(int64(len(body))); !ok {
			if global {
				http.Error(w, "bundle: host storage at capacity", http.StatusServiceUnavailable)
			} else {
				http.Error(w, "bundle: producer storage quota exceeded", http.StatusRequestEntityTooLarge)
			}
			return
		}
		// A bad producer id, an invalid bundle, or one past the cap is a client
		// error; keep the message generic — the typed reasons live in ingest, and
		// leaking git internals would aid a prober. A failed ingest releases the
		// reserved bytes so a rejected upload never permanently consumes quota.
		if err := ingest.IngestProducerObjects(r.Context(), cfg.RepoDir, key, body, maxBundleBytes); err != nil {
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
