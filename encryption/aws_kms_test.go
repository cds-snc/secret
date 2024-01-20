package encryption

import (
	"os"
	"testing"
)

func getKmsHost() string {
	host := "http://kms-local:8080"

	if h := os.Getenv("KMS_HOST"); h != "" {
		host = h
	}

	return host
}

func TestAwsKmsEncryptionInitMissingKeyId(t *testing.T) {
	t.Parallel()

	e := AwsKmsEncryption{}

	err := e.Init(map[string]string{
		"region": "ca-central-1",
	})

	if err == nil {
		t.Errorf("AwsKmsEncryption.Init() = %v, want %v", err, "error")
	}
}

func TestAwsKmsEncryptionInitMissingRegion(t *testing.T) {
	t.Parallel()

	e := AwsKmsEncryption{}

	err := e.Init(map[string]string{
		"kms_key_id": "test",
	})

	if err == nil {
		t.Errorf("AwsKmsEncryption.Init() = %v, want %v", err, "error")
	}
}

func TestAwsKmsEncryptionInitValid(t *testing.T) {
	t.Parallel()

	e := AwsKmsEncryption{}

	err := e.Init(map[string]string{
		"endpoint":   getKmsHost(),
		"kms_key_id": "test",
		"region":     "ca-central-1",
	})

	if err != nil {
		t.Errorf("AwsKmsEncryption.Init() = %v, want %v", err, nil)
	}
}

func TestAwsKmsEncryptionEncrypt(t *testing.T) {
	t.Parallel()

	e := AwsKmsEncryption{}

	_ = e.Init(map[string]string{
		"endpoint":   getKmsHost(),
		"kms_key_id": "bc436485-5092-42b8-92a3-0aa8b93536dc", // Set in .devcontainer/docker/kms/init.yml
		"region":     "ca-central-1",
	})

	plaintext := []byte("test")

	cipher, key, err := e.Encrypt(plaintext)

	if err != nil {
		t.Errorf("AwsKmsEncryption.Encrypt() = %v, want %v", err, nil)
	}

	if len(cipher) == 0 {
		t.Errorf("AwsKmsEncryption.Encrypt() = %v, want cipher length %v", len(cipher), "greater than 0")
	}

	if len(key) == 0 {
		t.Errorf("AwsKmsEncryption.Encrypt() = %v, want key length %v", len(key), "greater than 0")
	}
}

func TestAwsKmsEncryptionDecrypt(t *testing.T) {
	t.Parallel()

	e := AwsKmsEncryption{}

	_ = e.Init(map[string]string{
		"endpoint":   getKmsHost(),
		"kms_key_id": "bc436485-5092-42b8-92a3-0aa8b93536dc", // Set in .devcontainer/docker/kms/init.yml
		"region":     "ca-central-1",
	})

	plaintext := []byte("test")

	cipher, key, err := e.Encrypt(plaintext)

	if err != nil {
		t.Errorf("AwsKmsEncryption.Encrypt() = %v, want %v", err, nil)
	}

	decrypted, err := e.Decrypt(cipher, key)

	if err != nil {
		t.Errorf("AwsKmsEncryption.Decrypt() = %v, want %v", err, nil)
	}

	if string(decrypted) != string(plaintext) {
		t.Errorf("AwsKmsEncryption.Decrypt() = %v, want %v", string(decrypted), string(plaintext))
	}
}
