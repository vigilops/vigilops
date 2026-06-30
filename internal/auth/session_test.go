package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateSession_returnsTokenAndHash(t *testing.T) {
	plaintext, hash, err := GenerateSession()
	require.NoError(t, err)
	assert.Greater(t, len(plaintext), 20, "token body must be substantial")
	assert.Len(t, hash, HashByteSize, "hash must be 32 bytes (SHA-256)")
	assert.Equal(t, hash, HashSession(plaintext))
}

func TestGenerateSession_producesUniqueTokens(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 10; i++ {
		plaintext, _, err := GenerateSession()
		require.NoError(t, err)
		assert.False(t, seen[plaintext], "duplicate session token")
		seen[plaintext] = true
	}
}

func TestHashSession_isDeterministic(t *testing.T) {
	assert.Equal(t, HashSession("foo"), HashSession("foo"))
	assert.NotEqual(t, HashSession("foo"), HashSession("bar"))
}
