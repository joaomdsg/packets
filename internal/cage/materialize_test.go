package cage_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/packets/internal/cage"
	"github.com/joaomdsg/packets/internal/ledger"
)

// hostRepoWithThreeRevs builds a real git repo with three distinct commits and
// returns the repo dir plus the base/fix/tip SHAs — the shape a claim's Target
// references. Offline, no network.
func hostRepoWithThreeRevs(t *testing.T) (dir, base, fix, tip string) {
	t.Helper()
	dir = t.TempDir()
	runGit(t, dir, "init", "-q")
	runGit(t, dir, "config", "user.email", "t@t")
	runGit(t, dir, "config", "user.name", "t")
	write(t, dir, "adult.go", "package adult\nfunc Adult(age int) bool { return age >= 18 }\n")
	base = commitAll(t, dir, "base")
	write(t, dir, "adult.go", "package adult\nfunc Adult(age int) bool { return age >= 21 }\n")
	fix = commitAll(t, dir, "fix")
	write(t, dir, "extra.go", "package adult\n")
	tip = commitAll(t, dir, "tip")
	return dir, base, fix, tip
}

func targetOf(base, fix, tip string) ledger.Target {
	return ledger.Target{BaseRev: base, FixRev: fix, TipRev: tip, Path: "adult.go", Line: 2}
}

// The cage runs the oracle over the claim's revisions, so the materialized repo
// must actually CONTAIN base, fix and tip as reachable objects — otherwise the
// in-cage checkout of any of them fails before a single test runs.
func TestMaterialize_repoContainsEveryTargetRevision(t *testing.T) {
	t.Parallel()
	host, base, fix, tip := hostRepoWithThreeRevs(t)

	wd, cleanup, err := cage.Materialize(context.Background(), host, targetOf(base, fix, tip))
	require.NoError(t, err)
	t.Cleanup(cleanup)

	for _, rev := range []string{base, fix, tip} {
		err := exec.Command("git", "-C", wd.Repo, "cat-file", "-e", rev+"^{commit}").Run()
		assert.NoErrorf(t, err, "the materialized repo must contain revision %s", rev)
	}
}

// The cage mounts a SINGLE writable /work dir, and the mutation oracle copies the
// repo working tree per mutant — so the go caches must be EMPTY scratch dirs that
// are siblings of the repo under Root, never inside the repo (or they'd be
// recopied and balloon, and TMPDIR would land on the small noexec tmpfs). Root is
// what gets bind-mounted; the in-container paths derive from it.
func TestMaterialize_laysOutCacheSiblingsBesideTheRepo(t *testing.T) {
	t.Parallel()
	host, base, fix, tip := hostRepoWithThreeRevs(t)

	wd, cleanup, err := cage.Materialize(context.Background(), host, targetOf(base, fix, tip))
	require.NoError(t, err)
	t.Cleanup(cleanup)

	// The repo lives under Root.
	assert.Equal(t, wd.Root, filepath.Dir(wd.Repo), "the repo must be a direct child of Root")

	for name, dir := range map[string]string{"GoCache": wd.GoCache, "GoTmp": wd.GoTmp, "GoPath": wd.GoPath, "Tmp": wd.Tmp} {
		info, statErr := os.Stat(dir)
		require.NoErrorf(t, statErr, "%s must exist", name)
		assert.Truef(t, info.IsDir(), "%s must be a directory", name)

		entries, readErr := os.ReadDir(dir)
		require.NoError(t, readErr)
		assert.Emptyf(t, entries, "%s must be an empty scratch dir", name)

		assert.Equalf(t, wd.Root, filepath.Dir(dir), "%s must be a direct child of Root (a sibling of the repo)", name)
		assert.Falsef(t, strings.HasPrefix(dir, wd.Repo+string(os.PathSeparator)),
			"%s must NOT live inside the repo — the oracle's per-mutant copy would recurse into it", name)

		// The launch points go's caches/HOME/TMPDIR here, so they must be writable.
		assert.NoErrorf(t, os.WriteFile(filepath.Join(dir, "probe"), []byte("x"), 0o644),
			"%s must be writable — the cage's go toolchain writes into it", name)
	}
}

