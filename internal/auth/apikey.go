package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"errors"
	"strings"
)

const (
	KeyPrefix    = "vgl_"
	keyByteLen   = 32
	HashByteSize = sha256.Size
)

var (
	ErrEmptyKey = errors.New("empty api key")
	ErrBadKey   = errors.New("malformed api key")
)

var keyEncoding = base32.StdEncoding.WithPadding(base32.NoPadding)

// Generate returns a fresh API key (plaintext, show once to user) and its
// SHA-256 hash (store this). Plaintext = "vgl_" + base32(32 random bytes).
func Generate() (plaintext string, hash []byte, err error) {
	buf := make([]byte, keyByteLen)
	if _, err = rand.Read(buf); err != nil {
		return "", nil, err
	}
	plaintext = KeyPrefix + keyEncoding.EncodeToString(buf)
	hash = Hash(plaintext)
	return plaintext, hash, nil
}

func Hash(plaintext string) []byte {
	sum := sha256.Sum256([]byte(plaintext))
	return sum[:]
}

// Parse trims whitespace, validates prefix shape. Does not verify against DB.
func Parse(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", ErrEmptyKey
	}
	if !strings.HasPrefix(s, KeyPrefix) {
		return "", ErrBadKey
	}
	return s, nil
}
