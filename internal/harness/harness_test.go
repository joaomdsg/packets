package harness_test

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/packets/internal/harness"
)

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "git %v\n%s", args, out)
	return strings.TrimSpace(string(out))
}

func initRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runGit(t, dir, "init", "-q")
	runGit(t, dir, "config", "user.email", "t@t")
	runGit(t, dir, "config", "user.name", "t")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "f.txt"), []byte("one\ntwo\nthree\n"), 0o644))
	runGit(t, dir, "add", "-A")
	runGit(t, dir, "commit", "-qm", "base")
	return dir
}

func write(t *testing.T, dir, name, content string) func() {
	t.Helper()
	return func() {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644))
	}
}

func thinking(text string) string {
	return `{"type":"assistant","message":{"content":[{"type":"text","text":"` + text + `"}]}}`
}

func editing(path string) string {
	return `{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Edit","input":{"file_path":"` + path + `"}}]}}`
}

func turnEnd() string { return `{"type":"result","subtype":"success"}` }

// frame is one harness stream-json line, with an optional side effect run the
// moment the supervisor reads it — modeling the live agent editing the working
// tree on disk and then narrating that edit in its event stream.
type frame struct {
	do   func()
	line string
}

// scriptedHarness is a thin test double of the harness subprocess's stdout: a
// genuine I/O boundary. It yields one line per frame; a frame's side effect
// runs before its bytes are returned, so an edit a frame stages is on disk
// before the supervisor reads any later line (e.g. the turn's result). Because
// the supervisor reads the next line only after settling the previous turn, a
// frame's edit lands strictly between settles — deterministic, no goroutine.
type scriptedHarness struct {
	frames []frame
	i      int
	buf    []byte
}

func script(frames ...frame) *scriptedHarness { return &scriptedHarness{frames: frames} }

func (s *scriptedHarness) Read(p []byte) (int, error) {
	if len(s.buf) == 0 {
		if s.i >= len(s.frames) {
			return 0, io.EOF
		}
		f := s.frames[s.i]
		s.i++
		if f.do != nil {
			f.do()
		}
		s.buf = []byte(f.line + "\n")
	}
	n := copy(p, s.buf)
	s.buf = s.buf[n:]
	return n, nil
}

func hasEditing(turn harness.Turn, path string) bool {
	for _, e := range turn.Events {
		if e.Type == "activity.agent" && e.Kind == "editing" && e.Detail == path {
			return true
		}
	}
	return false
}

// Each completed agent turn must become its own reviewable revision, and a
// later turn must be diffed against the revision the previous turn minted —
// not the original base — so the review surface shows only what that turn
// changed.
func TestSupervisor_settlesARevisionAtEachTurnBoundaryThreadingTheBaseForward(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	base := runGit(t, dir, "rev-parse", "HEAD")

	stream := script(
		frame{line: thinking("editing the second line")},
		frame{do: write(t, dir, "f.txt", "one\nTWO\nthree\n"), line: editing("f.txt")},
		frame{line: turnEnd()},
		frame{line: thinking("now the third line")},
		frame{do: write(t, dir, "f.txt", "one\nTWO\nTHREE\n"), line: editing("f.txt")},
		frame{line: turnEnd()},
	)

	turns, err := harness.New(dir, base).Run(context.Background(), stream)
	require.NoError(t, err)
	require.Len(t, turns, 2, "two turn boundaries must settle two turns")

	require.True(t, turns[0].Outcome.Minted, "the first turn changed a file → a revision")
	require.True(t, turns[1].Outcome.Minted, "the second turn changed a file → a revision")
	assert.NotEqual(t, turns[0].Outcome.SHA, turns[1].Outcome.SHA, "each turn mints a distinct revision")

	// Diffed against turn 1's SHA, only line 3 changed (1/1). Against the
	// original base it would span both lines (2/2) — the discriminator.
	assert.Equal(t, 1, turns[1].Outcome.Added, "turn 2 must diff against turn 1's revision, not the base")
	assert.Equal(t, 1, turns[1].Outcome.Deleted, "turn 2 must diff against turn 1's revision, not the base")

	assert.True(t, hasEditing(turns[0], "f.txt"), "turn 1 must surface its editing activity")
	assert.True(t, hasEditing(turns[1], "f.txt"), "turn 2 must surface its editing activity")
	assert.Len(t, turns[0].Events, 2, "turn 1 accumulates both its thinking and editing activity")
}

// A secret in a turn's edit must block the revision (mint nothing) yet surface
// on the outcome, so the harness can never write a secret into review history
// but the user still sees why the turn produced nothing.
func TestSupervisor_surfacesSecretsAndMintsNothingForASecretBearingTurn(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	base := runGit(t, dir, "rev-parse", "HEAD")

	stream := script(
		frame{do: write(t, dir, "conf.env", "API_KEY=\"ABCDEFGHIJKLMNOP1234\"\n"), line: editing("conf.env")},
		frame{line: turnEnd()},
	)

	turns, err := harness.New(dir, base).Run(context.Background(), stream)
	require.NoError(t, err, "a blocked secret is surfaced, not an error")
	require.Len(t, turns, 1)
	assert.False(t, turns[0].Outcome.Minted, "a secret-bearing turn mints no revision")
	assert.NotEmpty(t, turns[0].Outcome.Secrets, "the secret is surfaced on the outcome")
	assert.Equal(t, base, runGit(t, dir, "rev-parse", "HEAD"), "HEAD must not move when a secret is blocked")
}

