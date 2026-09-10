package settle

import "testing"

// scanStagedPaths and its glob semantics are pure, and CONVENTIONS.md's
// public-API preference cannot exercise a non-"handshake/**" glob (the
// production deny globs are a fixed package-level var with no seam to inject
// a different set through Settle) — so this white-box test is the deliberate
// last resort for that one behavior.
func TestScanStagedPaths_directoryPrefixGlobMatchesNestedFilesNotNameCollisions(t *testing.T) {
	t.Parallel()
	diff := "diff --git a/handshake/foo.go b/handshake/foo.go\n" +
		"+++ b/handshake/foo.go\n" +
		"diff --git a/handshake-notes.md b/handshake-notes.md\n" +
		"+++ b/handshake-notes.md\n"

	hits := scanStagedPaths(diff, []string{"handshake/**"})

	if len(hits) != 1 || hits[0].File != "handshake/foo.go" {
		t.Fatalf("want exactly one hit on handshake/foo.go, got %+v", hits)
	}
}

func TestScanStagedPaths_fallsBackToPlainGlobSemanticsForANonDirectoryDenyGlob(t *testing.T) {
	t.Parallel()
	diff := "+++ b/secrets.env\n--- a/notes.txt\n"

	hits := scanStagedPaths(diff, []string{"*.env"})

	if len(hits) != 1 || hits[0].File != "secrets.env" {
		t.Fatalf("want exactly one hit on secrets.env, got %+v", hits)
	}
}
