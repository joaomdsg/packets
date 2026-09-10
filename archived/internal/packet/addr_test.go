package packet_test

import (
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/joaomdsg/packets/internal/packet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddr_stringJoinsOwnerAndNameWithSlash(t *testing.T) {
	t.Parallel()

	a := packet.Addr{Owner: "acme", Name: "widgets"}

	assert.Equal(t, "acme/widgets", a.String())
}

func TestParseRemoteURL_extractsOwnerAndNameAcrossRemoteForms(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		url       string
		wantAddr  packet.Addr
		wantFound bool
	}{
		{
			name:      "ssh form with .git suffix",
			url:       "git@github.com:acme/widgets.git",
			wantAddr:  packet.Addr{Owner: "acme", Name: "widgets"},
			wantFound: true,
		},
		{
			name:      "ssh form without .git suffix",
			url:       "git@github.com:acme/widgets",
			wantAddr:  packet.Addr{Owner: "acme", Name: "widgets"},
			wantFound: true,
		},
		{
			name:      "https form with .git suffix",
			url:       "https://github.com/acme/widgets.git",
			wantAddr:  packet.Addr{Owner: "acme", Name: "widgets"},
			wantFound: true,
		},
		{
			name:      "https form without .git suffix",
			url:       "https://github.com/acme/widgets",
			wantAddr:  packet.Addr{Owner: "acme", Name: "widgets"},
			wantFound: true,
		},
		{
			name:      "https form with trailing slash",
			url:       "https://github.com/acme/widgets/",
			wantAddr:  packet.Addr{Owner: "acme", Name: "widgets"},
			wantFound: true,
		},
		{
			name:      "https form with trailing slash and .git suffix",
			url:       "https://github.com/acme/widgets.git/",
			wantAddr:  packet.Addr{Owner: "acme", Name: "widgets"},
			wantFound: true,
		},
		{
			name:      "nested group joins all but last segment as owner",
			url:       "https://gitlab.example.com/a/b/c.git",
			wantAddr:  packet.Addr{Owner: "a/b", Name: "c"},
			wantFound: true,
		},
		{
			name:      "port in host does not affect owner/name extraction",
			url:       "https://gitlab.example.com:8443/acme/widgets.git",
			wantAddr:  packet.Addr{Owner: "acme", Name: "widgets"},
			wantFound: true,
		},
		{
			name:      "http scheme (not just https)",
			url:       "http://internal-git/acme/widgets.git",
			wantAddr:  packet.Addr{Owner: "acme", Name: "widgets"},
			wantFound: true,
		},
		{
			name:      "empty string is not a remote",
			url:       "",
			wantAddr:  packet.Addr{},
			wantFound: false,
		},
		{
			name:      "garbage input is not a remote",
			url:       "not a url at all",
			wantAddr:  packet.Addr{},
			wantFound: false,
		},
		{
			name:      "https url with no path has nothing to extract",
			url:       "https://github.com",
			wantAddr:  packet.Addr{},
			wantFound: false,
		},
		{
			name:      "https url with a single path segment has no owner",
			url:       "https://github.com/widgets",
			wantAddr:  packet.Addr{},
			wantFound: false,
		},
		{
			name:      "https url with an empty host is not a remote",
			url:       "https:///acme/widgets.git",
			wantAddr:  packet.Addr{},
			wantFound: false,
		},
		{
			name:      "doubled slash produces an empty path segment",
			url:       "https://github.com/acme//widgets.git",
			wantAddr:  packet.Addr{},
			wantFound: false,
		},
		{
			name:      "at sign with no following colon is not scp-like ssh",
			url:       "git@github.com",
			wantAddr:  packet.Addr{},
			wantFound: false,
		},
		{
			name:      "colon before the at sign is not scp-like ssh",
			url:       "not-ssh:git@host/x",
			wantAddr:  packet.Addr{},
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotAddr, gotFound := packet.ParseRemoteURL(tt.url)

			assert.Equal(t, tt.wantFound, gotFound)
			assert.Equal(t, tt.wantAddr, gotAddr)
		})
	}
}

func TestParseAddr_derivesOwnerAndNameFromOriginRemote(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	requireGit(t, dir, "init")
	requireGit(t, dir, "remote", "add", "origin", "git@github.com:acme/widgets.git")

	got := packet.ParseAddr(dir)

	assert.Equal(t, packet.Addr{Owner: "acme", Name: "widgets"}, got)
}

func TestParseAddr_fallsBackToHonestLocalIdentityWhenNoOriginRemote(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	requireGit(t, dir, "init")

	got := packet.ParseAddr(dir)

	assert.Equal(t, packet.Addr{Owner: "local", Name: filepath.Base(dir)}, got)
}

func TestParseAddr_fallsBackToHonestLocalIdentityWhenOriginIsUnparseable(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	requireGit(t, dir, "init")
	requireGit(t, dir, "remote", "add", "origin", "https://github.com/onlyname")

	got := packet.ParseAddr(dir)

	assert.Equal(t, packet.Addr{Owner: "local", Name: filepath.Base(dir)}, got)
}

func TestParseAddr_fallsBackToHonestLocalIdentityWhenNotAGitRepoAtAll(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	got := packet.ParseAddr(dir)

	assert.Equal(t, packet.Addr{Owner: "local", Name: filepath.Base(dir)}, got)
}

// requireGit runs a git command against dir and fails the test immediately if
// it errors, so a broken test-fixture setup never masquerades as a ParseAddr bug.
func requireGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "git %v: %s", args, out)
}
