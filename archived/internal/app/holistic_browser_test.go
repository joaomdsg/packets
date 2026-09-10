//go:build browser

// This test drives a REAL headless Chrome against the live server to holistically
// verify the built MVP end to end — compose -> forward -> hold -> inspect ->
// deliver, the same journey scripts/demo.sh walks a human through by hand, but
// exercised here by a real browser clicking a real button. A real click (never
// curl, which can't reproduce Datastar's tab/session cookie handshake without
// hand-rolling it) proves the render path, the SSE re-render, and the
// click-to-dispatch wiring all work together, not just the pure functions
// underneath them. It needs a Chrome/Chromium binary on disk:
//
//	CFT_CHROME=/usr/bin/chromium go test -tags browser -run TestHolisticBrowser ./internal/app/ -v
package app_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"net/http/httptest"

	"github.com/chromedp/chromedp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/packets/internal/app"
	"github.com/joaomdsg/packets/internal/ledger"
)

// brokenGo is adultGo with a syntax error — a real build-gate (G4) failure, so the
// packet dispatched against it holds for a real reason, never a fabricated one.
const brokenGo = "package adult\n\nfunc IsAdult(age int) bool {\n\treturn age >= 18 this is not valid go\n}\n"

// holisticBannedWords is a lighter, representative subset of vocabulary_internal_
// test.go's bannedWordPattern — this test's job is proving the REAL rendered
// browser DOM (after live SSE patches, not just server-side HTML strings) never
// regresses on the highest-signal retired terms; the canonical, exhaustive
// regression coverage lives in that file.
var holisticBannedWords = regexp.MustCompile(`(?i)\b(PRs?|merged?|merging|approved?|approving|sessions?|boards?|oracles?|verdicts?|bounced|LGTM)\b`)

var scriptOrStyleBlock = regexp.MustCompile(`(?is)<(script|style)[^>]*>.*?</(script|style)>`)

func assertNoRetiredVocabularyOnPage(t *testing.T, where, html string) {
	t.Helper()
	visible := scriptOrStyleBlock.ReplaceAllString(html, " ")
	text := regexp.MustCompile(`<[^>]*>`).ReplaceAllString(visible, " ")
	if loc := holisticBannedWords.FindStringIndex(text); loc != nil {
		lo, hi := loc[0]-40, loc[1]+40
		if lo < 0 {
			lo = 0
		}
		if hi > len(text) {
			hi = len(text)
		}
		t.Errorf("%s: retired vocabulary %q rendered on a REAL browser page — context: %q", where, text[loc[0]:loc[1]], text[lo:hi])
	}
}

// buildPacketsBinary builds the packets CLI once for this test's `deployed`
// command — the same binary an operator would run, exercised as a real subprocess
// rather than calling internal Go functions directly.
func buildPacketsBinary(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "packets")
	cmd := exec.Command("go", "build", "-o", bin, "../../cmd/packets")
	out, err := cmd.CombinedOutput()
	require.NoErrorf(t, err, "go build ./cmd/packets: %s", out)
	return bin
}

