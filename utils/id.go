package utils

import (
	"crypto/rand"
	"encoding/hex"
)

func NewID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic("failed to generate random id: " + err.Error())
	}
	return hex.EncodeToString(b)
}
