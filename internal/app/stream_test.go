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

func TestNewServer_streamsTheDefaultEconomyOverSSEAtStreamRoute(t *testing.T) {
	// Not parallel: NewServer mutates package globals (liveFabric/liveReg)
	// shared with the other live tests.
	var server *httptest.Server
	_, log, err := app.NewServer(app.LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f",
		TestCmd: []string{"true"},
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/stream", nil)
	require.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Contains(t, resp.Header.Get("Content-Type"), "text/event-stream")

	require.NoError(t, log.Append(ledger.CatchRecord{
		Outcome:           catch.Catch,
		Path:              "adult.go",
		Line:              4,
		BeforeRev:         "aaaa",
		AfterRev:          "bbbb",
		BeforeInventory:   []string{">="},
		AfterInventory:    []string{">="},
		MutantsConsidered: 1,
		ReasonTag:         "catch",
	}))

	lines := make(chan string, 64)
	go func() {
		defer close(lines)
		sc := bufio.NewScanner(resp.Body)
		for sc.Scan() {
			lines <- sc.Text()
		}
	}()
	deadline := time.After(5 * time.Second)
	for {
		select {
		case ln, ok := <-lines:
			require.True(t, ok, "stream ended before the balance frame arrived")
			if strings.Contains(ln, `"balance":1`) {
				return
			}
		case <-deadline:
			t.Fatal("timed out waiting for the balance frame on /stream")
		}
	}
}
