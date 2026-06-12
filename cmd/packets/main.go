// Command packets serves the single-user review wire: it runs one
// confirmed-catch cycle over two revisions and streams the verdict to a live
// review card over SSE, so a human opens a browser and watches one verdict go
// in-flight → resolved, with any catch appended to the ledger.
//
//	packets -repo . -base <weakSHA> -fix <fixSHA> -file adult.go -line 4
//	open http://localhost:3000
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"time"

	"github.com/joaomdsg/packets/internal/app"
	"github.com/joaomdsg/packets/internal/fabric"
	"github.com/joaomdsg/packets/internal/ledger"
	"github.com/joaomdsg/packets/internal/pipe"
	"github.com/joaomdsg/packets/internal/reanchor"
	"github.com/joaomdsg/packets/internal/sandbox"
)

// verifyTestCmd is the FIXED suite command the host runs the oracle with. It is
// host-controlled (never supplied by an agent), so a producer cannot choose what
// executes on its behalf.
var verifyTestCmd = []string{"go", "test", "./..."}

// producerGCInterval is how often the server sweeps idle producers' ingested git
// objects (council R39 housekeeping). Generous — disk hygiene, not a hot path.
const producerGCInterval = 10 * time.Minute

// runVerifyCatch is the `verify-catch` subcommand: it runs the SAME catch oracle
// (pipe.RunCatchCycle) over the given revisions and writes the deterministic
// verdict Transcript as JSON. This is the one binary that runs both in-process
// (today) and inside the #6c sandbox (later), so the verdict is identical
// wherever it runs.
func runVerifyCatch(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("verify-catch", flag.ContinueOnError)
	repo := fs.String("repo", ".", "git repo directory")
	base := fs.String("base", "", "base (pre-fix) revision")
	fix := fs.String("fix", "", "fix revision")
	tip := fs.String("tip", "", "trunk tip to integrate onto (defaults to -fix)")
	file := fs.String("file", "", "anchored file, relative to repo")
	line := fs.Int("line", 0, "1-based anchored line")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *base == "" || *fix == "" || *file == "" || *line == 0 {
		return fmt.Errorf("verify-catch: -base, -fix, -file and -line are required")
	}
	tipRev := *tip
	if tipRev == "" {
		tipRev = *fix
	}
	hash, err := lineHashAt(*repo, *base, *file, *line)
	if err != nil {
		return err
	}
	anchor := reanchor.Anchor{Path: *file, Start: *line, End: *line, LineHash: hash}
	cr, err := pipe.RunCatchCycle(context.Background(), *repo, *base, *fix, tipRev, anchor, verifyTestCmd)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	return enc.Encode(pipe.Transcribe(cr))
}

// primarySessionKey is the registry key the "/" card registers under (mirrors
// app's defaultSessionKey). A -session may not reuse it: doing so would clobber
// the primary card's registry entry while main still holds both ledgers.
const primarySessionKey = "default"

// sessionRef is the identity of one registered review target — the registry key
// it Stores under. validateSessions checks these for key collisions before any
// session is registered.
type sessionRef struct {
	key string
}

// validateSessions rejects -session sets that would silently corrupt state: a key
// duplicating another session's or the reserved primary card's. Two sessions
// sharing a key would have the second registerSession Store clobber the first
// liveReg entry (orphaning a review target) AND fuse two economies onto one
// session subtree of the shared fabric — the isolation is keyed by the session
// token, so a unique key IS the isolation guarantee.
func validateSessions(refs []sessionRef) error {
	seenKey := map[string]bool{primarySessionKey: true}
	for _, r := range refs {
		if !fabric.ValidToken(r.key) {
			return fmt.Errorf("session key %q is not a valid subject token", r.key)
		}
		if seenKey[r.key] {
			return fmt.Errorf("session key %q is already in use (duplicate or reserved)", r.key)
		}
		seenKey[r.key] = true
	}
	return nil
}

