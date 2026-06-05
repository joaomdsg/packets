package review_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/agntpr/internal/mutation"
	"github.com/joaomdsg/agntpr/internal/review"
)

func TestQuestionThreadsFromMutations_makesOpenQuestionThreadPerFinding(t *testing.T) {
	t.Parallel()

	findings := []mutation.Finding{{
		File:     "auth.go",
		Line:     42,
		Original: ">=",
		Mutated:  ">",
		Message:  "Mutation survived: changed `>=` to `>` on line 42 and all tests still passed — is line 42 actually constrained by a test?",
	}}

	threads := review.QuestionThreadsFromMutations(findings)

	require.Len(t, threads, 1)
	got := threads[0]
	assert.Equal(t, "auth.go", got.File)
	assert.Equal(t, 42, got.StartLine)
	assert.Equal(t, 42, got.EndLine)
	assert.Equal(t, "question", got.Tag)
	assert.Equal(t, "agntpr", got.Author)
	assert.Equal(t, review.Open, got.Status)
	assert.Equal(t, findings[0].Message, got.Body)
}

func TestQuestionThreadsFromMutations_preservesFindingOrder(t *testing.T) {
	t.Parallel()

	findings := []mutation.Finding{
		{File: "a.go", Line: 1, Message: "first"},
		{File: "a.go", Line: 9, Message: "second"},
	}

	threads := review.QuestionThreadsFromMutations(findings)

	require.Len(t, threads, 2)
	assert.Equal(t, "first", threads[0].Body)
	assert.Equal(t, "second", threads[1].Body)
}

func TestQuestionThreadsFromMutations_returnsNoThreadsForNoFindings(t *testing.T) {
	t.Parallel()

	assert.Empty(t, review.QuestionThreadsFromMutations(nil))
}

func TestThread_rendersAsConventionalComment(t *testing.T) {
	t.Parallel()

	th := review.Thread{Tag: "question", Body: "is line 42 constrained?", Status: review.Open}

	assert.Equal(t, "question: is line 42 constrained?", th.Render())
}