func TestHolisticBrowser_composeForwardHoldInspectDeliver(t *testing.T) {
	chromePath := os.Getenv("CFT_CHROME")
	if chromePath == "" {
		t.Skip("set CFT_CHROME to run the holistic browser verification")
	}

	dir := initRepo(t)
	write(t, dir, "go.mod", "module adultapp\n\ngo 1.23\n")
	write(t, dir, "adult.go", adultGo)
	write(t, dir, "adult_test.go", weakTest)
	base := commitAll(t, dir, "base: weak boundary test")
	write(t, dir, "adult_test.go", strongTest)
	goodFix := commitAll(t, dir, "fix: strengthen the boundary")

	runGit(t, dir, "checkout", "-q", base)
	write(t, dir, "adult.go", brokenGo)
	badFix := commitAll(t, dir, "fix: broken build")
	runGit(t, dir, "checkout", "-q", goodFix)

	ledgerPath := filepath.Join(t.TempDir(), "holistic.jsonl")
	viaApp, log, err := app.NewServer(app.LiveConfig{
		RepoDir: dir, BaseRev: base, FixRev: goodFix, TipRev: goodFix,
		Anchor: anchor(), TestCmd: goTestCmd, LedgerPath: ledgerPath,
		SendBacklog: []ledger.Target{{BaseRev: base, FixRev: badFix, TipRev: badFix, Path: "adult.go", Line: 4}},
	})
	require.NoError(t, err)
	server := httptest.NewServer(viaApp)
	t.Cleanup(server.Close)
	t.Cleanup(func() { _ = log.Close() })

	// Seed enough attention bandwidth for one dispatch (a cleared block/unblock pair
	// earns up to 3; one is enough for the single backlog target this test funds).
	now := time.Unix(1_700_000_000, 0)
	require.NoError(t, log.AppendBlock("seed", now))
	require.NoError(t, log.AppendUnblock("seed", now.Add(30*time.Second)))

	ctx := context.Background()
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.ExecPath(chromePath), chromedp.NoSandbox,
		chromedp.Flag("headless", true), chromedp.Flag("disable-gpu", true))
	allocCtx, cancelAlloc := chromedp.NewExecAllocator(ctx, opts...)
	t.Cleanup(cancelAlloc)
	bctx, cancel := chromedp.NewContext(allocCtx)
	t.Cleanup(cancel)

	shot := func(name string) {
		var png []byte
		if serr := chromedp.Run(bctx, chromedp.FullScreenshot(&png, 90)); serr == nil {
			_ = os.WriteFile(filepath.Join(os.TempDir(), "holistic-"+name+".png"), png, 0o644)
		}
	}

	// compose: the Console loads and, on the SAME load, runs the primary session's
	// own gauntlet from -base/-fix — a packet composing itself from a real catch.
	var body string
	require.NoError(t, chromedp.Run(bctx,
		chromedp.Navigate(server.URL+"/"),
		chromedp.WaitVisible(".console", chromedp.ByQuery),
		chromedp.OuterHTML("html", &body),
	))
	shot("01-console-loaded")
	assertNoRetiredVocabularyOnPage(t, "/ (initial)", body)

	// A REAL click — never curl — fires the compose affordance for the bad-fix
	// backlog target, exercising Datastar's tab handshake + SetSignal + POST wiring.
	require.NoError(t, chromedp.Run(bctx,
		chromedp.WaitVisible(".bench__fund", chromedp.ByQuery),
		chromedp.Click(".bench__fund", chromedp.ByQuery),
	))

	// forward + hold: the own cycle's real confirmed catch settles to verified; the
	// dispatched bad-fix packet's real build-gate failure settles to held — both
	// real gate outcomes, watched over the live SSE connection (no reload).
	require.Eventually(t, func() bool {
		if oerr := chromedp.Run(bctx, chromedp.OuterHTML("html", &body)); oerr != nil {
			return false
		}
		return strings.Contains(body, "console__rail--settled") && !strings.Contains(body, "nothing settled yet")
	}, 60*time.Second, 500*time.Millisecond, "the dispatched packets settle (verified/held) within the timeout")
	shot("02-console-settled")
	assertNoRetiredVocabularyOnPage(t, "/ (settled)", body)
	assert.Contains(t, body, "held", "the broken-build packet is a real, visible hold — never a silent drop")

	// inspect: open the Inspector for the held packet (pkt#1 — the FIRST dispatch;
	// the primary session's own cycle mints its packet OUTSIDE the dispatch ledger)
	// and confirm the page renders with no retired vocabulary either.
	require.NoError(t, chromedp.Run(bctx,
		chromedp.Navigate(server.URL+"/review?wo=1"),
		chromedp.WaitVisible(".inspector", chromedp.ByQuery),
		chromedp.OuterHTML("html", &body),
	))
	shot("03-inspector-held")
	assertNoRetiredVocabularyOnPage(t, "/review?wo=1", body)

	// deliver: ACK the primary session's own verified packet via the REAL CLI
	// binary — the same command an operator runs — while the server is STILL
	// live, proving the two processes can share the ledger concurrently. The
	// Console must show delivered only AFTER this explicit, external ACK.
	packetsBin := buildPacketsBinary(t)
	deploy := exec.Command(packetsBin, "deployed", "-ledger", strings.TrimSuffix(ledgerPath, ".jsonl"), "-session", "default", "-wo", "1")
	out, derr := deploy.CombinedOutput()
	require.NoErrorf(t, derr, "packets deployed: %s", out)

	require.NoError(t, chromedp.Run(bctx,
		chromedp.Navigate(server.URL+"/"),
		chromedp.WaitVisible(".console", chromedp.ByQuery),
		chromedp.OuterHTML("html", &body),
	))
	shot("04-console-delivered")
	assertNoRetiredVocabularyOnPage(t, "/ (delivered)", body)
}