// sessionFlag collects repeatable -session specs so one server can stand up ≥2
// review targets: the default card at "/" plus a keyed card at /?key=<key> per
// -session, each its own isolated economy.
type sessionFlag struct{ specs []string }

func (s *sessionFlag) String() string { return strings.Join(s.specs, " ") }

func (s *sessionFlag) Set(v string) error {
	s.specs = append(s.specs, v)
	return nil
}

// parseSessionSpec parses a "key=NAME,base=SHA,fix=SHA,file=F,line=N[,tip=SHA]"
// spec into the session key and its cycle config. tip defaults to fix (clean
// integration by construction). The session's economy is isolated by its key on
// the shared fabric, so it carries no ledger path — a unique key IS the isolation.
func parseSessionSpec(repo, spec string) (string, app.LiveConfig, error) {
	kv := map[string]string{}
	for _, pair := range strings.Split(spec, ",") {
		k, v, ok := strings.Cut(pair, "=")
		if !ok {
			return "", app.LiveConfig{}, fmt.Errorf("session %q: %q is not key=value", spec, pair)
		}
		kv[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}
	for _, req := range []string{"key", "base", "fix", "file", "line"} {
		if kv[req] == "" {
			return "", app.LiveConfig{}, fmt.Errorf("session %q: missing %s", spec, req)
		}
	}
	line, err := strconv.Atoi(kv["line"])
	if err != nil || line < 1 {
		return "", app.LiveConfig{}, fmt.Errorf("session %q: line must be a positive integer", spec)
	}
	hash, err := lineHashAt(repo, kv["base"], kv["file"], line)
	if err != nil {
		return "", app.LiveConfig{}, err
	}
	tip := kv["tip"]
	if tip == "" {
		tip = kv["fix"]
	}
	return kv["key"], app.LiveConfig{
		RepoDir:       repo,
		BaseRev:       kv["base"],
		FixRev:        kv["fix"],
		TipRev:        tip,
		Anchor:        reanchor.Anchor{Path: kv["file"], Start: line, End: line, LineHash: hash},
		TestCmd:       []string{"go", "test", "./..."},
		MaxConcurrent: 2,
	}, nil
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "verify-catch" {
		if err := runVerifyCatch(os.Args[2:], os.Stdout); err != nil {
			log.Fatalf("packets verify-catch: %v", err)
		}
		return
	}

	repo := flag.String("repo", ".", "git repo directory")
	base := flag.String("base", "", "base (pre-fix) revision")
	fix := flag.String("fix", "", "fix revision")
	tip := flag.String("tip", "", "trunk tip to integrate onto (defaults to -fix)")
	file := flag.String("file", "", "anchored file, relative to repo")
	line := flag.Int("line", 0, "1-based anchored line")
	ledgerPath := flag.String("ledger", "catches", "durable economy store base; the JetStream log lives in a <ledger>-fabric directory beside it")
	addr := flag.String("addr", ":3000", "listen address")
	cageImage := flag.String("cage-image", "packets-cage:dev", "Docker image the claim verifier runs producer-submitted work in")
	container := flag.Bool("container", false, "run the primary session's LIVE work orders in the hardened agent container (harness.RunContainer) instead of the host subprocess; needs the packets-agent image + an ANTHROPIC_API_KEY")
	seedBandwidth := flag.Int("bandwidth", 0, "seed N cleared attention intervals on the primary session at boot so live orders can be placed without first answering questions (dev/demo; each interval is worth ~3 attention bandwidth)")
	var sessions sessionFlag
	flag.Var(&sessions, "session", "additional keyed review target served at /?key=NAME; repeatable: key=NAME,base=SHA,fix=SHA,file=F,line=N[,tip=SHA]")
	var backlog backlogFlag
	flag.Var(&backlog, "backlog", "seed a fundable work-order target on the primary session so Spend can dispatch+fill it; repeatable: base=SHA,fix=SHA,file=F,line=N[,tip=SHA]")
	var live liveFlag
	flag.Var(&live, "live", "seed a PROMPT-BEARING live work-order on the primary session (a real Claude Code harness produces the fix); repeatable: file=F,line=N,base=SHA[,tip=SHA],prompt=<task>")
	producerListen := flag.String("producer-listen", "", "bind an AUTHENTICATED NATS socket (host:port) for cross-process producers to submit claims; empty keeps the fabric in-process-only")
	var producers producerFlag
	flag.Var(&producers, "producer", "authorize a cross-process producer to submit claims to its session's claim subtree (never mint); repeatable: key:user:pass")
	flag.Parse()

	if len(producers.grants) > 0 && *producerListen == "" {
		log.Fatal("packets: -producer needs -producer-listen <host:port> to bind the authenticated socket")
	}

	// The primary anchor is OPTIONAL: with all four flags set, the default session
	// runs a catch cycle on that line; without them the server boots flag-less into
	// the fleet board + settings (no default session, "/" is a calm landing) and
	// sessions are created from the board.
	configured := *base != "" && *fix != "" && *file != "" && *line != 0
	if !configured && (len(backlog.specs) > 0 || len(live.specs) > 0) {
		log.Fatal("packets: -backlog/-live seed the PRIMARY session — they need -base, -fix, -file, -line")
	}

	liveCfg := app.LiveConfig{
		RepoDir:      *repo,
		TestCmd:      []string{"go", "test", "./..."},
		LedgerPath:   *ledgerPath,
		UseContainer: *container,
		ListenAddr:   *producerListen,
		Grants:       producers.grants,
		// Cap concurrent catch cycles: each is several full-suite runs (#15), and
		// per-cycle wall-time stays flat through ~2 concurrent on the bench, so 2 is
		// the honest default ceiling — connects beyond it queue, never pile on.
		MaxConcurrent: 2,
	}
	if configured {
		tipRev := *tip
		if tipRev == "" {
			tipRev = *fix // no separate trunk tip given → integrate onto the fix itself (clean by construction)
		}
		hash, err := lineHashAt(*repo, *base, *file, *line)
		if err != nil {
			log.Fatalf("packets: %v", err)
		}
		liveCfg.BaseRev = *base
		liveCfg.FixRev = *fix
		liveCfg.TipRev = tipRev
		liveCfg.Anchor = reanchor.Anchor{Path: *file, Start: *line, End: *line, LineHash: hash}

		// Seed any -backlog/-live specs as fundable work-order targets on the primary
		// session (a -backlog target replays a pre-funded base→fix diff; a -live target
		// carries a prompt a real harness fills). Computed only when there is a primary
		// session to attach them to.
		var dispatchBacklog []ledger.Target
		for _, spec := range backlog.specs {
			tgt, err := parseBacklogSpec(spec)
			if err != nil {
				log.Fatalf("packets: %v", err)
			}
			tgt.LineHash, err = lineHashAt(*repo, tgt.BaseRev, tgt.Path, tgt.Line)
			if err != nil {
				log.Fatalf("packets: backlog %q: %v", spec, err)
			}
			dispatchBacklog = append(dispatchBacklog, tgt)
		}
		for _, spec := range live.specs {
			tgt, err := parseLiveSpec(spec)
			if err != nil {
				log.Fatalf("packets: %v", err)
			}
			tgt.LineHash, err = lineHashAt(*repo, tgt.BaseRev, tgt.Path, tgt.Line)
			if err != nil {
				log.Fatalf("packets: live %q: %v", spec, err)
			}
			dispatchBacklog = append(dispatchBacklog, tgt)
		}
		liveCfg.DispatchBacklog = dispatchBacklog
	}

	application, ledgerLog, err := app.NewServer(liveCfg)
	if err != nil {
		log.Fatalf("packets: %v", err)
	}
	defer ledgerLog.Close()

	// Dev/demo: seed cleared attention intervals so the Lead starts with spendable
	// bandwidth and can author + place a live order without first answering a review
	// question to earn it. Each interval is a block→unblock pair cleared instantly
	// (the throughput base + the fast-clear bonus). Uses only the public ledger
	// primitives; the events are real cleared-attention facts, just pre-seeded. The
	// default session is usable with just a repo (prompt-authoring), so this is gated
	// on a repo — not on the full primary anchor.
	hasRepo := *repo != ""
	if *seedBandwidth > 0 && !hasRepo {
		log.Fatal("packets: -bandwidth seeds the PRIMARY session — it needs -repo")
	}
	for i := 0; hasRepo && i < *seedBandwidth; i++ {
		now := time.Now()
		id := fmt.Sprintf("seed-bandwidth-%d", i)
		if err := ledgerLog.AppendBlock(id, now); err != nil {
			log.Fatalf("packets: seed bandwidth: %v", err)
		}
		if err := ledgerLog.AppendUnblock(id, now); err != nil {
			log.Fatalf("packets: seed bandwidth: %v", err)
		}
	}
	if *seedBandwidth > 0 && hasRepo {
		if bw, err := ledgerLog.Bandwidth(); err == nil {
			log.Printf("packets: seeded %d attention bandwidth on the primary session", bw)
		}
	}

	// Parse every -session and validate the whole set for key/ledger-path
	// collisions BEFORE opening any session ledger — a clobbered registry entry
	// or two handles on one JSONL must fail fast, not corrupt state at runtime.
	type parsedSession struct {
		key string
		cfg app.LiveConfig
	}
	var parsed []parsedSession
	var refs []sessionRef
	for _, spec := range sessions.specs {
		key, cfg, err := parseSessionSpec(*repo, spec)
		if err != nil {
			log.Fatalf("packets: %v", err)
		}
		parsed = append(parsed, parsedSession{key: key, cfg: cfg})
		refs = append(refs, sessionRef{key: key})
	}
	if err := validateSessions(refs); err != nil {
		log.Fatalf("packets: %v", err)
	}
	for _, p := range parsed {
		sessionLog, err := app.AddSession(p.key, p.cfg)
		if err != nil {
			log.Fatalf("packets: %v", err)
		}
		defer sessionLog.Close()
		log.Printf("packets: also serving session %q at /?key=%s — watch %s:%d resolve", p.key, p.key, p.cfg.Anchor.Path, p.cfg.Anchor.Start)
	}

	// All sessions are now registered, so start exactly one cage claim consumer
	// per session (the StartClaimConsumers single-call/register-first contract).
	// The consumers verify producer-submitted claims in the hardened Docker cage;
	// the shutdown-scoped ctx stops them on SIGINT.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	app.StartCageClaimConsumers(ctx, *cageImage, sandbox.DockerRunner{})

	// Background housekeeping: periodically reclaim idle producers' ingested git
	// objects (never a session with a claim in flight). Generous interval — this
	// is disk hygiene, not a hot path. Stops with ctx on SIGINT.
	app.StartProducerGC(ctx, producerGCInterval)

	switch {
	case configured:
		log.Printf("packets: serving the review card on %s — open it and watch %s:%d resolve", *addr, *file, *line)
	case hasRepo:
		log.Printf("packets: serving the session card on %s — author prompt orders against %s, or create more sessions from the board", *addr, *repo)
	default:
		log.Printf("packets: serving the fleet board on %s — open it; create sessions from the board (no repo configured)", *addr)
	}
	log.Fatal(http.ListenAndServe(*addr, application))
}

// lineHashAt returns the content hash of the anchored line at the base
// revision, the anchor's identity the re-anchor step verifies against.
func lineHashAt(repo, rev, file string, line int) (string, error) {
	cmd := exec.Command("git", "show", rev+":"+file)
	cmd.Dir = repo
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("read %s@%s: %w", file, rev, err)
	}
	lines := strings.Split(out.String(), "\n")
	if line < 1 || line > len(lines) {
		return "", fmt.Errorf("line %d out of range in %s@%s", line, file, rev)
	}
	return reanchor.HashLines(lines[line-1]), nil
}
