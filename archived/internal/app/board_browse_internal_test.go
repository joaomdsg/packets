package app

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-via/via/h"
	"github.com/go-via/via/vt"
)

// renderHTML renders a standalone h node to a string, so the pure view helpers can
// be asserted directly without driving the whole HTTP/SSE pipeline.
func renderHTML(t *testing.T, node h.H) string {
	t.Helper()
	if node == nil {
		return "" // a nil node renders nothing — the absent-panel case
	}
	var b strings.Builder
	require.NoError(t, node.Render(&b))
	return b.String()
}

// The directory browser lists ONLY child directories (a repo is a folder, never a
// loose file) and lists them sorted, so the Lead navigates a stable, predictable
// tree instead of OS-arbitrary order with files mixed in.
func TestBrowseSubdirsListsOnlyChildDirectoriesSorted(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(root, "beta"), 0o755))
	require.NoError(t, os.Mkdir(filepath.Join(root, "alpha"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "loose.txt"), []byte("x"), 0o644))

	assert.Equal(t, []string{"alpha", "beta"}, browseSubdirs(root),
		"only child dirs, sorted, files excluded")
}

// A missing or unreadable directory must browse as empty, never panic — the Lead
// can land on a stale/denied path and the panel just shows nothing to descend into.
func TestBrowseSubdirsOnMissingDirIsEmpty(t *testing.T) {
	assert.Empty(t, browseSubdirs(filepath.Join(t.TempDir(), "does-not-exist")),
		"a missing dir browses as empty, not a crash")
}

// Opening the browser must land on a USABLE absolute directory: the configured
// repos root is the natural home for board-created sessions, so it wins when set.
func TestBrowseStartPrefersReposRootWhenSet(t *testing.T) {
	root := t.TempDir()
	assert.Equal(t, root, browseStart(LiveConfig{ReposRoot: root, RepoDir: "/some/abs/repo"}),
		"the repos root is the natural start for board sessions")
}

// With no repos root, an absolute server repo dir is the next best start — the Lead
// browses near the tree the server already works, not some unrelated default.
func TestBrowseStartFallsBackToAbsoluteRepoDir(t *testing.T) {
	dir := t.TempDir()
	assert.Equal(t, dir, browseStart(LiveConfig{RepoDir: dir}),
		"an absolute repo dir starts the browse when there is no repos root")
}

// A relative repo dir (the default "." config) is not a meaningful filesystem
// anchor for a picker, so the start must still be an absolute path the Lead can
// navigate from — never a bare relative "." that the browser can't ascend out of.
func TestBrowseStartIsAlwaysAbsolute(t *testing.T) {
	got := browseStart(LiveConfig{RepoDir: "."})
	assert.True(t, filepath.IsAbs(got), "the browse start is always an absolute path")
	assert.NotEqual(t, ".", got, "a relative repo dir is never handed back verbatim as the start")
	if home, err := os.UserHomeDir(); err == nil {
		assert.Equal(t, home, got, "with no usable config anchor, browsing starts at the Lead's home dir")
	}
}

// Descending must move the picker INTO a clicked subdirectory, so the Lead can walk
// down to the repo they want — the whole point of a filesystem picker.
func TestNextBrowseDirMovesIntoARealDirectory(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "child")
	require.NoError(t, os.Mkdir(sub, 0o755))

	assert.Equal(t, sub, nextBrowseDir(root, sub),
		"a real directory becomes the new browse location")
}

// A target that is not a real directory (a loose file, a stale/forged path, or a
// blank) must be IGNORED, so a bad target can never strand the picker on a
// non-navigable location — it stays put on the current directory.
func TestNextBrowseDirRejectsNonDirectories(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "f.txt")
	require.NoError(t, os.WriteFile(file, []byte("x"), 0o644))

	assert.Equal(t, root, nextBrowseDir(root, file), "a loose file is rejected; the picker stays put")
	assert.Equal(t, root, nextBrowseDir(root, filepath.Join(root, "ghost")), "a missing path is rejected")
	assert.Equal(t, root, nextBrowseDir(root, ""), "a blank target is rejected")
}

// The stored browse location must be a CLEANED ABSOLUTE path — the entire reason for
// a server-side picker is to capture the real full path, never a name or a messy
// traversal that a later join could mis-resolve.
func TestNextBrowseDirStoresACleanAbsolutePath(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "child")
	require.NoError(t, os.Mkdir(sub, 0o755))
	messy := filepath.Join(root, "child", "x", "..") // resolves to sub, but with a .. segment

	got := nextBrowseDir(root, messy)
	assert.Equal(t, sub, got, "the stored path is cleaned of . / .. segments")
	assert.True(t, filepath.IsAbs(got), "and is absolute — a full path, never a bare name")
}

// A closed picker (no browse dir) must render NOTHING — the panel appears only after
// the Lead opens it, so the create form stays calm until asked.
func TestRepoBrowserIsAbsentWhenClosed(t *testing.T) {
	out := renderHTML(t, (&BoardCard{}).repoBrowser(""))
	assert.NotContains(t, out, "/_action/SelectRepo", "a closed picker renders no panel")
}

