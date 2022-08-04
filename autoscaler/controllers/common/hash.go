package common

import (
	"crypto/sha256"
	"encoding/base64"
)

func Sha256Hash(value string) string {
	h := sha256.New()
	h.Write([]byte(value))
	b := h.Sum(nil)
	return base64.StdEncoding.EncodeToString(b)
}
