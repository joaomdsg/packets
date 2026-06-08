package app_test

import (
	"bufio"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-via/via"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/packets/internal/app"
	"github.com/joaomdsg/packets/internal/catch"
	"github.com/joaomdsg/packets/internal/ledger"
)

func catchAt(line int) ledger.CatchRecord {
	return ledger.CatchRecord{
		Outcome:           catch.Catch,
		Path:              "adult.go",
		Line:              line,
		BeforeRev:         "aaaa",
		AfterRev:          "bbbb",
		BeforeInventory:   []string{">="},
		AfterInventory:    []string{">="},
		MutantsConsidered: 1,
		ReasonTag:         "catch",
	}
}

func bootServer(t *testing.T) (*httptest.Server, *ledger.Log) {
	t.Helper()
	var server *httptest.Server
	_, log, err := app.NewServer(app.LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f",
		TestCmd: []string{"true"},
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })
	return server, log
}

func getStream(t *testing.T, ctx context.Context, url string) *http.Response {
	t.Helper()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	require.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp.Body.Close() })
	return resp
}

func awaitFrame(t *testing.T, body *http.Response, want string) {
	t.Helper()
	lines := make(chan string, 64)
	go func() {
		defer close(lines)
		sc := bufio.NewScanner(body.Body)
		for sc.Scan() {
			lines <- sc.Text()
		}
	}()
	deadline := time.After(5 * time.Second)
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

func TestNewServer_streamsTheDefaultEconomyOverSSEAtStreamRoute(t *testing.T) {
	// Not parallel: NewServer mutates package globals (liveFabric/liveReg)
	// shared with the other live tests.
	server, log := bootServer(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	resp := getStream(t, ctx, server.URL+"/stream")
	assert.Contains(t, resp.Header.Get("Content-Type"), "text/event-stream")

	require.NoError(t, log.Append(catchAt(4)))
	awaitFrame(t, resp, `"balance":1`)
}

func TestNewServer_streamsAKeyedSessionsOwnEconomyNotTheDefaults(t *testing.T) {
	server, _ := bootServer(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// A second registered session with its own economy; the default gets nothing.
	logB, err := app.AddSession("streamB", app.LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", TestCmd: []string{"true"},
	})
	require.NoError(t, err)
	require.NoError(t, logB.Append(catchAt(4)))
	require.NoError(t, logB.Append(catchAt(5)))

	resp := getStream(t, ctx, server.URL+"/stream?key=streamB")
	// balance 2 can only come from streamB's two mints — the default has none.
	awaitFrame(t, resp, `"balance":2`)
}

func TestNewServer_refusesAStreamKeyThatIsNotARegisteredSession(t *testing.T) {
	server, _ := bootServer(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// A wildcard token would otherwise inject a fleet-wide subject filter; an
	// unregistered key has no economy to serve. Both must be refused, not streamed.
	for _, key := range []string{"*", ">", "neverRegistered"} {
		resp := getStream(t, ctx, server.URL+"/stream?key="+key)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode, "key %q must be refused", key)
	}
}

func TestAddSession_rejectsAKeyThatWouldCorruptItsSubjectToken(t *testing.T) {
	bootServer(t) // sets liveFabric

	// AddSession is the programmatic registration boundary: a key carrying a
	// subject separator/wildcard must be refused before it binds an economy,
	// not left to the documented caller-contract.
	_, err := app.AddSession("bad.key", app.LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", TestCmd: []string{"true"},
	})
	require.Error(t, err)
}

func TestNewServer_servesTheFleetBoardOverSSEAtFleetRoute(t *testing.T) {
	server, log := bootServer(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	resp := getStream(t, ctx, server.URL+"/fleet")
	assert.Contains(t, resp.Header.Get("Content-Type"), "text/event-stream")

	require.NoError(t, log.Append(catchAt(4)))
	// The default session appears as a fleet row off the stream, end-to-end.
	awaitFrame(t, resp, `"key":"default"`)
}
