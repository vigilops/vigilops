package auth

import (
	"crypto/rand"
	"crypto/sha256"
)

const sessionByteLen = 32

func GenerateSession() (plaintext string, hash []byte, err error) {
	buf := make([]byte, sessionByteLen)
	if _, err = rand.Read(buf); err != nil {
		return "", nil, err
	}
	plaintext = keyEncoding.EncodeToString(buf)
	hash = HashSession(plaintext)
	return plaintext, hash, nil
}

func HashSession(plaintext string) []byte {
	sum := sha256.Sum256([]byte(plaintext))
	return sum[:]
}
