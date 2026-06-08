package auth

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerate_returnsPrefixedKeyAndHash(t *testing.T) {
	plaintext, hash, err := Generate()
	require.NoError(t, err)

	assert.True(t, strings.HasPrefix(plaintext, KeyPrefix), "plaintext must use vgl_ prefix")
	assert.Greater(t, len(plaintext), len(KeyPrefix)+20, "key body must be substantial")
	assert.Len(t, hash, HashByteSize, "hash must be 32 bytes (SHA-256)")
}

func TestGenerate_producesUniqueKeys(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 10; i++ {
		plaintext, _, err := Generate()
		require.NoError(t, err)
		assert.False(t, seen[plaintext], "duplicate plaintext from Generate")
		seen[plaintext] = true
	}
}

func TestHash_isDeterministicAndMatchesGenerate(t *testing.T) {
	plaintext, hash, err := Generate()
	require.NoError(t, err)
	assert.Equal(t, hash, Hash(plaintext))
	assert.Equal(t, Hash("foo"), Hash("foo"))
	assert.NotEqual(t, Hash("foo"), Hash("bar"))
}

func TestParse_trimsWhitespace(t *testing.T) {
	got, err := Parse("  vgl_abc  ")
	require.NoError(t, err)
	assert.Equal(t, "vgl_abc", got)
}

func TestParse_rejectsEmpty(t *testing.T) {
	_, err := Parse("   ")
	assert.ErrorIs(t, err, ErrEmptyKey)
}

func TestParse_rejectsMissingPrefix(t *testing.T) {
	_, err := Parse("not-a-vigil-key")
	assert.ErrorIs(t, err, ErrBadKey)
}
