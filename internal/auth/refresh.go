package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
)

// GenerateRefreshToken returns a high-entropy random token. Only its hash is
// ever stored (see HashRefreshToken) — the raw value goes to the client only.
func GenerateRefreshToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// HashRefreshToken is a plain SHA-256 digest (not HMAC): the refresh token is
// already high-entropy random data, so no secret key is needed to make the
// hash unguessable — unlike the blind-index use in internal/crypto, which
// hashes low-entropy PII and therefore needs an HMAC secret.
func HashRefreshToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
