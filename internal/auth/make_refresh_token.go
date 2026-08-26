package auth

import (
	"crypto/rand"
	"encoding/hex"
)

func MakeRefreshToken() string {
	key := make([]byte, 32)
	rand.Read(key)
	encoded_data := hex.EncodeToString(key)
	return encoded_data
}
