package store

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"os"

	"golang.org/x/crypto/pbkdf2"
)

// deriveEncryptionKey generates a machine-specific encryption key using PBKDF2.
// This provides better security than a hardcoded key by deriving the key from:
// - Machine hostname (unique per machine)
// - Current username (unique per user)
// - A fixed salt (for consistency across app restarts)
//
// Security Note: This is NOT cryptographically secure against determined attackers
// who have access to the binary and the machine, but it's significantly better than
// a hardcoded key.
func deriveEncryptionKey() ([]byte, error) {
	// Get machine hostname
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "haya-tab-default-host"
	}

	// Get current username
	username := os.Getenv("USER")
	if username == "" {
		username = os.Getenv("USERNAME") // Windows
	}
	if username == "" {
		username = "haya-tab-default-user"
	}

	// Combine hostname and username as the password material
	password := hostname + ":" + username

	// Use a fixed salt for consistency (same machine/user always gets same key)
	// This allows decryption of previously encrypted data
	salt := []byte("HAYA-TAB-SALT-V1-2026")

	// Derive a 32-byte key using PBKDF2 with SHA-256
	// 100,000 iterations provides good security while remaining fast enough
	key := pbkdf2.Key([]byte(password), salt, 100000, 32, sha256.New)

	return key, nil
}

// getEncryptionKey returns the cached encryption key, deriving it on first call
var encryptionKeyCache []byte

func getEncryptionKey() ([]byte, error) {
	if encryptionKeyCache == nil {
		key, err := deriveEncryptionKey()
		if err != nil {
			return nil, err
		}
		encryptionKeyCache = key
	}
	return encryptionKeyCache, nil
}

func Encrypt(text string) (string, error) {
	if text == "" {
		return "", nil
	}

	key, err := getEncryptionKey()
	if err != nil {
		return "", err
	}

	c, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(gcm.Seal(nonce, nonce, []byte(text), nil)), nil
}

func Decrypt(cryptoText string) (string, error) {
	if cryptoText == "" {
		return "", nil
	}
	ciphertext, err := base64.StdEncoding.DecodeString(cryptoText)
	if err != nil {
		return "", err
	}

	key, err := getEncryptionKey()
	if err != nil {
		return "", err
	}

	c, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}
