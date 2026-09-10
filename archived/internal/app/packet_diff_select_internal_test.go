package app

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/joaomdsg/packets/internal/diff"
	"github.com/joaomdsg/packets/internal/ledger"
)

// A live/prompt order carries no anchored path (Target.Path == "") — yet its fix
// still changed files. The review diff editor must open on one of THOSE changed
// files, not a blank pane: an anchorless order whose diff defaults to "" renders
// an empty editor, hiding the very edits the Lead came to inspect. Default to the
// first changed file so there is always something to see.
func TestSelectedFile_anchorlessPacketDefaultsToAChangedFile(t *testing.T) {
	swapTreeSeams(t, nil, nil,
		diff.Diff{Files: []diff.FileDiff{{Path: "internal/app/live.go"}, {Path: "internal/app/board.go"}}})
	tgt := ledger.Target{BaseRev: "b", FixRev: "f", Path: ""} // anchorless

	got := resolveSelectedFile(LiveConfig{RepoDir: "."}, tgt, "")

	assert.Equal(t, "internal/app/live.go", got,
		"an anchorless order opens on the first changed file, never a blank diff")
}

// An explicit ?file= pick (the Lead clicked a tree leaf) always wins — it is a
// direct navigation and must not be overridden by the anchor or a default.
func TestSelectedFile_anExplicitPickAlwaysWins(t *testing.T) {
	swapTreeSeams(t, nil, nil,
		diff.Diff{Files: []diff.FileDiff{{Path: "internal/app/live.go"}}})
	tgt := ledger.Target{BaseRev: "b", FixRev: "f", Path: "internal/app/anchored.go"}

	got := resolveSelectedFile(LiveConfig{RepoDir: "."}, tgt, "internal/app/picked.go")

	assert.Equal(t, "internal/app/picked.go", got, "an explicit file pick is honored verbatim")
}

// When the order IS anchored on a path, that anchor is the natural default (the
// reviewed line lives there) — the changed-file default only kicks in for the
// anchorless case, so an anchored order must not be redirected to some other file.
func TestSelectedFile_anchoredPacketDefaultsToItsAnchorNotAChangedFile(t *testing.T) {
	swapTreeSeams(t, nil, nil,
		diff.Diff{Files: []diff.FileDiff{{Path: "internal/app/other.go"}}})
	tgt := ledger.Target{BaseRev: "b", FixRev: "f", Path: "internal/app/anchored.go"}

	got := resolveSelectedFile(LiveConfig{RepoDir: "."}, tgt, "")

	assert.Equal(t, "internal/app/anchored.go", got,
		"an anchored order opens on its anchor, not a changed file")
}

// An anchorless order whose diff has NO changed files (nothing to show) degrades
// to an empty selection rather than inventing a path — the diff pane is honestly
// blank because there is genuinely nothing changed to inspect.
func TestSelectedFile_anchorlessWithNoChangesDegradesToEmpty(t *testing.T) {
	swapTreeSeams(t, nil, nil, diff.Diff{})
	tgt := ledger.Target{BaseRev: "b", FixRev: "f", Path: ""}

	got := resolveSelectedFile(LiveConfig{RepoDir: "."}, tgt, "")

	assert.Equal(t, "", got, "no anchor and no changed files → an honest empty selection")
}
