package bridge_test

import (
	"bufio"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/packets/internal/bridge"
	"github.com/joaomdsg/packets/internal/ledger"
)

func scanLines(body io.Reader) <-chan string {
	lines := make(chan string, 256)
	go func() {
		defer close(lines)
		sc := bufio.NewScanner(body)
		for sc.Scan() {
			lines <- sc.Text()
		}
	}()
	return lines
}

func awaitLine(t *testing.T, lines <-chan string, want string) {
	t.Helper()
	deadline := time.After(3 * time.Second)
	for {
		select {
		case ln, ok := <-lines:
			require.True(t, ok, "stream ended before a frame containing %q", want)
			if strings.Contains(ln, want) {
				return
			}
		case <-deadline:
			t.Fatalf("timed out waiting for a frame containing %q", want)
		}
	}
}

func connect(t *testing.T, ctx context.Context, url string) *http.Response {
	t.Helper()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	require.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp.Body.Close() })
	return resp
}

func TestHandler_streamsAnEconomySnapshotWhenACatchIsMinted(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	f := startFabric(t)
	log := ledger.Bind(f, "session", "i")
	srv := httptest.NewServer(bridge.Handler(f, "session", "i"))
	defer srv.Close()

	resp := connect(t, ctx, srv.URL)
	lines := scanLines(resp.Body)

	require.NoError(t, log.Append(sampleCatch()))
	// The full SSE data line pins both the `data: ` framing and the exact
	// JSON snapshot shape, not just a loose substring.
	awaitLine(t, lines, `data: {"balance":1,"catches":1,"orders":0,"queued":0}`)

	require.NoError(t, log.AppendSpend(1, "fund"))
	awaitLine(t, lines, `data: {"balance":0,"catches":1,"orders":0,"queued":0}`)
}

func TestHandler_sendsTheEventStreamContentTypeSoBrowsersTreatItAsSSE(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	f := startFabric(t)
	log := ledger.Bind(f, "session", "i")
	srv := httptest.NewServer(bridge.Handler(f, "session", "i"))
	defer srv.Close()

	resp := connect(t, ctx, srv.URL)
	assert.Contains(t, resp.Header.Get("Content-Type"), "text/event-stream")
	// Caching an event stream breaks live delivery, so it must be disabled.
	assert.Equal(t, "no-cache", resp.Header.Get("Cache-Control"))

	// A stub that only sets the header but never streams must not pass.
	lines := scanLines(resp.Body)
	require.NoError(t, log.Append(sampleCatch()))
	awaitLine(t, lines, `data: {"balance":1`)
}

func TestHandler_endsTheStreamWhenTheClientDisconnects(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	f := startFabric(t)
	log := ledger.Bind(f, "session", "i")
	srv := httptest.NewServer(bridge.Handler(f, "session", "i"))
	defer srv.Close()

	resp := connect(t, ctx, srv.URL)
	lines := scanLines(resp.Body)
	require.NoError(t, log.Append(sampleCatch()))
	awaitLine(t, lines, `"balance":1`) // alive before disconnect

	cancel() // client goes away
	require.Eventually(t, func() bool {
		select {
		case _, ok := <-lines:
			return !ok
		default:
			return false
		}
	}, 3*time.Second, 10*time.Millisecond, "stream did not end when the client disconnected")
}
