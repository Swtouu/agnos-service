package crypto

import "testing"

func testCryptor(t *testing.T) *Cryptor {
	t.Helper()
	c, err := New([]byte("01234567890123456789012345678901"[:32]), []byte("test-hmac-secret"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return c
}

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	c := testCryptor(t)
	plaintext := "1234567890123"

	ciphertext, err := c.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	got, err := c.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if got != plaintext {
		t.Errorf("got %q, want %q", got, plaintext)
	}
}

func TestEncrypt_NonDeterministic(t *testing.T) {
	c := testCryptor(t)
	plaintext := "1234567890123"

	a, err := c.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	b, err := c.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if a == b {
		t.Error("expected different ciphertext for same plaintext (random nonce), got identical output")
	}
}

func TestDecrypt_InvalidInput(t *testing.T) {
	c := testCryptor(t)
	if _, err := c.Decrypt("not-valid-base64!!!"); err == nil {
		t.Error("expected error for invalid base64 input")
	}
	if _, err := c.Decrypt("dG9vc2hvcnQ="); err == nil {
		t.Error("expected error for ciphertext shorter than nonce size")
	}
}

func TestHash_Deterministic(t *testing.T) {
	c := testCryptor(t)
	value := "1234567890123"

	a := c.Hash(value)
	b := c.Hash(value)
	if a != b {
		t.Errorf("expected same hash for same input, got %q vs %q", a, b)
	}
}

func TestHash_DifferentInputsDifferentHashes(t *testing.T) {
	c := testCryptor(t)
	a := c.Hash("1234567890123")
	b := c.Hash("1234567890124")
	if a == b {
		t.Error("expected different hashes for different inputs")
	}
}

func TestNew_RejectsBadKeySize(t *testing.T) {
	if _, err := New([]byte("too-short"), []byte("secret")); err == nil {
		t.Error("expected error for non-32-byte encryption key")
	}
}

func TestNew_RejectsEmptyHMACSecret(t *testing.T) {
	if _, err := New([]byte("01234567890123456789012345678901"[:32]), nil); err == nil {
		t.Error("expected error for empty hmac secret")
	}
}
