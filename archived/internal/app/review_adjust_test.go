package app_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-via/via/vt"

	"github.com/joaomdsg/packets/internal/app"
)

// An empty adjustment comment is nothing to address — a silent no-op, never a
// dispatched turn. NOT parallel (shared globals).
func TestLiveCard_addAdjustmentIsANoOpOnEmptyComment(t *testing.T) {
	repo := initGitRepoForPacket(t)
	head := gitPacket(t, repo, "rev-parse", "HEAD")

	server, _ := bootDefaultServer(t, defaultBootCfg)
	log := addFundedSession(t, "adjnoop", app.LiveConfig{RepoDir: repo, BaseRev: head, Anchor: anchorForCap(), TestCmd: []string{"true"}})

	tc := vt.NewClient(t, server, "/review?key=adjnoop")
	require.Equal(t, 200, tc.Action((&app.ReviewCard{Key: "adjnoop"}).AddAdjustment).
		WithSignal("adjfile", "main.go").WithSignal("adjline", "3").WithSignal("adjtext", "   ").Fire())

	got := orderRecordFor(t, log, 1)
	assert.Equal(t, "", got.Target.Prompt, "an empty comment never dispatches a turn")
}

// Leaving an adjustment now also persists a DURABLE annotation — the anchored
// comment survives a restart and folds into the review rail, not just a
// fire-and-forget harness turn. It carries the file, line, the Lead's words, an
// author, and an id a reply can target.
func TestLiveCard_addAdjustmentPersistsADurableAnnotation(t *testing.T) {
	repo := initGitRepoForPacket(t)
	head := gitPacket(t, repo, "rev-parse", "HEAD")

	server, _ := bootDefaultServer(t, defaultBootCfg)
	log := addFundedSession(t, "adjann", app.LiveConfig{RepoDir: repo, BaseRev: head, Anchor: anchorForCap(), TestCmd: []string{"true"}})

	tc := vt.NewClient(t, server, "/review?key=adjann")
	require.Equal(t, 200, tc.Action((&app.ReviewCard{Key: "adjann"}).AddAdjustment).
		WithSignal("adjfile", "main.go").WithSignal("adjline", "3").WithSignal("adjtext", "guard the negative case").Fire())

	anns, err := log.Annotations()
	require.NoError(t, err)
	require.Len(t, anns, 1, "leaving an adjustment persists exactly one durable annotation")
	assert.Equal(t, "main.go", anns[0].File)
	assert.Equal(t, 3, anns[0].StartLine)
	assert.Equal(t, "guard the negative case", anns[0].Body)
	assert.Equal(t, "lead", anns[0].Author, "a Lead-authored comment records the human as its author")
	assert.NotEmpty(t, anns[0].ID, "it has an id so a reply can target it")
	assert.Empty(t, anns[0].ParentID, "a fresh adjustment is a top-level annotation, not a reply")
	// The session here is unfunded, so the harness re-trigger dispatch is refused —
	// yet the annotation persisted. The durable comment is recorded regardless of
	// whether the budget allows an agent turn right now.
}

// Two adjustments persist as two annotations with DISTINCT ids, so a reply can
// unambiguously target one — the ids are the thread's addressing. (The session
// is unfunded, so this also holds when the re-trigger dispatch is refused.)
func TestLiveCard_adjustmentsGetDistinctAnnotationIDs(t *testing.T) {
	repo := initGitRepoForPacket(t)
	head := gitPacket(t, repo, "rev-parse", "HEAD")

	server, _ := bootDefaultServer(t, defaultBootCfg)
	log := addFundedSession(t, "adjids", app.LiveConfig{RepoDir: repo, BaseRev: head, Anchor: anchorForCap(), TestCmd: []string{"true"}})

	fire := func(line, text string) {
		require.Equal(t, 200, vt.NewClient(t, server, "/review?key=adjids").
			Action((&app.ReviewCard{Key: "adjids"}).AddAdjustment).
			WithSignal("adjfile", "main.go").WithSignal("adjline", line).WithSignal("adjtext", text).Fire())
	}
	fire("3", "guard the negative case")
	fire("8", "and the overflow case")

	anns, err := log.Annotations()
	require.NoError(t, err)
	require.Len(t, anns, 2, "two adjustments persist two annotations")
	assert.NotEqual(t, anns[0].ID, anns[1].ID, "each annotation gets a distinct id a reply can target")
}

// Selecting a line RANGE in the diff (Monaco sets adjendline) persists an
// annotation anchored to the whole span, so a comment on a block reads as the
// block it covers — not just its first line.
func TestLiveCard_addAdjustmentPersistsALineRange(t *testing.T) {
	repo := initGitRepoForPacket(t)
	head := gitPacket(t, repo, "rev-parse", "HEAD")

	server, _ := bootDefaultServer(t, defaultBootCfg)
	log := addFundedSession(t, "adjrange", app.LiveConfig{RepoDir: repo, BaseRev: head, Anchor: anchorForCap(), TestCmd: []string{"true"}})

	tc := vt.NewClient(t, server, "/review?key=adjrange")
	require.Equal(t, 200, tc.Action((&app.ReviewCard{Key: "adjrange"}).AddAdjustment).
		WithSignal("adjfile", "main.go").WithSignal("adjline", "10").WithSignal("adjendline", "14").
		WithSignal("adjtext", "this whole block ignores the error").Fire())

	anns, err := log.Annotations()
	require.NoError(t, err)
	require.Len(t, anns, 1)
	assert.Equal(t, 10, anns[0].StartLine, "the range starts where the selection did")
	assert.Equal(t, 14, anns[0].EndLine, "and ends where it ended — the annotation covers the span")
}

