package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsePeerSpec_buildsAGrantConfinedToTheSessionKey(t *testing.T) {
	g, err := parsePeerSpec("rate-limiter:prodA:s3cr3t")
	require.NoError(t, err)
	assert.Equal(t, "rate-limiter", g.Session, "the session key is the peer identity (peer == session key)")
	assert.Equal(t, "prodA", g.User)
	assert.Equal(t, "s3cr3t", g.Pass)
	assert.NotEmpty(t, g.Instance, "the grant binds to the economy instance so its claims are publishable AND consumable")
}

func TestParsePeerSpec_keepsAPasswordContainingColons(t *testing.T) {
	g, err := parsePeerSpec("k:user:a:b:c")
	require.NoError(t, err)
	assert.Equal(t, "user", g.User)
	assert.Equal(t, "a:b:c", g.Pass, "only the first two colons split key/user; the password keeps its colons")
}

func TestParsePeerSpec_rejectsAMissingField(t *testing.T) {
	for _, spec := range []string{"justkey", "key:user", "key::pass", ":user:pass", "key:user:"} {
		_, err := parsePeerSpec(spec)
		assert.Errorf(t, err, "a peer spec missing key/user/pass must fail fast: %q", spec)
	}
}
