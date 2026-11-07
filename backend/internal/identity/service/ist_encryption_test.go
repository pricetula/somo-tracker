package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestISTEncryption_EncryptDecrypt(t *testing.T) {
	// Generate a valid 32-byte key (base64 encoded)
	keyB64 := "AAECAwQFBgcICQoLDA0ODxAREhMUFRYXGBkaGxwdHh8=" // 32 bytes

	enc, err := NewISTEncryption(keyB64)
	require.NoError(t, err)

	plaintext := []byte("test-intermediate-session-token-12345")
	ciphertext, err := enc.Encrypt(plaintext)
	require.NoError(t, err)
	require.NotEmpty(t, ciphertext)

	decrypted, err := enc.Decrypt(ciphertext)
	require.NoError(t, err)
	require.Equal(t, plaintext, decrypted)
}

func TestISTEncryption_EncryptProducesUniqueCiphertext(t *testing.T) {
	keyB64 := "AAECAwQFBgcICQoLDA0ODxAREhMUFRYXGBkaGxwdHh8="
	enc, err := NewISTEncryption(keyB64)
	require.NoError(t, err)

	plaintext := []byte("same-plaintext")
	c1, _ := enc.Encrypt(plaintext)
	c2, _ := enc.Encrypt(plaintext)

	// Each encryption should produce different ciphertext (different nonce)
	require.NotEqual(t, c1, c2)

	// But both should decrypt to same plaintext
	d1, _ := enc.Decrypt(c1)
	d2, _ := enc.Decrypt(c2)
	require.Equal(t, plaintext, d1)
	require.Equal(t, plaintext, d2)
}

func TestISTEncryption_WrongKeyFails(t *testing.T) {
	key1 := "AAECAwQFBgcICQoLDA0ODxAREhMUFRYXGBkaGxwdHh8="
	key2 := "BBECAwQFBgcICQoLDA0ODxAREhMUFRYXGBkaGxwdHh8="

	enc1, _ := NewISTEncryption(key1)
	enc2, _ := NewISTEncryption(key2)

	plaintext := []byte("test")
	ciphertext, _ := enc1.Encrypt(plaintext)

	// Decrypting with wrong key should fail
	_, err := enc2.Decrypt(ciphertext)
	require.Error(t, err)
}

func TestISTEncryption_InvalidKeyLength(t *testing.T) {
	// 16 bytes (128 bits) - too short
	shortKey := "AAECAwQFBgcICQoLDA0O"
	_, err := NewISTEncryption(shortKey)
	require.Error(t, err)
	require.Contains(t, err.Error(), "32 bytes")

	// 64 bytes - too long
	longKey := "AAECAwQFBgcICQoLDA0ODxAREhMUFRYXGBkaGxwdHh8gAAECAwQFBgcICQoLDA0ODxAREhMUFRYXGBkaGxwdHh8="
	_, err = NewISTEncryption(longKey)
	require.Error(t, err)
}

func TestISTEncryption_InvalidBase64(t *testing.T) {
	_, err := NewISTEncryption("not-valid-base64!@#")
	require.Error(t, err)
}

func TestISTEncryption_TamperedCiphertextFails(t *testing.T) {
	keyB64 := "AAECAwQFBgcICQoLDA0ODxAREhMUFRYXGBkaGxwdHh8="
	enc, _ := NewISTEncryption(keyB64)

	plaintext := []byte("test")
	ciphertext, _ := enc.Encrypt(plaintext)

	// Tamper with ciphertext (flip a bit in the base64)
	tampered := ciphertext[:len(ciphertext)-1] + "X"

	_, err := enc.Decrypt(tampered)
	require.Error(t, err)
}

func TestISTEncryption_EmptyPlaintext(t *testing.T) {
	keyB64 := "AAECAwQFBgcICQoLDA0ODxAREhMUFRYXGBkaGxwdHh8="
	enc, _ := NewISTEncryption(keyB64)

	ciphertext, err := enc.Encrypt([]byte(""))
	require.NoError(t, err)

	decrypted, err := enc.Decrypt(ciphertext)
	require.NoError(t, err)
	require.Empty(t, decrypted)
}

func TestISTEncryption_LargePlaintext(t *testing.T) {
	keyB64 := "AAECAwQFBgcICQoLDA0ODxAREhMUFRYXGBkaGxwdHh8="
	enc, _ := NewISTEncryption(keyB64)

	// 10KB plaintext
	plaintext := make([]byte, 10240)
	for i := range plaintext {
		plaintext[i] = byte(i % 256)
	}

	ciphertext, err := enc.Encrypt(plaintext)
	require.NoError(t, err)

	decrypted, err := enc.Decrypt(ciphertext)
	require.NoError(t, err)
	require.Equal(t, plaintext, decrypted)
}