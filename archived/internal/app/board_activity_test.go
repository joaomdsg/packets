package app_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/packets/internal/app"
)

// An idle session (no live fill) shows no beat — no dead "·" with nothing after
// it. NOT parallel (shared liveReg).
func TestBoardCard_omitsTheActivityBeatWhenTheAgentIsIdle(t *testing.T) {
	_, _ = bootDefaultServer(t, defaultBootCfg)
	boardSession(t, "act-idle", 0, nil) // registered, never given an activity beat

	rows := app.BoardRows()
	for _, r := range rows {
		if r.Key == "act-idle" {
			require.Empty(t, r.Activity, "an idle session carries no activity beat")
		}
	}
}
