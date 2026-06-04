package settle

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A binary file a turn introduces pollutes the review diff and can't be
// line-reviewed or secret-scanned — so it must be SURFACED as an artifact for
// the reviewer. Crucially it is NOT blocked and NOT dropped: the revision is
// minted and the binary is committed. Surfacing, never silent exclusion.
func TestStagedBinaryFileIsSurfacedButStillCommitted(t *testing.T) {
	dir := initRepo(t)
	if err := os.WriteFile(filepath.Join(dir, "data.bin"), []byte{0, 1, 2, 3, 0, 4, 5, 6, 7}, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "note.txt"), []byte("just text\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := Settle(context.Background(), dir, "add binary + text")
	if err != nil {
		t.Fatalf("Settle: %v", err)
	}
	if !res.Committed {
		t.Fatalf("a binary must be surfaced, NOT block the commit; got Committed=false (Secrets=%+v)", res.Secrets)
	}

	found := false
	for _, a := range res.Artifacts {
		if a == "data.bin" {
			found = true
		}
		if a == "note.txt" {
			t.Errorf("a text file must not be flagged as an artifact, got %+v", res.Artifacts)
		}
	}
	if !found {
		t.Errorf("the binary data.bin must be surfaced in Artifacts, got %+v", res.Artifacts)
	}

	// The binary must still be committed — never silently dropped. cat-file -e
	// exits non-zero (failing via runGit) if it's not in the commit.
	runGit(t, dir, "cat-file", "-e", "HEAD:data.bin")
}

// When a secret blocks the commit, NOTHING was committed — so there are no
// artifacts to surface either. Artifacts must stay empty on the blocked path
// (it's only populated for a minted revision), even if the turn also staged a
// binary.
func TestSecretBlockedTurnSurfacesNoArtifacts(t *testing.T) {
	dir := initRepo(t)
	if err := os.WriteFile(filepath.Join(dir, "data.bin"), []byte{0, 1, 2, 0, 3}, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "conf.env"), []byte("API_KEY=\"ABCDEFGHIJKLMNOP1234\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := Settle(context.Background(), dir, "binary + secret")
	if err != nil {
		t.Fatalf("Settle: %v", err)
	}
	if res.Committed {
		t.Fatalf("the secret must block the commit")
	}
	if len(res.Secrets) == 0 {
		t.Fatalf("the secret must be surfaced")
	}
	if len(res.Artifacts) != 0 {
		t.Errorf("no commit means no artifacts surfaced, got %+v", res.Artifacts)
	}
}

// A binary already in history that this turn MODIFIES must also be surfaced —
// `git numstat` reports binary changes as `-\t-` whether the file is added or
// modified, so the surfacing must catch both.
func TestModifiedBinaryFileIsSurfaced(t *testing.T) {
	dir := initRepo(t)
	// A binary already committed to history.
	if err := os.WriteFile(filepath.Join(dir, "data.bin"), []byte{0, 1, 2, 3, 0, 4}, 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, dir, "add", "-A")
	runGit(t, dir, "commit", "-qm", "add binary")
	// This turn modifies it.
	if err := os.WriteFile(filepath.Join(dir, "data.bin"), []byte{0, 9, 8, 7, 0, 6, 5}, 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := Settle(context.Background(), dir, "modify binary")
	if err != nil {
		t.Fatalf("Settle: %v", err)
	}
	if !res.Committed {
		t.Fatalf("a modified binary must still commit")
	}
	found := false
	for _, a := range res.Artifacts {
		if a == "data.bin" {
			found = true
		}
	}
	if !found {
		t.Errorf("a modified binary must be surfaced as an artifact, got %+v", res.Artifacts)
	}
}

// A turn that DELETES a binary must not surface it as an artifact: Artifacts
// reports unreviewable binaries PRESENT in the minted revision (they pollute
// the diff, can't be line-reviewed). A removal is the opposite — the binary is
// leaving the tree, polluting nothing. `git numstat` reports a deleted binary
// as `-\t-\tpath` identically to an added/modified one, so surfacing must
// filter deletions out explicitly.
func TestDeletedBinaryFileIsNotSurfaced(t *testing.T) {
	dir := initRepo(t)
	if err := os.WriteFile(filepath.Join(dir, "data.bin"), []byte{0, 1, 2, 3, 0, 4}, 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, dir, "add", "-A")
	runGit(t, dir, "commit", "-qm", "add binary")
	// This turn deletes it (and adds a text file so there's something to commit).
	if err := os.Remove(filepath.Join(dir, "data.bin")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "note.txt"), []byte("bye\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := Settle(context.Background(), dir, "delete binary")
	if err != nil {
		t.Fatalf("Settle: %v", err)
	}
	if !res.Committed {
		t.Fatalf("the deletion must still commit")
	}
	if len(res.Artifacts) != 0 {
		t.Errorf("a deleted binary is not an artifact present in the revision, got %+v", res.Artifacts)
	}
}

// A staged binary whose path contains a space must surface with the path
// intact — numstat is tab-separated, so the space stays in the path field.
func TestBinaryPathWithSpaceIsSurfacedIntact(t *testing.T) {
	dir := initRepo(t)
	if err := os.WriteFile(filepath.Join(dir, "my data.bin"), []byte{0, 1, 2, 0, 3}, 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := Settle(context.Background(), dir, "add spaced binary")
	if err != nil {
		t.Fatalf("Settle: %v", err)
	}
	found := false
	for _, a := range res.Artifacts {
		if a == "my data.bin" {
			found = true
		}
	}
	if !found {
		t.Errorf("a spaced binary path must surface intact, got %+v", res.Artifacts)
	}
}

// A staged binary whose path contains non-ASCII bytes must surface with the
// path RAW (not git's core.quotePath "..."-with-C-escapes form), so the
// surfaced path is usable. `-z` numstat output guarantees raw, unquoted paths.
func TestBinaryPathNonASCIIIsSurfacedUnquoted(t *testing.T) {
	dir := initRepo(t)
	name := "café.bin"
	if err := os.WriteFile(filepath.Join(dir, name), []byte{0, 1, 2, 0, 3}, 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := Settle(context.Background(), dir, "add non-ascii binary")
	if err != nil {
		t.Fatalf("Settle: %v", err)
	}
	found := false
	for _, a := range res.Artifacts {
		if a == name {
			found = true
		}
		if strings.HasPrefix(a, "\"") {
			t.Errorf("path must be surfaced raw, not git-quoted, got %q", a)
		}
	}
	if !found {
		t.Errorf("non-ascii binary path must surface raw as %q, got %+v", name, res.Artifacts)
	}
}

// A turn touching only text files has no artifacts to surface — the scanner
// must not flag ordinary source.
func TestTextOnlyTurnSurfacesNoArtifacts(t *testing.T) {
	dir := initRepo(t)
	if err := os.WriteFile(filepath.Join(dir, "doc.md"), []byte("# title\n\nbody\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := Settle(context.Background(), dir, "add doc")
	if err != nil {
		t.Fatalf("Settle: %v", err)
	}
	if !res.Committed {
		t.Fatalf("a text change must commit")
	}
	if len(res.Artifacts) != 0 {
		t.Errorf("a text-only turn must surface no artifacts, got %+v", res.Artifacts)
	}
}
