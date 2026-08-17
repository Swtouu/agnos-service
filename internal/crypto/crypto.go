// Package crypto implements the blind-index pattern for searchable-but-encrypted PII:
// Encrypt/Decrypt (AES-GCM, random nonce) for storage+display, Hash (HMAC-SHA256,
// deterministic) for exact-match lookup. Never use Hash output for anything but
// equality search — it is not safe as a display value or a secret.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
)

type Cryptor struct {
	gcm        cipher.AEAD
	hmacSecret []byte
}

// New builds a Cryptor from a 32-byte AES-256 key and an HMAC secret. Both are
// expected as raw bytes decoded from env vars by the caller (see cmd/api and cmd/seed).
func New(encryptionKey, hmacSecret []byte) (*Cryptor, error) {
	if len(encryptionKey) != 32 {
		return nil, fmt.Errorf("encryption key must be 32 bytes for AES-256, got %d", len(encryptionKey))
	}
	if len(hmacSecret) == 0 {
		return nil, errors.New("hmac secret must not be empty")
	}

	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("init aes cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("init gcm: %w", err)
	}

	return &Cryptor{gcm: gcm, hmacSecret: hmacSecret}, nil
}

// Encrypt returns a base64 string containing a random nonce + ciphertext.
// Same plaintext encrypted twice produces different output (random nonce).
func (c *Cryptor) Encrypt(plaintext string) (string, error) {
	nonce := make([]byte, c.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}
	ciphertext := c.gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt reverses Encrypt.
func (c *Cryptor) Decrypt(encoded string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("decode base64: %w", err)
	}
	nonceSize := c.gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("ciphertext too short")
	}
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := c.gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt: %w", err)
	}
	return string(plaintext), nil
}

// Hash returns a deterministic HMAC-SHA256 hex digest for exact-match lookup
// (e.g. WHERE national_id_hash = crypto.Hash(input)). Same input always
// produces the same output; this is the whole point of a blind index.
func (c *Cryptor) Hash(value string) string {
	mac := hmac.New(sha256.New, c.hmacSecret)
	mac.Write([]byte(value))
	return hex.EncodeToString(mac.Sum(nil))
}
