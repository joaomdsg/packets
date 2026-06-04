// Command agntpr serves the single-user review wire (DESIGN §17): it runs one
// confirmed-catch cycle over two revisions and streams the verdict to a live
// review card over SSE, so a human opens a browser and watches one verdict go
// in-flight → resolved, with any catch appended to the ledger.
//
//	agntpr -repo . -base <weakSHA> -fix <fixSHA> -file adult.go -line 4
//	open http://localhost:3000
package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strings"

	"github.com/joaomdsg/agntpr/internal/app"
	"github.com/joaomdsg/agntpr/internal/reanchor"
)

func main() {
	repo := flag.String("repo", ".", "git repo directory")
	base := flag.String("base", "", "base (pre-fix) revision")
	fix := flag.String("fix", "", "fix revision")
	tip := flag.String("tip", "", "trunk tip to integrate onto (defaults to -fix)")
	file := flag.String("file", "", "anchored file, relative to repo")
	line := flag.Int("line", 0, "1-based anchored line")
	ledgerPath := flag.String("ledger", "catches.jsonl", "catch ledger path")
	addr := flag.String("addr", ":3000", "listen address")
	flag.Parse()

	if *base == "" || *fix == "" || *file == "" || *line == 0 {
		log.Fatal("agntpr: -base, -fix, -file and -line are required")
	}
	tipRev := *tip
	if tipRev == "" {
		tipRev = *fix // no separate trunk tip given → integrate onto the fix itself (clean by construction)
	}

	hash, err := lineHashAt(*repo, *base, *file, *line)
	if err != nil {
		log.Fatalf("agntpr: %v", err)
	}

	application, ledgerLog, err := app.NewServer(app.LiveConfig{
		RepoDir:    *repo,
		BaseRev:    *base,
		FixRev:     *fix,
		TipRev:     tipRev,
		Anchor:     reanchor.Anchor{Path: *file, Start: *line, End: *line, LineHash: hash},
		TestCmd:    []string{"go", "test", "./..."},
		LedgerPath: *ledgerPath,
	})
	if err != nil {
		log.Fatalf("agntpr: %v", err)
	}
	defer ledgerLog.Close()

	log.Printf("agntpr: serving the review card on %s — open it and watch %s:%d resolve", *addr, *file, *line)
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