// A single-line comment (no adjendline, or end == start) records no spurious
// range — EndLine stays 0 so it anchors as one line.
func TestLiveCard_addAdjustmentSingleLineHasNoRange(t *testing.T) {
	repo := initGitRepoForPacket(t)
	head := gitPacket(t, repo, "rev-parse", "HEAD")

	server, _ := bootDefaultServer(t, defaultBootCfg)
	log := addFundedSession(t, "adjsingle", app.LiveConfig{RepoDir: repo, BaseRev: head, Anchor: anchorForCap(), TestCmd: []string{"true"}})

	tc := vt.NewClient(t, server, "/review?key=adjsingle")
	require.Equal(t, 200, tc.Action((&app.ReviewCard{Key: "adjsingle"}).AddAdjustment).
		WithSignal("adjfile", "main.go").WithSignal("adjline", "3").WithSignal("adjtext", "one line").Fire())

	anns, err := log.Annotations()
	require.NoError(t, err)
	require.Len(t, anns, 1)
	assert.Equal(t, 3, anns[0].StartLine)
	assert.Equal(t, 0, anns[0].EndLine, "a single-line comment carries no range")
}

// A zero-length selection (end == start) collapses to a single line, not a
// bogus range — guards against a guardless "EndLine = whatever was sent".
func TestLiveCard_addAdjustmentEndEqualToStartIsNotARange(t *testing.T) {
	repo := initGitRepoForPacket(t)
	head := gitPacket(t, repo, "rev-parse", "HEAD")

	server, _ := bootDefaultServer(t, defaultBootCfg)
	log := addFundedSession(t, "adjeq", app.LiveConfig{RepoDir: repo, BaseRev: head, Anchor: anchorForCap(), TestCmd: []string{"true"}})

	tc := vt.NewClient(t, server, "/review?key=adjeq")
	require.Equal(t, 200, tc.Action((&app.ReviewCard{Key: "adjeq"}).AddAdjustment).
		WithSignal("adjfile", "main.go").WithSignal("adjline", "7").WithSignal("adjendline", "7").
		WithSignal("adjtext", "one line, selected as itself").Fire())

	anns, err := log.Annotations()
	require.NoError(t, err)
	require.Len(t, anns, 1)
	assert.Equal(t, 7, anns[0].StartLine)
	assert.Equal(t, 0, anns[0].EndLine, "end == start is a single line, not a zero-length range")
}

// An empty comment persists no annotation — nothing to record, same as it
// dispatches no turn.
func TestLiveCard_addAdjustmentPersistsNoAnnotationForAnEmptyComment(t *testing.T) {
	repo := initGitRepoForPacket(t)
	head := gitPacket(t, repo, "rev-parse", "HEAD")

	server, _ := bootDefaultServer(t, defaultBootCfg)
	log := addFundedSession(t, "adjannempty", app.LiveConfig{RepoDir: repo, BaseRev: head, Anchor: anchorForCap(), TestCmd: []string{"true"}})

	tc := vt.NewClient(t, server, "/review?key=adjannempty")
	require.Equal(t, 200, tc.Action((&app.ReviewCard{Key: "adjannempty"}).AddAdjustment).
		WithSignal("adjfile", "main.go").WithSignal("adjline", "3").WithSignal("adjtext", "   ").Fire())

	anns, err := log.Annotations()
	require.NoError(t, err)
	assert.Empty(t, anns, "an empty comment records no annotation")
}

// The review surface must render the adjustment entry point — inputs bound to the
// adjustment signals and a button wired to AddAdjustment — else the Lead has no way to
// leave an adjustment (the comment→harness round-trip would be unreachable from the
// UI). NOT parallel (shared globals).
func TestReviewCard_rendersTheAdjustmentEntryPoint(t *testing.T) {
	repo := initGitRepoForPacket(t)
	head := gitPacket(t, repo, "rev-parse", "HEAD")

	server, _ := bootDefaultServer(t, defaultBootCfg)
	_ = addFundedSession(t, "adjui", app.LiveConfig{RepoDir: repo, BaseRev: head, Anchor: anchorForCap(), TestCmd: []string{"true"}})

	body := bodyOf(vt.NewClient(t, server, "/review?key=adjui").HTML())
	assert.Contains(t, body, "/_action/AddAdjustment", "the review surface renders the leave-adjustment action")
	assert.Contains(t, body, `data-bind="adjtext"`, "with an input bound to the adjustment comment signal")
}
