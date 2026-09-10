//go:build browser

// This test drives a REAL headless Chrome against the live /review surface to
// verify the one thing unit tests can't: that selecting line(s) in the Monaco
// diff editor fires the viaannotate CustomEvent through the datastar bridge,
// fills the adjustment anchor signals, and — after leaving the adjustment —
// persists a durable annotation carrying the selected RANGE. The server-side
// pieces are unit-tested; this proves the browser half of the loop is wired.
// Build-tagged `browser` so it never runs in normal CI; it needs a Chrome
// binary and the Monaco assets on disk (Monaco's CDN is unreachable here, so
// the CDN requests are intercepted and fulfilled from MONACO_VS — reusing the
// interceptor + contentType helper from authoring_browser_test.go):
//
//	CFT_CHROME=/tmp/chrome-linux64/chrome MONACO_VS=/tmp/monaco/package/min/vs \
//	  go test -tags browser -run TestAnnotationBrowser ./internal/app/ -v
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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/packets/internal/fabric"
	"github.com/joaomdsg/packets/internal/ledger"
)

func TestAnnotationBrowser_selectingARangeInTheDiffPersistsARangedAnnotation(t *testing.T) {
	chromePath := os.Getenv("CFT_CHROME")
	monacoVS := os.Getenv("MONACO_VS")
	if chromePath == "" || monacoVS == "" {
		t.Skip("set CFT_CHROME and MONACO_VS to run the browser test")
	}

	resetConsumersForTest()
	// A real repo with a real base→fix diff, so the Monaco diff editor mounts on a
	// file with lines to select (main.go: 3 lines).
	repo, base, fix := initMeasurableRepo(t)

	ctx := context.Background()
	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	log := ledger.Bind(f, "annbrowse", "i")
	fundSend(t, log, "d1", ledger.Target{BaseRev: base, FixRev: fix, TipRev: fix, Path: "main.go", Line: 1})
	registerSession("annbrowse", LiveConfig{RepoDir: repo, BaseRev: "own-b", FixRev: "own-f", Anchor: anchorForCap(), TestCmd: []string{"true"}}, log)

	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	viaApp, defLog, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	})
	require.NoError(t, err)
	server := httptest.NewServer(viaApp)
	t.Cleanup(func() { _ = defLog.Close() })

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.ExecPath(chromePath), chromedp.NoSandbox,
		chromedp.Flag("headless", true), chromedp.Flag("disable-gpu", true),
	)
	allocCtx, cancelAlloc := chromedp.NewExecAllocator(ctx, opts...)
	t.Cleanup(cancelAlloc)
	bctx, cancel := chromedp.NewContext(allocCtx)
	t.Cleanup(cancel)

	// Serve Monaco from disk (CDN unreachable) — same interceptor as the authoring
	// browser test (contentType + monacoCDNMarker live there, same package).
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

	// setSelection on the modified editor fires onDidChangeCursorSelection, which
	// dispatches viaannotate {file, start:"1", end:"3"} — a real ranged selection.
	const selectRange = `(function(){
	  var des = (monaco.editor.getDiffEditors && monaco.editor.getDiffEditors()) || [];
	  if (!des.length) return 'no-diff-editor';
	  var me = des[0].getModifiedEditor();
	  me.setSelection({startLineNumber:1, startColumn:1, endLineNumber:3, endColumn:1});
	  me.focus();
	  return 'ok';
	})()`

	var selResult, adjFile, adjLine string
	err = chromedp.Run(bctx,
		fetch.Enable(),
		chromedp.Navigate(server.URL+"/review?key=annbrowse&wo=1"),
		chromedp.WaitVisible(".monaco-editor", chromedp.ByQuery), // the diff editor mounted
		chromedp.Sleep(1200*time.Millisecond),                    // let the editor + selection listener settle
		chromedp.Evaluate(selectRange, &selResult),
		chromedp.Sleep(300*time.Millisecond), // the viaannotate event → datastar signal update
		// The adjustment form's anchor inputs are bound to the signals the selection set.
		chromedp.Value(".review-adjust__file", &adjFile, chromedp.ByQuery),
		chromedp.Value(".review-adjust__line", &adjLine, chromedp.ByQuery),
		// Type the comment and leave the adjustment — posts AddAdjustment.
		chromedp.SendKeys(".review-adjust__text", "this whole block ignores the error", chromedp.ByQuery),
		chromedp.Click(".review-adjust__submit", chromedp.ByQuery),
		chromedp.Sleep(600*time.Millisecond), // the post round-trip
	)
	require.NoError(t, err)
	require.Equal(t, "ok", selResult, "the Monaco diff editor was found and a range selected")

	// Screenshot for the human record.
	var png []byte
	if shotErr := chromedp.Run(bctx, chromedp.FullScreenshot(&png, 90)); shotErr == nil {
		_ = os.WriteFile("/tmp/annotation-browser.png", png, 0o644)
	}

	// The selection populated the adjustment anchor signals (the bridge worked).
	assert.Equal(t, "main.go", adjFile, "the selection filled the adjustment file signal")
	assert.Equal(t, "1", adjLine, "and the start line")

	// End to end: leaving the adjustment persisted a durable annotation with the
	// SELECTED RANGE (start 1, end 3) — the browser half of the loop is real.
	anns, err := log.Annotations()
	require.NoError(t, err)
	require.Len(t, anns, 1, "leaving the adjustment persisted one durable annotation")
	assert.Equal(t, "main.go", anns[0].File)
	assert.Equal(t, 1, anns[0].StartLine, "anchored at the selection start")
	assert.Equal(t, 3, anns[0].EndLine, "and covering the selected range")
	assert.Equal(t, "this whole block ignores the error", anns[0].Body)
}
