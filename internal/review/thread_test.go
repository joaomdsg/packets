package review

import (
	"testing"

	"github.com/joaomdsg/agntpr/internal/mutation"
)

// A surviving mutant is worthless to a reviewer unless it shows up as an
// actionable, anchored review comment — an open question authored by the
// agent, pinned to the exact line the tests failed to constrain.
func TestSurvivingMutantBecomesAnOpenQuestionThreadForTheReviewer(t *testing.T) {
	findings := []mutation.Finding{{
		File:     "auth.go",
		Line:     42,
		Original: ">=",
		Mutated:  ">",
		Message:  "Mutation survived: changed `>=` to `>` on line 42 and all tests still passed — is line 42 actually constrained by a test?",
	}}

	threads := QuestionThreadsFromMutations(findings)

	if len(threads) != 1 {
		t.Fatalf("want exactly 1 thread, got %d", len(threads))
	}
	got := threads[0]
	if got.File != "auth.go" {
		t.Errorf("File = %q, want auth.go", got.File)
	}
	if got.StartLine != 42 || got.EndLine != 42 {
		t.Errorf("anchor = %d-%d, want 42-42 (single line)", got.StartLine, got.EndLine)
	}
	if got.Tag != "question" {
		t.Errorf("Tag = %q, want question", got.Tag)
	}
	if got.Author != "agntpr" {
		t.Errorf("Author = %q, want agntpr", got.Author)
	}
	if got.Status != Open {
		t.Errorf("Status = %q, want Open", got.Status)
	}
	if got.Body != findings[0].Message {
		t.Errorf("Body = %q, want the finding's message", got.Body)
	}
}

// Threads must mirror finding order so the reviewer reads them top-to-
// bottom in file order, the way they would scan a diff.
func TestThreadOrderMirrorsFindingOrder(t *testing.T) {
	findings := []mutation.Finding{
		{File: "a.go", Line: 1, Message: "first"},
		{File: "a.go", Line: 9, Message: "second"},
	}

	threads := QuestionThreadsFromMutations(findings)

	if len(threads) != 2 {
		t.Fatalf("want 2 threads, got %d", len(threads))
	}
	if threads[0].Body != "first" || threads[1].Body != "second" {
		t.Errorf("order not preserved: got %q then %q", threads[0].Body, threads[1].Body)
	}
}

// A strong suite yields no findings; that must produce a clean review,
// not a phantom thread.
func TestNoFindingsProduceNoThreads(t *testing.T) {
	if got := QuestionThreadsFromMutations(nil); len(got) != 0 {
		t.Fatalf("want no threads for no findings, got %d", len(got))
	}
}

// The reviewer sees a Conventional Comment, so the agent's intent is
// machine-readable as well as human-readable: "<tag>: <body>".
func TestThreadRendersAsAConventionalComment(t *testing.T) {
	th := Thread{Tag: "question", Body: "is line 42 constrained?", Status: Open}
	if got := th.Render(); got != "question: is line 42 constrained?" {
		t.Errorf("Render() = %q, want %q", got, "question: is line 42 constrained?")
	}
}
