package common

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"os"
	"strings"
)

const (
	// encryptedKeyPrefix marks a key as encrypted (vs plaintext legacy).
	encryptedKeyPrefix = "enc:"
	// masterKeyEnv is the environment variable for the master encryption key.
	// Must be exactly 32 bytes (AES-256) when set.
	masterKeyEnv = "OCTO_MASTER_KEY"
)

// getMasterKey returns the master key from environment, or nil if not configured.
func getMasterKey() []byte {
	key := os.Getenv(masterKeyEnv)
	if key == "" {
		return nil
	}
	return []byte(key)
}

// encryptKey encrypts a plaintext key using AES-256-GCM with the master key.
// Returns base64-encoded ciphertext prefixed with "enc:".
// If no master key is configured, returns the plaintext unchanged.
func encryptKey(plaintext string) (string, error) {
	masterKey := getMasterKey()
	if masterKey == nil {
		return plaintext, nil
	}
	if len(masterKey) != 32 {
		return "", errors.New("OCTO_MASTER_KEY must be exactly 32 bytes")
	}

	block, err := aes.NewCipher(masterKey)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return encryptedKeyPrefix + base64.StdEncoding.EncodeToString(ciphertext), nil
}

// decryptKey decrypts an encrypted key string.
// If the key doesn't have the "enc:" prefix, it's treated as plaintext (legacy).
// If the key is encrypted but no master key is configured, returns an error.
func decryptKey(encrypted string) (string, error) {
	if !strings.HasPrefix(encrypted, encryptedKeyPrefix) {
		// Legacy plaintext key — return as-is
		return encrypted, nil
	}

	masterKey := getMasterKey()
	if masterKey == nil {
		return "", errors.New("encrypted key found but OCTO_MASTER_KEY is not set")
	}
	if len(masterKey) != 32 {
		return "", errors.New("OCTO_MASTER_KEY must be exactly 32 bytes")
	}

	data, err := base64.StdEncoding.DecodeString(encrypted[len(encryptedKeyPrefix):])
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(masterKey)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("ciphertext too short")
	}
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}
