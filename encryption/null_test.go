package encryption

import (
	"testing"
)

func TestNullEncryptionInit(t *testing.T) {
	t.Parallel()

	encryption := NullEncryption{}
	err := encryption.Init(map[string]string{})
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestNullEncryptionEncrypt(t *testing.T) {
	t.Parallel()

	encryption := NullEncryption{}
	plaintext := []byte("hello world")
	ciphertext, _, err := encryption.Encrypt(plaintext)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if string(ciphertext) != string(plaintext) {
		t.Errorf("expected ciphertext to be %v, got %v", plaintext, ciphertext)
	}
}

func TestNullEncryptionDecrypt(t *testing.T) {
	t.Parallel()

	encryption := NullEncryption{}
	ciphertext := []byte("hello world")
	plaintext, err := encryption.Decrypt(ciphertext, []byte{})
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if string(plaintext) != string(ciphertext) {
		t.Errorf("expected plaintext to be %v, got %v", ciphertext, plaintext)
	}
}
