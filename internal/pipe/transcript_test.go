package pipe_test

import (
	"encoding/json"
	"testing"

	"github.com/joaomdsg/packets/internal/catch"
	"github.com/joaomdsg/packets/internal/pipe"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Transcribe projects a CycleResult to the deterministic, verdict-relevant
// Transcript the host re-derives a catch from. It must carry every field the
// verdict depends on and DROP the Trace (its timestamps are non-deterministic
// and irrelevant to the verdict) so the same work yields byte-identical bytes.
func TestTranscribe_carriesEveryVerdictFieldButNotTheTrace(t *testing.T) {
	t.Parallel()
	cr := pipe.CycleResult{
		Outcome: catch.Catch,
		Reason:  pipe.ReasonNone,
		Path:    "adult.go",
		Line:    4,
		Land:    pipe.LandClean,
		Before:  catch.LineState{Inventory: []string{">=", "<"}, Survivors: []string{">="}},
		After:   catch.LineState{Inventory: []string{">=", "<"}, Survivors: nil},
		Trace:   []pipe.TraceEvent{{Kind: "catch", Msg: "minted"}},
	}

	tr := pipe.Transcribe(cr)

	assert.Equal(t, cr.Outcome, tr.Outcome)
	assert.Equal(t, cr.Reason, tr.Reason)
	assert.Equal(t, cr.Path, tr.Path)
	assert.Equal(t, cr.Line, tr.Line)
	assert.Equal(t, cr.Land, tr.Land)
	assert.Equal(t, cr.Before, tr.Before)
	assert.Equal(t, cr.After, tr.After)

	b1, err := json.Marshal(tr)
	require.NoError(t, err)
	b2, err := json.Marshal(tr)
	require.NoError(t, err)
	assert.Equal(t, b1, b2, "the transcript must serialize deterministically")
	assert.NotContains(t, string(b1), "Trace", "the non-deterministic trace must not be in the transcript")
	assert.NotContains(t, string(b1), "minted", "no trace content leaks into the transcript")
}

// The Transcript is the cross-boundary wire format the host decodes from a
// sandboxed verifier. Its keys must be coherent: the nested before/after
// LineStates serialize lowercase like every other Transcript field, so the
// format a later equivalence-lock pins is one consistent scheme, not a mix of
// lowercase outer keys and capitalized Go-default inner keys.
func TestTranscript_serializesEveryKeyLowercaseSoTheWireFormatIsCoherent(t *testing.T) {
	t.Parallel()
	tr := pipe.Transcribe(pipe.CycleResult{
		Before: catch.LineState{Inventory: []string{">="}, Survivors: []string{">="}},
		After:  catch.LineState{Inventory: []string{">="}, Survivors: nil},
	})

	b, err := json.Marshal(tr)
	require.NoError(t, err)
	s := string(b)

	assert.Contains(t, s, `"inventory"`, "the LineState inventory must serialize as a lowercase key")
	assert.Contains(t, s, `"survivors"`, "the LineState survivors must serialize as a lowercase key")
	assert.NotContains(t, s, `"Inventory"`, "no capitalized Go-default key leaks into the wire format")
	assert.NotContains(t, s, `"Survivors"`, "no capitalized Go-default key leaks into the wire format")
}
