package store

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/zalando/go-keyring"
	"golang.org/x/crypto/pbkdf2"
)

const (
	keyringService     = "haya-tab"
	keyringUsername    = "encryption-master-key"
	encryptionV2Prefix = "v2:"
)

// getOrCreateMasterKey retrieves or creates a random master key stored in the OS keyring.
// This provides strong security by:
// - Using a cryptographically random 32-byte key
// - Storing the key in the OS-native credential manager (Keychain/Credential Manager/Secret Service)
// - Leveraging OS-level encryption and access controls
func getOrCreateMasterKey() ([]byte, error) {
	// Try to retrieve existing key from keyring
	keyStr, err := keyring.Get(keyringService, keyringUsername)
	if err == nil {
		// Key exists, decode and return it
		key, err := base64.StdEncoding.DecodeString(keyStr)
		if err != nil {
			return nil, fmt.Errorf("failed to decode master key: %w", err)
		}
		if len(key) != 32 {
			return nil, fmt.Errorf("invalid master key length: %d", len(key))
		}
		return key, nil
	}

	// Key doesn't exist or error occurred
	if err != keyring.ErrNotFound {
		// Real error occurred (not just "not found")
		return nil, fmt.Errorf("failed to access keyring: %w", err)
	}

	// Generate new random master key
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("failed to generate random key: %w", err)
	}

	// Store in keyring
	keyStr = base64.StdEncoding.EncodeToString(key)
	if err := keyring.Set(keyringService, keyringUsername, keyStr); err != nil {
		return nil, fmt.Errorf("failed to store key in keyring: %w", err)
	}

	return key, nil
}

// deriveEncryptionKeyLegacy generates a machine-specific encryption key using PBKDF2.
// This is the legacy method kept for backward compatibility with existing encrypted data.
// New encryptions should use getOrCreateMasterKey() instead.
func deriveEncryptionKeyLegacy() ([]byte, error) {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "haya-tab-default-host"
	}

	username := os.Getenv("USER")
	if username == "" {
		username = os.Getenv("USERNAME")
	}
	if username == "" {
		username = "haya-tab-default-user"
	}

	password := hostname + ":" + username
	salt := []byte("HAYA-TAB-SALT-V1-2026")
	key := pbkdf2.Key([]byte(password), salt, 100000, 32, sha256.New)

	return key, nil
}

// getEncryptionKey returns the cached encryption key
var encryptionKeyCache []byte
var legacyKeyCache []byte

func getEncryptionKey() ([]byte, error) {
	if encryptionKeyCache == nil {
		key, err := getOrCreateMasterKey()
		if err != nil {
			return nil, err
		}
		encryptionKeyCache = key
	}
	return encryptionKeyCache, nil
}

func getLegacyEncryptionKey() ([]byte, error) {
	if legacyKeyCache == nil {
		key, err := deriveEncryptionKeyLegacy()
		if err != nil {
			return nil, err
		}
		legacyKeyCache = key
	}
	return legacyKeyCache, nil
}

// Encrypt encrypts plaintext using AES-256-GCM with the master key from OS keyring.
// Returns base64-encoded ciphertext with "v2:" prefix to indicate the new encryption method.
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

	ciphertext := gcm.Seal(nonce, nonce, []byte(text), nil)
	encoded := base64.StdEncoding.EncodeToString(ciphertext)

	// Add version prefix to indicate new encryption method
	return encryptionV2Prefix + encoded, nil
}

// Decrypt decrypts ciphertext, automatically detecting the encryption version.
// Supports both v2 (keyring-based) and legacy (PBKDF2-based) encryption.
func Decrypt(cryptoText string) (string, error) {
	if cryptoText == "" {
		return "", nil
	}

	// Check if this is v2 encrypted data
	if strings.HasPrefix(cryptoText, encryptionV2Prefix) {
		// Remove prefix and decrypt with new method
		cryptoText = strings.TrimPrefix(cryptoText, encryptionV2Prefix)
		return decryptWithKey(cryptoText, getEncryptionKey)
	}

	// Legacy encrypted data - try legacy decryption first
	plaintext, err := decryptWithKey(cryptoText, getLegacyEncryptionKey)
	if err != nil {
		// If legacy decryption fails, try new method (in case prefix was missing)
		return decryptWithKey(cryptoText, getEncryptionKey)
	}
	return plaintext, nil
}

// decryptWithKey is a helper function that decrypts using the provided key getter
func decryptWithKey(cryptoText string, keyGetter func() ([]byte, error)) (string, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(cryptoText)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}

	key, err := keyGetter()
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
		return "", fmt.Errorf("decryption failed: %w", err)
	}

	return string(plaintext), nil
}