// The open panel must show the current directory, its child folders to descend into
// (each carrying its FULL absolute path so a click navigates by full path, never a
// name), and the select/close controls — without these the picker is unusable.
func TestRepoBrowserListsChildDirsWithFullPathsAndControls(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(root, "repo-a"), 0o755))

	out := renderHTML(t, (&BoardCard{}).repoBrowser(root))
	assert.Contains(t, out, root, "the panel shows the directory it is browsing")
	assert.Contains(t, out, "repo-a", "child directories are listed")
	assert.Contains(t, out, filepath.Join(root, "repo-a"),
		"each entry carries its full absolute path — the click navigates by full path")
	assert.Contains(t, out, "/_action/SelectRepo", "the panel offers select-this-folder")
	assert.Contains(t, out, "/_action/CloseBrowser", "the panel offers close")
	assert.Contains(t, out, "/_action/Browse", "child entries navigate via the Browse action")
}

// The up control must let the Lead ascend to the parent — except at the filesystem
// root, where there is nowhere higher to climb (so no dead up control is shown).
func TestRepoBrowserOffersUpExceptAtFilesystemRoot(t *testing.T) {
	root := t.TempDir()
	assert.Contains(t, renderHTML(t, (&BoardCard{}).repoBrowser(root)), "(up)",
		"a non-root directory can ascend to its parent")
	assert.NotContains(t, renderHTML(t, (&BoardCard{}).repoBrowser("/")), "(up)",
		"the filesystem root has no parent to ascend to")
}

// End-to-end wiring guard: opening the picker and clicking into a real subdirectory
// must, through the live action pipeline, re-render the panel showing that
// directory's contents — proving OpenBrowser, Browse, and the View are wired
// together (not just individually correct). NOT parallel (shared globals).
func TestOpeningAndDescendingShowsTheDescendedDirectoryLive(t *testing.T) {
	root := t.TempDir()
	repoA := filepath.Join(root, "repo-a")
	require.NoError(t, os.MkdirAll(filepath.Join(repoA, "nested-child"), 0o755))

	resetConsumersForTest()
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
	frames, cancel := tc.SSEReady()
	defer cancel()

	require.Equal(t, 200, tc.Action((&BoardCard{}).OpenBrowser).Fire())
	require.Equal(t, 200, tc.Action((&BoardCard{}).Browse).WithSignal("browsetarget", repoA).Fire())

	frame := vt.AwaitFrame(t, frames, 5*time.Second, "nested-child")
	assert.Contains(t, frame, repoA, "the live panel browses the descended directory by its full path")
}

// Selecting a browsed folder must store its FULL absolute path into the new-repo
// signal — the entire reason for a server-side picker (a browser file input could
// only ever yield a folder name). After open → descend → select, the newrepo signal
// patched to the client carries the chosen ABSOLUTE path, which CreateSession then
// uses verbatim via resolveRepoDir. NOT parallel (shared globals).
func TestSelectRepoStoresTheFullBrowsedPathOnTheForm(t *testing.T) {
	root := t.TempDir()
	repoA := filepath.Join(root, "repo-a")
	require.NoError(t, os.Mkdir(repoA, 0o755))

	resetConsumersForTest()
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
	frames, cancel := tc.SSEReady()
	defer cancel()

	require.Equal(t, 200, tc.Action((&BoardCard{}).OpenBrowser).Fire())
	require.Equal(t, 200, tc.Action((&BoardCard{}).Browse).WithSignal("browsetarget", repoA).Fire())
	require.Equal(t, 200, tc.Action((&BoardCard{}).SelectRepo).Fire())

	// The newrepo signal patch (distinct from the panel's HTML dir header) carries the
	// full chosen absolute path — proving SelectRepo stored a path, not a folder name.
	vt.AwaitFrame(t, frames, 5*time.Second, `signals {"newrepo":"`+repoA+`"}`)
}

// A symlink that points to a directory is a perfectly navigable folder — Leads
// commonly keep repos as symlinks into a workspace — so the picker must LIST it.
// (nextBrowseDir already follows symlinks when navigating, so excluding them from
// the listing would be an inconsistent dead-end: visible-nowhere yet enterable.)
func TestBrowseSubdirsIncludesSymlinkedDirectories(t *testing.T) {
	root := t.TempDir()
	realDir := t.TempDir()
	require.NoError(t, os.Symlink(realDir, filepath.Join(root, "linked-repo")))

	assert.Contains(t, browseSubdirs(root), "linked-repo",
		"a symlink to a directory is navigable, so it is listed")
}

// A symlink that points to a FILE (or dangles) is not a navigable folder, so it must
// be excluded — the listing stays "directories only" by the followed target's type,
// never by the link itself.
func TestBrowseSubdirsExcludesSymlinksToNonDirectories(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(t.TempDir(), "f.txt")
	require.NoError(t, os.WriteFile(file, []byte("x"), 0o644))
	require.NoError(t, os.Symlink(file, filepath.Join(root, "file-link")))
	require.NoError(t, os.Symlink(filepath.Join(t.TempDir(), "gone"), filepath.Join(root, "dangling")))

	got := browseSubdirs(root)
	assert.NotContains(t, got, "file-link", "a symlink to a file is not a navigable folder")
	assert.NotContains(t, got, "dangling", "a broken symlink resolves to nothing navigable")
}