// The host's real repo must never be the thing handed to the cage: materialize
// produces a SEPARATE Root, so nothing the (eventually untrusted) cage does can
// reach host state.
func TestMaterialize_isASeparateDirectoryFromTheHostRepo(t *testing.T) {
	t.Parallel()
	host, base, fix, tip := hostRepoWithThreeRevs(t)

	wd, cleanup, err := cage.Materialize(context.Background(), host, targetOf(base, fix, tip))
	require.NoError(t, err)
	t.Cleanup(cleanup)

	assert.NotEqual(t, host, wd.Root, "the materialized root must not be the host repo itself")
	rootAbs, err := filepath.Abs(wd.Root)
	require.NoError(t, err)
	hostAbs, err := filepath.Abs(host)
	require.NoError(t, err)
	assert.False(t, strings.HasPrefix(rootAbs, hostAbs+string(os.PathSeparator)),
		"the materialized root must not live inside the host repo tree")
}

// The whole point of option 1: the cage repo is WRITABLE and disposable, so the
// oracle's `git worktree add` (which writes .git/worktrees/<id>) succeeds —
// unlike a read-only host-repo mount, which would fail before any verification.
func TestMaterialize_repoIsWritableSoTheOracleCanAddWorktrees(t *testing.T) {
	t.Parallel()
	host, base, fix, tip := hostRepoWithThreeRevs(t)

	wd, cleanup, err := cage.Materialize(context.Background(), host, targetOf(base, fix, tip))
	require.NoError(t, err)
	t.Cleanup(cleanup)

	wt := filepath.Join(t.TempDir(), "wt")
	out, err := exec.Command("git", "-C", wd.Repo, "worktree", "add", "--detach", wt, fix).CombinedOutput()
	require.NoErrorf(t, err, "a disposable cage repo must permit `git worktree add`: %s", out)
}

// The host shares no inodes with the cage copy (`--no-hardlinks`): a disposable
// copy that hardlinked objects back into the host repo would let in-cage object
// corruption bleed into host state.
func TestMaterialize_doesNotHardlinkObjectsBackIntoTheHostRepo(t *testing.T) {
	t.Parallel()
	host, base, fix, tip := hostRepoWithThreeRevs(t)

	wd, cleanup, err := cage.Materialize(context.Background(), host, targetOf(base, fix, tip))
	require.NoError(t, err)
	t.Cleanup(cleanup)

	// Non-vacuous control: a plain `git clone --local` (WITHOUT --no-hardlinks)
	// of the same host hardlinks object files when the clone lands on the same
	// filesystem. If even that control shares no inode, hardlinks are impossible
	// here (cross-filesystem) and the assertion below would pass vacuously — so
	// skip rather than claim a guarantee the environment can't exercise.
	ctrl := filepath.Join(t.TempDir(), "hardlinked-control")
	out, err := exec.Command("git", "clone", "--local", "--quiet", host, ctrl).CombinedOutput()
	require.NoErrorf(t, err, "control clone failed: %s", out)
	if !sharesAnInode(t, filepath.Join(host, ".git", "objects"), filepath.Join(ctrl, ".git", "objects")) {
		t.Skip("hardlinks not possible on this filesystem; --no-hardlinks check is inconclusive")
	}

	assert.False(t, sharesAnInode(t, filepath.Join(host, ".git", "objects"), filepath.Join(wd.Repo, ".git", "objects")),
		"the cage copy must not hardlink objects back into the host repo (--no-hardlinks)")
}

// Cleanup reaps the whole Root — an unbounded farm of verification runs must not
// leak a workdir (repo + caches) per claim.
func TestMaterialize_cleanupRemovesTheWholeRoot(t *testing.T) {
	t.Parallel()
	host, base, fix, tip := hostRepoWithThreeRevs(t)

	wd, cleanup, err := cage.Materialize(context.Background(), host, targetOf(base, fix, tip))
	require.NoError(t, err)

	cleanup()
	_, statErr := os.Stat(wd.Root)
	assert.True(t, os.IsNotExist(statErr), "cleanup must remove the whole workdir root")
}

