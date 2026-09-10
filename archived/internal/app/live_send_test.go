package app_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-via/via/vt"

	"github.com/joaomdsg/packets/internal/app"
)

// Placing a live order with no handshake authored yet must be refused — the
// handshake is the contract the agent's turn cannot touch,
// so a live order with none is never dispatched. NOT parallel (shared
// liveReg/liveFabric).
func TestLiveCard_sendRefusesALivePacketWithNoHandshakeAuthored(t *testing.T) {
	repo := initGitRepoForPacket(t)
	head := gitPacket(t, repo, "rev-parse", "HEAD")

	server, _ := bootDefaultServer(t, defaultBootCfg)
	log := addFundedSession(t, "authornohs", app.LiveConfig{RepoDir: repo, BaseRev: head, Anchor: anchorForCap(), TestCmd: []string{"true"}})
	fundBandwidth(t, log)

	tc := vt.NewClient(t, server, "/?key=authornohs")
	resp := tc.Action((&app.LiveCard{Key: "authornohs"}).Send).
		WithSignal("draft", "add a feature.go file").Fire()
	require.Equal(t, 200, resp, "a refusal is still a calm 200, never an error status")

	got := orderRecordFor(t, log, 1)
	assert.Empty(t, got.Target.Prompt, "no order was funded — the prompt never dispatched")
	bw, err := log.Bandwidth()
	require.NoError(t, err)
	assert.Equal(t, 3, bw, "a refused placement spends no bandwidth")

	body := bodyOf(vt.NewClient(t, server, "/?key=authornohs").HTML())
	assert.Contains(t, body, "author a handshake before sending", "the refusal leaves an honest inline message")
}

// Two sessions never share a pending handshake — authoring one under session
// A must not let session B's Send succeed. NOT parallel (shared
// liveReg/liveFabric).
func TestLiveCard_pendingHandshakeDoesNotLeakAcrossSessions(t *testing.T) {
	repoA := initGitRepoForPacket(t)
	headA := gitPacket(t, repoA, "rev-parse", "HEAD")
	repoB := initGitRepoForPacket(t)
	headB := gitPacket(t, repoB, "rev-parse", "HEAD")

	server, _ := bootDefaultServer(t, defaultBootCfg)
	logA := addFundedSession(t, "isoa", app.LiveConfig{RepoDir: repoA, BaseRev: headA, Anchor: anchorForCap(), TestCmd: []string{"true"}})
	fundBandwidth(t, logA)
	logB := addFundedSession(t, "isob", app.LiveConfig{RepoDir: repoB, BaseRev: headB, Anchor: anchorForCap(), TestCmd: []string{"true"}})
	fundBandwidth(t, logB)

	tcA := vt.NewClient(t, server, "/?key=isoa")
	require.Equal(t, 200, tcA.Action((&app.LiveCard{Key: "isoa"}).AuthorHandshake).
		WithSignal("handshakedraft", "package handshake\n\nfunc TestSpec(t *testing.T) {}\n").
		WithSignal("handshakestrengthpick", "examples").
		Fire(), "session A authors its own handshake")

	tcB := vt.NewClient(t, server, "/?key=isob")
	require.Equal(t, 200, tcB.Action((&app.LiveCard{Key: "isob"}).Send).
		WithSignal("draft", "b's order").Fire(), "a refusal is still a calm 200")

	assert.Empty(t, orderRecordFor(t, logB, 1).Target.Prompt, "session B never authored a handshake — A's must not leak into B's placement")
}

// An empty prompt is not an order: placing one must be a silent no-op, never a
// funded work-order with no task. NOT parallel (shared globals).
func TestLiveCard_sendIsANoOpOnAnEmptyPrompt(t *testing.T) {
	repo := initGitRepoForPacket(t)
	head := gitPacket(t, repo, "rev-parse", "HEAD")

	server, _ := bootDefaultServer(t, defaultBootCfg)
	log := addFundedSession(t, "empty", app.LiveConfig{RepoDir: repo, BaseRev: head, Anchor: anchorForCap(), TestCmd: []string{"true"}})
	fundBandwidth(t, log)

	tc := vt.NewClient(t, server, "/?key=empty")
	require.Equal(t, 200, tc.Action((&app.LiveCard{Key: "empty"}).Send).WithSignal("draft", "   ").Fire())

	bw, err := log.Bandwidth()
	require.NoError(t, err)
	assert.Equal(t, 3, bw, "an empty prompt funds nothing — the bandwidth meter is untouched")
}

// The card must render the order-authoring control (a prompt input bound to the
// order signal + the place-order action) when there is balance to fund it, so the
// Lead has a way to author and place a live order. NOT parallel (shared globals).
func TestLiveCard_rendersThePacketAuthoringControlWhenFunded(t *testing.T) {
	server, _ := bootDefaultServer(t, defaultBootCfg)
	log := addFundedSession(t, "compose", app.LiveConfig{RepoDir: ".", BaseRev: "b", Anchor: anchorForCap(), TestCmd: []string{"true"}})
	fundBandwidth(t, log)

	body := bodyOf(vt.NewClient(t, server, "/?key=compose").HTML())
	require.Contains(t, body, "/_action/Send", "the card renders the place-order action binding")
	require.Contains(t, body, "authoring-editor", "with the editable Monaco editor as the draft source")
	require.Contains(t, body, "$draft=evt.detail.draft", "whose value is lifted into the order-prompt signal at place time")
}
