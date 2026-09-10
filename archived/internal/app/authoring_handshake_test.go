package app_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-via/via/vt"

	"github.com/joaomdsg/packets/internal/app"
)

// AuthorHandshake writes the Lead's authored contract to the protected
// handshake/ directory — the one place internal/settle's deny-rule then
// refuses any later agent turn from touching. NOT parallel (shared globals).
func TestLiveCard_authorHandshakeWritesToTheProtectedDirectoryAndCachesIt(t *testing.T) {
	repo := initGitRepoForPacket(t)
	head := gitPacket(t, repo, "rev-parse", "HEAD")

	server, _ := bootDefaultServer(t, defaultBootCfg)
	log := addFundedSession(t, "authorhs", app.LiveConfig{RepoDir: repo, BaseRev: head, Anchor: anchorForCap(), TestCmd: []string{"true"}})
	fundBandwidth(t, log) // the authoring control only renders with bandwidth to place with

	tc := vt.NewClient(t, server, "/?key=authorhs")
	require.Equal(t, 200, tc.Action((&app.LiveCard{Key: "authorhs"}).AuthorHandshake).
		WithSignal("handshakedraft", "package handshake\n\nfunc TestSpec(t *testing.T) {}\n").
		WithSignal("handshakestrengthpick", "examples").
		Fire())

	got, err := os.ReadFile(filepath.Join(repo, "handshake", "spec_test.go"))
	require.NoError(t, err)
	// AuthorHandshake trims the draft like every other compose signal (AnalyzeDraft
	// does the same to Draft) — leading/trailing whitespace is authoring
	// hygiene, not content.
	assert.Equal(t, "package handshake\n\nfunc TestSpec(t *testing.T) {}", string(got))

	body := bodyOf(vt.NewClient(t, server, "/?key=authorhs").HTML())
	assert.Contains(t, body, `data-state="authored"`, "the card reflects that a handshake now exists")
}

// A blank draft is nothing to author — a silent no-op, never an empty
// handshake file. NOT parallel (shared globals).
func TestLiveCard_authorHandshakeIsANoOpOnAnEmptyDraft(t *testing.T) {
	repo := initGitRepoForPacket(t)
	head := gitPacket(t, repo, "rev-parse", "HEAD")

	server, _ := bootDefaultServer(t, defaultBootCfg)
	_ = addFundedSession(t, "authorhsempty", app.LiveConfig{RepoDir: repo, BaseRev: head, Anchor: anchorForCap(), TestCmd: []string{"true"}})

	tc := vt.NewClient(t, server, "/?key=authorhsempty")
	require.Equal(t, 200, tc.Action((&app.LiveCard{Key: "authorhsempty"}).AuthorHandshake).
		WithSignal("handshakestrengthpick", "examples").
		Fire())

	_, err := os.Stat(filepath.Join(repo, "handshake"))
	assert.True(t, os.IsNotExist(err), "an empty draft must write nothing")
}

// Strength is self-declared, never inferred or defaulted — a blank/unknown
// pick is a silent no-op, not a guess at what the Lead meant. NOT parallel
// (shared globals).
func TestLiveCard_authorHandshakeIsANoOpWithoutADeclaredStrength(t *testing.T) {
	repo := initGitRepoForPacket(t)
	head := gitPacket(t, repo, "rev-parse", "HEAD")

	server, _ := bootDefaultServer(t, defaultBootCfg)
	_ = addFundedSession(t, "authorhsnostrength", app.LiveConfig{RepoDir: repo, BaseRev: head, Anchor: anchorForCap(), TestCmd: []string{"true"}})

	tc := vt.NewClient(t, server, "/?key=authorhsnostrength")
	require.Equal(t, 200, tc.Action((&app.LiveCard{Key: "authorhsnostrength"}).AuthorHandshake).
		WithSignal("handshakedraft", "package handshake\n\nfunc TestSpec(t *testing.T) {}\n").
		Fire())

	_, err := os.Stat(filepath.Join(repo, "handshake"))
	assert.True(t, os.IsNotExist(err), "no strength was declared — nothing is written")
}

// The card renders the handshake authoring control so a Lead can compose one
// before placing a live order. NOT parallel (shared globals).
func TestLiveCard_rendersTheHandshakeAuthoringControl(t *testing.T) {
	server, _ := bootDefaultServer(t, defaultBootCfg)
	log := addFundedSession(t, "handshakecontrol", app.LiveConfig{RepoDir: ".", BaseRev: "b", Anchor: anchorForCap(), TestCmd: []string{"true"}})
	fundBandwidth(t, log)

	body := bodyOf(vt.NewClient(t, server, "/?key=handshakecontrol").HTML())
	assert.Contains(t, body, "/_action/AuthorHandshake", "the card renders the author-handshake action binding")
	assert.Contains(t, body, `data-bind="handshakedraft"`, "with a control bound to the handshake draft signal")
	assert.Contains(t, body, `data-bind="handshakestrengthpick"`, "and a control bound to the self-declared strength signal")
}
