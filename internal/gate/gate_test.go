package gate_test

import (
	"errors"
	"testing"

	"github.com/joaomdsg/packets/internal/gate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubLLM is a hand-rolled stand-in for the claude -p process boundary,
// returning canned responses in call order.
type stubLLM struct {
	responses []string
	errs      []error
	calls     int
}

func (s *stubLLM) Review(prompt string) (string, error) {
	i := s.calls
	s.calls++
	var resp string
	var err error
	if i < len(s.responses) {
		resp = s.responses[i]
	}
	if i < len(s.errs) {
		err = s.errs[i]
	}
	return resp, err
}

func TestRun_parsesCleanJSON(t *testing.T) {
	t.Parallel()
	llm := &stubLLM{responses: []string{
		`{"warnings":[{"code":"AMBIGUOUS_GOAL","detail":"goal has no measure"}],"suggested_terminal":["npm test"]}`,
	}}

	result, err := gate.Run(llm, "goal: improve things\n", "package.json\n")

	require.NoError(t, err)
	require.Len(t, result.Warnings, 1)
	assert.Equal(t, "AMBIGUOUS_GOAL", result.Warnings[0].Code)
	assert.Equal(t, []string{"npm test"}, result.SuggestedTerminal)
}

func TestRun_extractsJSONWrappedInProseAndFences(t *testing.T) {
	t.Parallel()
	llm := &stubLLM{responses: []string{
		"Sure, here you go:\n```json\n" +
			`{"warnings":[],"suggested_terminal":[]}` +
			"\n```",
	}}

	result, err := gate.Run(llm, "goal: x\n", "")

	require.NoError(t, err)
	assert.Empty(t, result.Warnings)
}

func TestRun_retriesOnceOnMalformedJSONThenSucceeds(t *testing.T) {
	t.Parallel()
	llm := &stubLLM{responses: []string{
		"not json at all",
		`{"warnings":[],"suggested_terminal":[]}`,
	}}

	result, err := gate.Run(llm, "goal: x\n", "")

	require.NoError(t, err)
	assert.Empty(t, result.Warnings)
	assert.Equal(t, 2, llm.calls)
}

func TestRun_errorsAfterSecondMalformedResponse(t *testing.T) {
	t.Parallel()
	llm := &stubLLM{responses: []string{"garbage", "still garbage"}}

	_, err := gate.Run(llm, "goal: x\n", "")

	assert.Error(t, err)
	assert.Equal(t, 2, llm.calls)
}

func TestRun_errorsAfterSecondProcessFailure(t *testing.T) {
	t.Parallel()
	llm := &stubLLM{errs: []error{errors.New("boom"), errors.New("boom again")}}

	_, err := gate.Run(llm, "goal: x\n", "")

	assert.Error(t, err)
	assert.Equal(t, 2, llm.calls)
}

func TestBuildPrompt_substitutesPacketAndRepoFiles(t *testing.T) {
	t.Parallel()

	prompt := gate.BuildPrompt("goal: fix login\n", "go.mod\nmain.go\n")

	assert.Contains(t, prompt, "goal: fix login")
	assert.Contains(t, prompt, "go.mod\nmain.go")
}
