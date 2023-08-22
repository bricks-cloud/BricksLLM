package encrypter

import (
	"crypto/sha256"
	"encoding/hex"
)

type Encrypter struct {
	key string
}

func NewEncrypter(key string) *Encrypter {
	return &Encrypter{
		key: key,
	}
}

func (e *Encrypter) Encrypt(secret string) string {
	h := sha256.New()
	h.Write([]byte(secret))
	sha := hex.EncodeToString(h.Sum(nil))
	return sha
}
