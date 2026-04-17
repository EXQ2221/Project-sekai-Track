package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
)

func NewSID() (string, error) {
	return NewToken(16)
}

func NewToken(size int) (string, error) {
	if size <= 0 {
		size = 16
	}

	b := make([]byte, size)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	return hex.EncodeToString(b), nil
}

func HashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
