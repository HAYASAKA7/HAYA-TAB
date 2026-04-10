package store

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"io"
	"strings"
	"testing"
)

func TestEncryptDecrypt(t *testing.T) {
	testCases := []string{
		"simple password",
		"password with spaces",
		"パスワード", // Japanese
		"密码",      // Chinese
		"пароль",    // Russian
		"!@#$%^&*()", // Special characters
		"a",          // Single character
		strings.Repeat("x", 1000), // Long string
	}

	for _, plaintext := range testCases {
		t.Run(plaintext[:min(len(plaintext), 20)], func(t *testing.T) {
			// Encrypt
			encrypted, err := Encrypt(plaintext)
			if err != nil {
				t.Fatalf("Encrypt() error = %v", err)
			}

			if encrypted == "" {
				t.Error("Encrypt() returned empty string")
			}

			if encrypted == plaintext {
				t.Error("Encrypted text should not equal plaintext")
			}

			// Check for v2 prefix
			if !strings.HasPrefix(encrypted, "v2:") {
				t.Error("Encrypted text should have v2: prefix")
			}

			// Decrypt
			decrypted, err := Decrypt(encrypted)
			if err != nil {
				t.Fatalf("Decrypt() error = %v", err)
			}

			if decrypted != plaintext {
				t.Errorf("Decrypt() = %v, want %v", decrypted, plaintext)
			}
		})
	}
}

func TestEncryptEmptyString(t *testing.T) {
	encrypted, err := Encrypt("")
	if err != nil {
		t.Fatalf("Encrypt('') error = %v", err)
	}

	if encrypted != "" {
		t.Errorf("Encrypt('') = %v, want empty string", encrypted)
	}
}

func TestDecryptEmptyString(t *testing.T) {
	decrypted, err := Decrypt("")
	if err != nil {
		t.Fatalf("Decrypt('') error = %v", err)
	}

	if decrypted != "" {
		t.Errorf("Decrypt('') = %v, want empty string", decrypted)
	}
}

func TestDecryptInvalidData(t *testing.T) {
	testCases := []struct {
		name  string
		input string
	}{
		{"invalid base64", "not-valid-base64!!!"},
		{"too short", "YWJj"}, // "abc" in base64, but too short for cipher
		{"random data", "SGVsbG8gV29ybGQ="}, // "Hello World" in base64
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Decrypt(tc.input)
			if err == nil {
				t.Error("Decrypt() should return error for invalid data")
			}
		})
	}
}

func TestEncryptDeterministic(t *testing.T) {
	plaintext := "test password"

	// Encrypt twice
	encrypted1, err := Encrypt(plaintext)
	if err != nil {
		t.Fatalf("First Encrypt() error = %v", err)
	}

	encrypted2, err := Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Second Encrypt() error = %v", err)
	}

	// Should be different due to random nonce
	if encrypted1 == encrypted2 {
		t.Error("Encrypt() should produce different ciphertext each time (due to random nonce)")
	}

	// But both should decrypt to the same plaintext
	decrypted1, _ := Decrypt(encrypted1)
	decrypted2, _ := Decrypt(encrypted2)

	if decrypted1 != plaintext || decrypted2 != plaintext {
		t.Error("Both encrypted values should decrypt to the same plaintext")
	}
}

func TestEncryptionKeyConsistency(t *testing.T) {
	// Encrypt a value
	plaintext := "test password"
	encrypted, err := Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	// Clear the key cache to force re-retrieval from keyring
	encryptionKeyCache = nil

	// Decrypt should still work (key should be retrieved from keyring)
	decrypted, err := Decrypt(encrypted)
	if err != nil {
		t.Fatalf("Decrypt() after cache clear error = %v", err)
	}

	if decrypted != plaintext {
		t.Errorf("Decrypt() after cache clear = %v, want %v", decrypted, plaintext)
	}
}

func TestLegacyDecryption(t *testing.T) {
	// Simulate legacy encrypted data (without v2: prefix)
	plaintext := "legacy password"

	// Get legacy key and encrypt manually
	key, err := deriveEncryptionKeyLegacy()
	if err != nil {
		t.Fatalf("deriveEncryptionKeyLegacy() error = %v", err)
	}

	// Manually encrypt using legacy method
	c, err := aes.NewCipher(key)
	if err != nil {
		t.Fatalf("aes.NewCipher() error = %v", err)
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		t.Fatalf("cipher.NewGCM() error = %v", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		t.Fatalf("rand.Reader error = %v", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	legacyEncrypted := base64.StdEncoding.EncodeToString(ciphertext)

	// Should be able to decrypt legacy data
	decrypted, err := Decrypt(legacyEncrypted)
	if err != nil {
		t.Fatalf("Decrypt(legacy) error = %v", err)
	}

	if decrypted != plaintext {
		t.Errorf("Decrypt(legacy) = %v, want %v", decrypted, plaintext)
	}
}

func TestEncryptionVersionPrefix(t *testing.T) {
	plaintext := "test password"
	encrypted, err := Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	if !strings.HasPrefix(encrypted, "v2:") {
		t.Errorf("Encrypted data should have v2: prefix, got: %s", encrypted[:min(len(encrypted), 10)])
	}
}

func TestDeriveEncryptionKey(t *testing.T) {
	key1, err := deriveEncryptionKeyLegacy()
	if err != nil {
		t.Fatalf("deriveEncryptionKeyLegacy() error = %v", err)
	}

	if len(key1) != 32 {
		t.Errorf("Key length = %d, want 32", len(key1))
	}

	// Derive again - should be the same
	key2, err := deriveEncryptionKeyLegacy()
	if err != nil {
		t.Fatalf("Second deriveEncryptionKeyLegacy() error = %v", err)
	}

	if string(key1) != string(key2) {
		t.Error("deriveEncryptionKeyLegacy() should produce consistent keys")
	}
}

func TestGetEncryptionKey(t *testing.T) {
	// Clear cache
	encryptionKeyCache = nil

	key1, err := getEncryptionKey()
	if err != nil {
		t.Fatalf("getEncryptionKey() error = %v", err)
	}

	// Second call should return cached key
	key2, err := getEncryptionKey()
	if err != nil {
		t.Fatalf("Second getEncryptionKey() error = %v", err)
	}

	if string(key1) != string(key2) {
		t.Error("getEncryptionKey() should return cached key")
	}
}

func TestEncryptLongString(t *testing.T) {
	// Test with a very long string
	plaintext := strings.Repeat("This is a long password with many characters. ", 100)

	encrypted, err := Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	decrypted, err := Decrypt(encrypted)
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}

	if decrypted != plaintext {
		t.Error("Failed to encrypt/decrypt long string")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}