// A turn that edits nothing must not mint a revision (turn = revision only when
// real work happened), yet its live activity is still surfaced for the user to
// watch.
func TestSupervisor_mintsNothingForANoEditTurnButStillSurfacesItsActivity(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	base := runGit(t, dir, "rev-parse", "HEAD")

	stream := strings.NewReader(thinking("nothing to change here") + "\n" + turnEnd() + "\n")

	turns, err := harness.New(dir, base).Run(context.Background(), stream)
	require.NoError(t, err)
	require.Len(t, turns, 1, "the one turn boundary yields one turn")
	assert.False(t, turns[0].Outcome.Minted, "a no-edit turn mints no revision")
	assert.Empty(t, turns[0].Outcome.SHA, "no revision means no SHA")
	assert.Equal(t, base, runGit(t, dir, "rev-parse", "HEAD"), "HEAD must not move on a no-edit turn")

	require.Len(t, turns[0].Events, 1, "the thinking activity is surfaced")
	assert.Equal(t, "thinking", turns[0].Events[0].Kind, "the surfaced activity is the agent's thinking")
}

// The turn-ended signal is a boundary marker, not work the user reviews — it
// must not pollute the turn's activity log.
func TestSupervisor_excludesTheTurnEndSignalFromTheActivityLog(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	base := runGit(t, dir, "rev-parse", "HEAD")

	stream := strings.NewReader(thinking("done") + "\n" + turnEnd() + "\n")

	turns, err := harness.New(dir, base).Run(context.Background(), stream)
	require.NoError(t, err)
	require.Len(t, turns, 1)
	for _, e := range turns[0].Events {
		assert.NotEqual(t, "turn.ended", e.Type, "the boundary signal is not an activity event")
	}
}

// A truncated stream (the harness was killed mid-turn) must not mint a
// half-baked revision: only a completed turn — one the agent signalled done —
// is a revision.
func TestSupervisor_doesNotSettleAnIncompleteTrailingTurn(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	base := runGit(t, dir, "rev-parse", "HEAD")

	stream := script(
		frame{line: thinking("starting")},
		frame{do: write(t, dir, "f.txt", "one\nTWO\nthree\n"), line: editing("f.txt")},
		// no turn-end: the stream ends mid-turn.
	)

	turns, err := harness.New(dir, base).Run(context.Background(), stream)
	require.NoError(t, err)
	assert.Empty(t, turns, "an incomplete trailing turn settles nothing")
	assert.Equal(t, base, runGit(t, dir, "rev-parse", "HEAD"), "HEAD must not move for an unsettled turn")
}

// A corrupt stream line must surface as an error, never be silently skipped —
// a phantom or dropped revision would corrupt the review record.
func TestSupervisor_returnsErrorOnAMalformedStreamLine(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	base := runGit(t, dir, "rev-parse", "HEAD")

	stream := strings.NewReader("this is not json\n")

	_, err := harness.New(dir, base).Run(context.Background(), stream)
	assert.Error(t, err, "a malformed harness event must error")
}

// A single assistant message can be large (well past bufio.Scanner's default
// 64KB token limit); the stream reader must not choke on it.
func TestSupervisor_handlesAStreamLineLargerThanTheDefaultBuffer(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	base := runGit(t, dir, "rev-parse", "HEAD")

	big := strings.Repeat("a", 70*1024)
	stream := strings.NewReader(thinking(big) + "\n" + turnEnd() + "\n")

	turns, err := harness.New(dir, base).Run(context.Background(), stream)
	require.NoError(t, err, "a long event line must not exceed the reader's buffer")
	require.Len(t, turns, 1)
	require.Len(t, turns[0].Events, 1)
	assert.Equal(t, big, turns[0].Events[0].Detail, "the full large message is surfaced")
}

type errReader struct{ err error }

func (e errReader) Read(p []byte) (int, error) { return 0, e.err }

// A read failure on the stream (e.g. a broken pipe from a dead harness
// subprocess) must surface as an error, never be mistaken for a clean end.
func TestSupervisor_surfacesAStreamReadFailure(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	base := runGit(t, dir, "rev-parse", "HEAD")

	_, err := harness.New(dir, base).Run(context.Background(), errReader{err: io.ErrClosedPipe})
	assert.ErrorIs(t, err, io.ErrClosedPipe, "a stream read failure must propagate")
}

// A failure to settle a turn (here, a diff against an unresolvable base) must
// abort the run, never yield a phantom turn.
func TestSupervisor_surfacesASettleFailure(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)

	stream := script(
		frame{do: write(t, dir, "f.txt", "one\nTWO\nthree\n"), line: editing("f.txt")},
		frame{line: turnEnd()},
	)

	_, err := harness.New(dir, "nonexistent-base-rev").Run(context.Background(), stream)
	assert.Error(t, err, "a settle/diff failure must abort the run")
}

// Blank lines in the stream are framing noise, not events — they must be
// skipped without erroring (a malformed-JSON parse would otherwise fail).
func TestSupervisor_skipsBlankStreamLines(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	base := runGit(t, dir, "rev-parse", "HEAD")

	stream := strings.NewReader("\n" + thinking("ok") + "\n\n" + turnEnd() + "\n")

	turns, err := harness.New(dir, base).Run(context.Background(), stream)
	require.NoError(t, err)
	require.Len(t, turns, 1, "blank lines must not break turn detection")
}