// Fail-closed: a Target whose revisions the host repo cannot resolve — a SHA it
// does not contain (forged/stale claim) or an empty rev (malformed claim) — is
// rejected, not silently materialized into a repo missing the commit the cage
// would then fail to check out.
func TestMaterialize_rejectsAnUnresolvableTargetRevision(t *testing.T) {
	t.Parallel()
	host, base, fix, _ := hostRepoWithThreeRevs(t)

	bad := []struct {
		name   string
		target ledger.Target
	}{
		{"absent SHA", targetOf(base, fix, "0000000000000000000000000000000000000000")},
		{"empty base rev", targetOf("", fix, fix)},
		{"empty fix rev", targetOf(base, "", base)},
		{"empty tip rev", targetOf(base, fix, "")},
	}
	for _, b := range bad {
		t.Run(b.name, func(t *testing.T) {
			t.Parallel()
			wd, cleanup, err := cage.Materialize(context.Background(), host, b.target)
			if cleanup != nil {
				t.Cleanup(cleanup)
			}
			require.Error(t, err, "an unresolvable Target (%s) must be refused", b.name)
			require.Nil(t, wd, "no workdir is returned on rejection")
		})
	}
}

// Fail-closed against argument injection: a forged claim could set a Target rev
// to a leading-dash value (e.g. "--git-dir=...") that git would otherwise parse
// as a flag instead of a revision. Such a rev names no commit in the host repo,
// so it must be REJECTED, never smuggled through verification as an option.
func TestMaterialize_rejectsAFlagLikeTargetRevision(t *testing.T) {
	t.Parallel()
	host, base, fix, _ := hostRepoWithThreeRevs(t)

	flagRevs := []string{"--git-dir=/tmp", "--help", "-q", "--output=/tmp/x"}
	for _, rev := range flagRevs {
		t.Run(rev, func(t *testing.T) {
			t.Parallel()
			wd, cleanup, err := cage.Materialize(context.Background(), host, targetOf(base, fix, rev))
			if cleanup != nil {
				t.Cleanup(cleanup)
			}
			require.Error(t, err, "a flag-like rev %q must be refused, not parsed as a git option", rev)
			require.Nil(t, wd, "no workdir is returned on rejection")
		})
	}
}

// A leading-dash hostRepo must not be parsed by `git clone` as an option: it is a
// repo path (or a non-existent one), so it must fail as a missing repo, never act
// as a flag.
func TestMaterialize_doesNotTreatAFlagLikeHostRepoAsAnOption(t *testing.T) {
	t.Parallel()
	_, base, fix, tip := hostRepoWithThreeRevs(t)

	// The validation step (rev-parse with cmd.Dir=hostRepo) fails first for a
	// bogus dir, which is enough to prove the flag-like value never reaches clone
	// as a parsed option; either way Materialize must error and leave nothing.
	wd, cleanup, err := cage.Materialize(context.Background(), "--upload-pack=touch /tmp/packets-pwned", targetOf(base, fix, tip))
	if cleanup != nil {
		t.Cleanup(cleanup)
	}
	require.Error(t, err, "a flag-like hostRepo must be refused")
	require.Nil(t, wd)
	_, statErr := os.Stat("/tmp/packets-pwned")
	require.True(t, os.IsNotExist(statErr), "no injected command may have run")
}

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	full := append([]string{"-C", dir}, args...)
	out, err := exec.Command("git", full...).CombinedOutput()
	require.NoErrorf(t, err, "git %v: %s", args, out)
	return strings.TrimSpace(string(out))
}

func write(t *testing.T, dir, name, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644))
}

func commitAll(t *testing.T, dir, msg string) string {
	t.Helper()
	runGit(t, dir, "add", "-A")
	runGit(t, dir, "commit", "-qm", msg)
	return runGit(t, dir, "rev-parse", "HEAD")
}

// sharesAnInode reports whether any regular file under a is hardlinked to a file
// under b (same device+inode via os.SameFile), the check that proves
// --no-hardlinks held: the cage copy must not alias host objects.
func sharesAnInode(t *testing.T, a, b string) bool {
	t.Helper()
	aFiles := regularFiles(t, a)
	shared := false
	require.NoError(t, filepath.Walk(b, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !info.Mode().IsRegular() {
			return err
		}
		for _, af := range aFiles {
			if os.SameFile(af, info) {
				shared = true
			}
		}
		return nil
	}))
	return shared
}

func regularFiles(t *testing.T, root string) []os.FileInfo {
	t.Helper()
	var infos []os.FileInfo
	require.NoError(t, filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !info.Mode().IsRegular() {
			return err
		}
		infos = append(infos, info)
		return nil
	}))
	return infos
}
