package app

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-via/via/vt"
)

// A picked folder name (a single segment, never an absolute path) must resolve to a
// repo UNDER the configured repos root — so the directory picker actually points the
// session at a real tree. NOT parallel (shared globals).
func TestBoardCard_resolvesAPickedFolderNameUnderTheReposRoot(t *testing.T) {
	resetConsumersForTest()
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "myproj"), 0o755))
	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	viaApp, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath, ReposRoot: root,
	})
	require.NoError(t, err)
	server = httptest.NewServer(viaApp)
	t.Cleanup(func() { _ = log.Close() })

	tc := vt.NewClient(t, server, "/board")
	require.Equal(t, 200, tc.Action((&BoardCard{}).CreateSession).
		WithSignal("newkey", "picked").WithSignal("newrepo", "myproj").Fire())

	cfg, slog := readLiveState("picked")
	require.NotNil(t, slog, "the created session is registered")
	assert.Equal(t, filepath.Join(root, "myproj"), cfg.RepoDir, "the picked folder resolves under the repos root")
}

// A malicious/odd pick must never escape the repos root: only the final path segment
// is used, so traversal collapses to a name joined under the root. NOT parallel.
func TestBoardCard_pickedFolderCannotEscapeTheReposRoot(t *testing.T) {
	resetConsumersForTest()
	root := t.TempDir()
	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	viaApp, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath, ReposRoot: root,
	})
	require.NoError(t, err)
	server = httptest.NewServer(viaApp)
	t.Cleanup(func() { _ = log.Close() })

	tc := vt.NewClient(t, server, "/board")
	require.Equal(t, 200, tc.Action((&BoardCard{}).CreateSession).
		WithSignal("newkey", "guarded").WithSignal("newrepo", "../../etc").Fire())

	cfg, _ := readLiveState("guarded")
	assert.Equal(t, filepath.Join(root, "etc"), cfg.RepoDir, "traversal collapses to a name under the root")
	assert.True(t, strings.HasPrefix(cfg.RepoDir, root), "resolution never escapes the repos root")
}

// A pick whose final segment is dot-only (".", "..", "a/..") has no real folder
// name — joining it under the root would either land ON the root or climb ABOVE
// it. Such a pick must be treated as blank, so the created session inherits the
// server's repo (RepoDir unchanged from the seeded default) instead of escaping.
func TestBoardCard_dotOnlyPickInheritsTheServerRepo(t *testing.T) {
	cases := []struct {
		name string
		pick string
	}{
		{"bare dotdot climbs above root", ".."},
		{"trailing dotdot climbs above root", "a/.."},
		{"bare dot lands on root", "."},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resetConsumersForTest()
			root := t.TempDir()
			defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
			var server *httptest.Server
			viaApp, log, err := NewServer(LiveConfig{
				RepoDir: "server-repo", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
				TestCmd: []string{"true"}, LedgerPath: defLogPath, ReposRoot: root,
			})
			require.NoError(t, err)
			server = httptest.NewServer(viaApp)
			t.Cleanup(func() { _ = log.Close() })

			vtc := vt.NewClient(t, server, "/board")
			require.Equal(t, 200, vtc.Action((&BoardCard{}).CreateSession).
				WithSignal("newkey", "dotpick").WithSignal("newrepo", tc.pick).Fire())

			cfg, _ := readLiveState("dotpick")
			assert.Equal(t, "server-repo", cfg.RepoDir, "a dot-only pick inherits the server's repo, never escapes")
		})
	}
}

// A folder NAMED with dots that is not a traversal token (e.g. "....") is a real,
// legitimate directory name — the guard must reject only the dot-only traversal
// tokens "." and "..", never blank a valid name that merely contains dots.
func TestBoardCard_dottyButRealFolderNameResolvesUnderTheRoot(t *testing.T) {
	resetConsumersForTest()
	root := t.TempDir()
	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	viaApp, log, err := NewServer(LiveConfig{
		RepoDir: "server-repo", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath, ReposRoot: root,
	})
	require.NoError(t, err)
	server = httptest.NewServer(viaApp)
	t.Cleanup(func() { _ = log.Close() })

	tc := vt.NewClient(t, server, "/board")
	require.Equal(t, 200, tc.Action((&BoardCard{}).CreateSession).
		WithSignal("newkey", "dotty").WithSignal("newrepo", "....").Fire())

	cfg, _ := readLiveState("dotty")
	assert.Equal(t, filepath.Join(root, "...."), cfg.RepoDir, "a dotty-but-real name resolves under the root, not blanked")
}

// A session created from the board must be fundable immediately: the Lead earns
// bandwidth by answering review questions, but a prompt-first session has no
// anchored catch flow to earn from — so create seeds starting attention bandwidth,
// and the card renders the place-order control right away (no chicken-and-egg).
// NOT parallel (shared globals).
func TestBoardCard_createdSessionCanPlaceAPromptPacketImmediately(t *testing.T) {
	resetConsumersForTest()
	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	viaApp, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	})
	require.NoError(t, err)
	server = httptest.NewServer(viaApp)
	t.Cleanup(func() { _ = log.Close() })

	tc := vt.NewClient(t, server, "/board")
	require.Equal(t, 200, tc.Action((&BoardCard{}).CreateSession).
		WithSignal("newkey", "promptable").WithSignal("newrepo", ".").Fire())

	_, slog := readLiveState("promptable")
	require.NotNil(t, slog, "the created session is registered")
	bw, err := slog.Bandwidth()
	require.NoError(t, err)
	assert.Greater(t, bw, 0, "create seeds starting bandwidth so an order can be placed")

	card := bodyOf(vt.NewClient(t, server, "/?key=promptable").HTML())
	assert.Contains(t, card, "compose__place", "the place-order control renders immediately on the new session")
	// The new session is prompt-first — it inherits no anchor from the (anchored)
	// default, so it runs no catch-cycle and shows no phantom Oracle-running spinner.
	assert.NotContains(t, card, "Gate running", "a created session is prompt-first — no inherited catch-cycle")
}

// The typed repo dir must become the created session's repo — else a session
// pointed at a different tree silently works the server's repo instead. NOT
// parallel (shared globals).
func TestBoardCard_createSessionUsesTheTypedRepoDir(t *testing.T) {
	resetConsumersForTest()
	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	viaApp, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	})
	require.NoError(t, err)
	server = httptest.NewServer(viaApp)
	t.Cleanup(func() { _ = log.Close() })

	typed := freshGitRepo(t)
	tc := vt.NewClient(t, server, "/board")
	require.Equal(t, 200, tc.Action((&BoardCard{}).CreateSession).
		WithSignal("newkey", "typedrepo").WithSignal("newrepo", typed).Fire())

	cfg, slog := readLiveState("typedrepo")
	require.NotNil(t, slog, "the created session is registered")
	assert.Equal(t, typed, cfg.RepoDir, "the created session works the typed repo, not the server's")
}
