package service

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
)

// ISTEncryption provides AES-256-GCM encryption for Intermediate Session Tokens.
type ISTEncryption struct {
	key []byte // 32 bytes for AES-256
}

// NewISTEncryption creates a new IST encryption instance.
// keyB64 must be a base64-encoded 32-byte (256-bit) key.
func NewISTEncryption(keyB64 string) (*ISTEncryption, error) {
	key, err := base64.StdEncoding.DecodeString(keyB64)
	if err != nil {
		return nil, fmt.Errorf("decode IST_ENCRYPTION_KEY: %w", err)
	}
	if len(key) != 32 {
		return nil, errors.New("IST_ENCRYPTION_KEY must be 32 bytes (256 bits) after base64 decode")
	}
	return &ISTEncryption{key: key}, nil
}

// Encrypt encrypts plaintext using AES-256-GCM.
// Returns base64-encoded ciphertext with prepended nonce (12 bytes).
func (e *ISTEncryption) Encrypt(plaintext []byte) (string, error) {
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", fmt.Errorf("new cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("new GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize()) // 12 bytes for GCM
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts ciphertext using AES-256-GCM.
// ciphertextB64 must be base64-encoded ciphertext with prepended nonce.
func (e *ISTEncryption) Decrypt(ciphertextB64 string) ([]byte, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(ciphertextB64)
	if err != nil {
		return nil, fmt.Errorf("decode ciphertext: %w", err)
	}

	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, fmt.Errorf("new cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("new GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}

	return plaintext, nil
}