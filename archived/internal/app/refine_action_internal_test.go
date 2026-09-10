package app

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/joaomdsg/packets/internal/ledger"
)

func TestBuildRefinement_turnsTheLeadsInputIntoTheRightSharpeningFact(t *testing.T) {
	t.Parallel()
	tgt := ledger.Target{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "pay.go", Line: 88}
	tests := []struct {
		name    string
		kind    string
		text    string
		wantOK  bool
		wantRec ledger.RefinedPacketRecord
	}{
		{
			name:   "criteria splits into one fact per non-blank line",
			kind:   "criteria",
			text:   "rejects a negative amount\n\n  caps at the daily ceiling  \n",
			wantOK: true,
			wantRec: ledger.RefinedPacketRecord{Target: tgt, Refine: "criteria",
				Criteria: []string{"rejects a negative amount", "caps at the daily ceiling"}},
		},
		{
			name:   "criteria with no non-blank line is not a refinement",
			kind:   "criteria",
			text:   "   \n\n",
			wantOK: false,
		},
		{
			name:   "convention carries the trimmed note",
			kind:   "convention",
			text:   "  wrap errors with an origin prefix  ",
			wantOK: true,
			wantRec: ledger.RefinedPacketRecord{Target: tgt, Refine: "convention",
				Note: "wrap errors with an origin prefix"},
		},
		{
			name:   "convention with empty text is not a refinement",
			kind:   "convention",
			text:   "   ",
			wantOK: false,
		},
		{
			name:   "split is built elsewhere (needs harvested sub-targets), not here",
			kind:   "split",
			text:   "anything",
			wantOK: false,
		},
		{
			name:   "an unknown kind is refused",
			kind:   "bogus",
			text:   "x",
			wantOK: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			rec, ok := buildRefinement(tgt, tt.kind, tt.text)
			assert.Equal(t, tt.wantOK, ok)
			if tt.wantOK {
				assert.Equal(t, tt.wantRec, rec)
			}
		})
	}
}
