// Command packets serves the single-user review wire (DESIGN §17): it runs one
// confirmed-catch cycle over two revisions and streams the verdict to a live
// review card over SSE, so a human opens a browser and watches one verdict go
// in-flight → resolved, with any catch appended to the ledger.
//
//	packets -repo . -base <weakSHA> -fix <fixSHA> -file adult.go -line 4
//	open http://localhost:3000
package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strconv"
	"strings"

	"github.com/joaomdsg/packets/internal/app"
	"github.com/joaomdsg/packets/internal/reanchor"
)

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
// session subtree of the shared fabric — the isolation is now keyed by the
// session token, so a unique key IS the isolation guarantee (the per-file
// validation the JSONL substrate needed is retired with it).
func validateSessions(refs []sessionRef) error {
	seenKey := map[string]bool{primarySessionKey: true}
	for _, r := range refs {
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
	repo := flag.String("repo", ".", "git repo directory")
	base := flag.String("base", "", "base (pre-fix) revision")
	fix := flag.String("fix", "", "fix revision")
	tip := flag.String("tip", "", "trunk tip to integrate onto (defaults to -fix)")
	file := flag.String("file", "", "anchored file, relative to repo")
	line := flag.Int("line", 0, "1-based anchored line")
	ledgerPath := flag.String("ledger", "catches", "durable economy store base; the JetStream log lives in a <ledger>-fabric directory beside it")
	addr := flag.String("addr", ":3000", "listen address")
	var sessions sessionFlag
	flag.Var(&sessions, "session", "additional keyed review target served at /?key=NAME; repeatable: key=NAME,base=SHA,fix=SHA,file=F,line=N[,tip=SHA]")
	flag.Parse()

	if *base == "" || *fix == "" || *file == "" || *line == 0 {
		log.Fatal("packets: -base, -fix, -file and -line are required")
	}
	tipRev := *tip
	if tipRev == "" {
		tipRev = *fix // no separate trunk tip given → integrate onto the fix itself (clean by construction)
	}

	hash, err := lineHashAt(*repo, *base, *file, *line)
	if err != nil {
		log.Fatalf("packets: %v", err)
	}

	application, ledgerLog, err := app.NewServer(app.LiveConfig{
		RepoDir:    *repo,
		BaseRev:    *base,
		FixRev:     *fix,
		TipRev:     tipRev,
		Anchor:     reanchor.Anchor{Path: *file, Start: *line, End: *line, LineHash: hash},
		TestCmd:    []string{"go", "test", "./..."},
		LedgerPath: *ledgerPath,
		// Cap concurrent catch cycles: each is several full-suite runs (#15), and
		// per-cycle wall-time stays flat through ~2 concurrent on the bench, so 2 is
		// the honest default ceiling — connects beyond it queue, never pile on.
		MaxConcurrent: 2,
	})
	if err != nil {
		log.Fatalf("packets: %v", err)
	}
	defer ledgerLog.Close()

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

	log.Printf("packets: serving the review card on %s — open it and watch %s:%d resolve", *addr, *file, *line)
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
