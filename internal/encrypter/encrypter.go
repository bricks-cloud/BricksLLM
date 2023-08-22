package encrypter

import (
	"crypto/sha256"
	"encoding/hex"
)

type Encrypter struct{}

func NewEncrypter() *Encrypter {
	return &Encrypter{}
}

func (e *Encrypter) Encrypt(secret string) string {
	h := sha256.New()
	h.Write([]byte(secret))
	sha := hex.EncodeToString(h.Sum(nil))
	return sha
}
