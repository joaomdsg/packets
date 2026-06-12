//go:build browser

// This test drives a REAL headless Chrome (Chrome for Testing) against the live
// server to verify the one thing unit tests can't: that the editable Monaco editor
// mounts, typing into it fires the debounced re-analysis through the datastar
// bridge, the server runs the (stubbed) producer, and the analysis renders back —
// the full author→analyze loop in a browser. It is build-tagged `browser` so it
// never runs in normal CI; it needs a Chrome binary and the Monaco assets on disk:
//
//	CFT_CHROME=/tmp/chrome-linux64/chrome MONACO_VS=/tmp/monaco/package/min/vs \
//	  go test -tags browser -run TestAuthoringBrowser ./internal/app/ -v
//
// Monaco's CDN (cdn.jsdelivr.net) is not reachable here, so the test intercepts
// those requests via the CDP fetch domain and fulfills them from MONACO_VS.
package app

import (
	"context"
	"encoding/base64"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/chromedp/cdproto/fetch"
	"github.com/chromedp/chromedp"
	"github.com/stretchr/testify/require"

	"github.com/go-via/via"

	"github.com/joaomdsg/packets/internal/fabric"
	"github.com/joaomdsg/packets/internal/ledger"
)

const monacoCDNMarker = "/npm/monaco-editor@"

func contentType(path string) string {
	switch {
	case strings.HasSuffix(path, ".js"):
		return "text/javascript"
	case strings.HasSuffix(path, ".css"):
		return "text/css"
	case strings.HasSuffix(path, ".ttf"):
		return "font/ttf"
	default:
		return "application/octet-stream"
	}
}

func TestAuthoringBrowser_editorMountsTypingDrivesAnalysisRendersBack(t *testing.T) {
	chromePath := os.Getenv("CFT_CHROME")
	monacoVS := os.Getenv("MONACO_VS")
	if chromePath == "" || monacoVS == "" {
		t.Skip("set CFT_CHROME and MONACO_VS to run the browser test")
	}

	// A stubbed producer: the browser test's subject is the editor + bridge + render
	// round-trip, not a live claude run. The summary is the marker we assert renders.
	restore := analyzeDraft
	t.Cleanup(func() { analyzeDraft = restore })
	analyzeDraft = func(_ context.Context, _, _, _ string) (string, error) {
		return `{"summary":"PRODUCER-SAW-THE-DRAFT","ready":false,` +
			`"highlights":[{"start":0,"end":6,"note":"flagged span","severity":"question"}],` +
			`"questions":["What is the retry budget?"]}`, nil
	}

	resetConsumersForTest()
	ctx := context.Background()
	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	log := ledger.Bind(f, "browsz", "i")
	bbase := time.Unix(1_700_000_000, 0)
	require.NoError(t, log.AppendBlock("q1", bbase))
	require.NoError(t, log.AppendUnblock("q1", bbase.Add(30*time.Second))) // +3 bandwidth → compose renders
	registerSession("browsz", LiveConfig{RepoDir: ".", BaseRev: "b", Anchor: anchorForCap(), TestCmd: []string{"true"}}, log)

	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	_, defLog, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = defLog.Close() })

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.ExecPath(chromePath),
		chromedp.NoSandbox,
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
	)
	allocCtx, cancelAlloc := chromedp.NewExecAllocator(ctx, opts...)
	t.Cleanup(cancelAlloc)
	bctx, cancel := chromedp.NewContext(allocCtx)
	t.Cleanup(cancel)

	// Serve Monaco from disk: intercept cdn.jsdelivr.net (unreachable here) and
	// fulfill each min/vs/* request from MONACO_VS so the editor actually mounts.
	chromedp.ListenTarget(bctx, func(ev interface{}) {
		e, ok := ev.(*fetch.EventRequestPaused)
		if !ok {
			return
		}
		go func() {
			if i := strings.Index(e.Request.URL, monacoCDNMarker); i >= 0 {
				rel := e.Request.URL[strings.Index(e.Request.URL[i:], "/min/vs/")+i+len("/min/vs/"):]
				if body, rerr := os.ReadFile(filepath.Join(monacoVS, rel)); rerr == nil {
					_ = chromedp.Run(bctx, fetch.FulfillRequest(e.RequestID, 200).
						WithResponseHeaders([]*fetch.HeaderEntry{{Name: "Content-Type", Value: contentType(rel)}}).
						WithBody(base64.StdEncoding.EncodeToString(body)))
					return
				}
			}
			_ = chromedp.Run(bctx, fetch.ContinueRequest(e.RequestID))
		}()
	})

	var summary string
	var flagCount int
	err = chromedp.Run(bctx,
		fetch.Enable(),
		chromedp.Navigate(server.URL+"/?key=browsz"),
		chromedp.WaitVisible(".monaco-editor", chromedp.ByQuery), // the editor mounted
		chromedp.Click(".monaco-editor .view-lines", chromedp.ByQuery),
		chromedp.SendKeys(".monaco-editor textarea", "Add retry logic to the uploader.", chromedp.ByQuery),
		chromedp.Sleep(2500*time.Millisecond), // debounce (900ms) + analyze round-trip + SSE re-render
		chromedp.WaitVisible(".analysis__summary", chromedp.ByQuery),
		chromedp.Text(".analysis__summary", &summary, chromedp.ByQuery),
		chromedp.Evaluate(`document.querySelectorAll('.authoring-flag-question').length`, &flagCount),
	)
	require.NoError(t, err)

	// Screenshot for the human record.
	var png []byte
	if shotErr := chromedp.Run(bctx, chromedp.FullScreenshot(&png, 90)); shotErr == nil {
		_ = os.WriteFile("/tmp/authoring-browser.png", png, 0o644)
	}

	require.Contains(t, summary, "PRODUCER-SAW-THE-DRAFT",
		"typing into the Monaco editor drove the debounced analysis through the bridge and the producer's read rendered back")
	require.Greater(t, flagCount, 0, "the producer's flagged span decorated the editor inline")
}
