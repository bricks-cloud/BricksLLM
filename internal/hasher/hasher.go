package hasher

import (
	"crypto/sha256"
	"encoding/hex"
)

func Hash(secret string) string {
	h := sha256.New()
	h.Write([]byte(secret))
	sha := hex.EncodeToString(h.Sum(nil))
	return sha
}
