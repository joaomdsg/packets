package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDistinctSessionKeysAreAcceptedAsIsolatedEconomies(t *testing.T) {
	t.Parallel()
	// The whole point of keyed sessions: distinct keys coexist as isolated
	// economies (each its own subtree of the shared fabric). A clean set passes.
	err := validateSessions([]sessionRef{
		{key: "alpha"},
		{key: "beta"},
	})
	require.NoError(t, err)
}

func TestDuplicateSessionKeyIsRejectedSoOneNeverClobbersAnotherInTheRegistry(t *testing.T) {
	t.Parallel()
	// Two -session specs with the same key would have the second registerSession
	// Store clobber the first entry in liveReg, orphaning the first review target
	// AND fusing two economies onto one session subtree. Reject it instead.
	err := validateSessions([]sessionRef{
		{key: "dup"},
		{key: "dup"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "dup")
}

func TestSessionKeyDefaultIsReservedSoItCannotClobberThePrimaryCard(t *testing.T) {
	t.Parallel()
	// The primary "/" card registers under "default"; a -session key=default would
	// clobber it while main still holds and closes both ledgers. Reserve the key.
	err := validateSessions([]sessionRef{
		{key: "default"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "default")
}
