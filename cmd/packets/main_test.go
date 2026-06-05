package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDistinctSessionsAreAcceptedAsIsolatedEconomies(t *testing.T) {
	t.Parallel()
	// The whole point of keyed sessions: distinct keys + distinct ledger paths
	// coexist as isolated economies. A clean set must pass validation untouched.
	err := validateSessions("catches.jsonl", []sessionRef{
		{key: "alpha", ledgerPath: "alpha.jsonl"},
		{key: "beta", ledgerPath: "beta.jsonl"},
	})
	require.NoError(t, err)
}

func TestDuplicateSessionKeyIsRejectedSoOneNeverClobbersAnotherInTheRegistry(t *testing.T) {
	t.Parallel()
	// Two -session specs with the same key would have the second registerSession
	// Store clobber the first entry in liveReg, orphaning the first ledger and
	// silently dropping a review target. Reject it instead of losing it.
	err := validateSessions("catches.jsonl", []sessionRef{
		{key: "dup", ledgerPath: "one.jsonl"},
		{key: "dup", ledgerPath: "two.jsonl"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "dup")
}

func TestSessionKeyDefaultIsReservedSoItCannotClobberThePrimaryCard(t *testing.T) {
	t.Parallel()
	// The primary "/" card registers under "default"; a -session key=default would
	// clobber it while main still holds and closes both ledgers. Reserve the key.
	err := validateSessions("catches.jsonl", []sessionRef{
		{key: "default", ledgerPath: "other.jsonl"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "default")
}

func TestTwoSessionsSharingALedgerPathAreRejectedToProtectTheJSONL(t *testing.T) {
	t.Parallel()
	// Two *os.File handles appending to one path interleave and corrupt the JSONL,
	// and fuse two economies that must stay isolated. Reject the collision.
	err := validateSessions("catches.jsonl", []sessionRef{
		{key: "alpha", ledgerPath: "shared.jsonl"},
		{key: "beta", ledgerPath: "shared.jsonl"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "shared.jsonl")
}

func TestSessionSharingThePrimaryLedgerPathIsRejected(t *testing.T) {
	t.Parallel()
	// A session whose ledger resolves to the default -ledger path would write the
	// same file as the primary card — same corruption + fused economy. Reject it.
	err := validateSessions("catches.jsonl", []sessionRef{
		{key: "alpha", ledgerPath: "catches.jsonl"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "catches.jsonl")
}

func TestLedgerPathsAreComparedAfterCleaningSoEquivalentSpellingsCollide(t *testing.T) {
	t.Parallel()
	// "./a.jsonl" and "a.jsonl" name the same file; raw string compare would miss
	// it and let two handles corrupt one log. Paths must be cleaned before compare.
	err := validateSessions("catches.jsonl", []sessionRef{
		{key: "alpha", ledgerPath: "./a.jsonl"},
		{key: "beta", ledgerPath: "a.jsonl"},
	})
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "a.jsonl"))
}
