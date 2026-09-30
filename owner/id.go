package owner

import (
	"crypto/rand"
	"encoding/hex"
)

func newID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic("owner: failed to generate random id: " + err.Error())
	}
	return hex.EncodeToString(b)
}
