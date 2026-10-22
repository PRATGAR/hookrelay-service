package guard

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

var keyPepper []byte

func UseKeyPepper(p string) {
	keyPepper = []byte(p)
}

func DigestKey(key string) string {
	mac := hmac.New(sha256.New, keyPepper)
	mac.Write([]byte(key))
	return hex.EncodeToString(mac.Sum(nil))
}
