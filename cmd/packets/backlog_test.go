package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/packets/internal/ledger"
)

func TestBacklogSpecSeedsAFundableWorkOrderTarget(t *testing.T) {
	t.Parallel()
	// The whole point of -backlog: a spec on the CLI becomes a fundable work-order
	// target so the dispatch→fund→fill→review loop is runnable without test fixtures.
	tgt, err := parseBacklogSpec("base=aaa,fix=bbb,file=pkg/x.go,line=12,tip=ccc")
	require.NoError(t, err)
	assert.Equal(t, ledger.Target{
		BaseRev: "aaa",
		FixRev:  "bbb",
		TipRev:  "ccc",
		Path:    "pkg/x.go",
		Line:    12,
	}, tgt)
}

func TestBacklogSpecTipDefaultsToFixForACleanIntegrationByConstruction(t *testing.T) {
	t.Parallel()
	// tip is optional: omitting it integrates onto the fix itself, mirroring sessions.
	tgt, err := parseBacklogSpec("base=aaa,fix=bbb,file=x.go,line=3")
	require.NoError(t, err)
	assert.Equal(t, "bbb", tgt.TipRev, "tip defaults to fix")
}

func TestBacklogSpecRejectsAMissingRequiredFieldSoNoHalfTargetIsFunded(t *testing.T) {
	t.Parallel()
	// A target missing base/fix/file/line would fund garbage — reject it, echoing the
	// spec so the operator can see which one was malformed.
	_, err := parseBacklogSpec("base=aaa,file=x.go,line=3")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "fix")
	assert.Contains(t, err.Error(), "base=aaa,file=x.go,line=3")
}

func TestBacklogSpecRejectsANonPositiveLineSoTheAnchorIsAlwaysReal(t *testing.T) {
	t.Parallel()
	// Line is 1-based; 0/negative/non-numeric can't anchor a real mutant.
	for _, spec := range []string{
		"base=a,fix=b,file=x.go,line=0",
		"base=a,fix=b,file=x.go,line=-2",
		"base=a,fix=b,file=x.go,line=oops",
	} {
		_, err := parseBacklogSpec(spec)
		require.Error(t, err, spec)
		assert.Contains(t, err.Error(), "line", spec)
	}
}

func TestBacklogSpecTrimsWhitespaceSoCopyPastedSpecsStillFund(t *testing.T) {
	t.Parallel()
	// Operators paste specs with stray spaces; trim them so the target still resolves
	// (mirrors parseSessionSpec).
	tgt, err := parseBacklogSpec("base= aaa , fix=bbb, file=x.go , line=5")
	require.NoError(t, err)
	assert.Equal(t, "aaa", tgt.BaseRev)
	assert.Equal(t, "x.go", tgt.Path)
	assert.Equal(t, 5, tgt.Line)
}

func TestBacklogSpecRejectsAMalformedPairSoTyposFailFast(t *testing.T) {
	t.Parallel()
	// A bare token (no '=') is an operator typo; fail at parse, not silently.
	_, err := parseBacklogSpec("base=a,fix=b,file=x.go,line=3,garbage")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "garbage")
}
