package encryption

import (
	"testing"
)

func TestRsaKeyPairInit(t *testing.T) {
	t.Parallel()

	encryption := RsaKeyPair{}
	err := encryption.Init(map[string]string{
		"publicKeyPath":  "../keys/public.pem",
		"privateKeyPath": "../keys/private.pem",
	})
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if len(encryption.privateKey) == 0 {
		t.Errorf("expected private key to be set")
	}

	if len(encryption.publicKey) == 0 {
		t.Errorf("expected public key to be set")
	}
}

func TestRsaKeyPairEncrypt(t *testing.T) {
	t.Parallel()

	encryption := RsaKeyPair{}
	err := encryption.Init(map[string]string{
		"publicKeyPath":  "../keys/public.pem",
		"privateKeyPath": "../keys/private.pem",
	})
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	plaintext := []byte("hello world")
	ciphertext, dataKey, err := encryption.Encrypt(plaintext)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if len(ciphertext) == 0 {
		t.Errorf("expected ciphertext to be set")
	}

	if len(dataKey) == 0 {
		t.Errorf("expected data key to be set")
	}
}

func TestRsaKeyPairDecrypt(t *testing.T) {
	t.Parallel()

	encryption := RsaKeyPair{}
	err := encryption.Init(map[string]string{
		"publicKeyPath":  "../keys/public.pem",
		"privateKeyPath": "../keys/private.pem",
	})
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	plaintext := []byte("hello world")
	ciphertext, encryptedKey, err := encryption.Encrypt(plaintext)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	decryptedPlaintext, err := encryption.Decrypt(ciphertext, encryptedKey)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if string(decryptedPlaintext) != string(plaintext) {
		t.Errorf("expected plaintext to be %v, got %v", plaintext, decryptedPlaintext)
	}
}
