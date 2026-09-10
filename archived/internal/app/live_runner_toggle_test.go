package app_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/go-via/via/vt"

	"github.com/joaomdsg/packets/internal/catch"
	"github.com/joaomdsg/packets/internal/ledger"
)

func TestLiveCard_rendersTheAgentRunnerControl(t *testing.T) {
	// NOT parallel (shared globals).
	server, log := bootDefaultServer(t, defaultBootCfg)
	// The runner control sits with the funding controls, so the session needs
	// something to fund (a balance) for act-now — and the control — to render.
	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Line: 1, ReasonTag: "catch"}))

	body := bodyOf(vt.NewClient(t, server, "/").HTML())
	require.Contains(t, body, "live-runner__toggle", "the card carries a runner toggle control")
	require.Contains(t, body, "ToggleRunner", "the toggle is wired to the ToggleRunner action")
	require.Contains(t, body, "agent runner: host", "the current runner mode reads in plain words (host by default)")
}
