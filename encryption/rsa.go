// A valid question to ask of this code is why it implements envelope encryption by encrypting the
// data with a symmetric key and then encrypting the symmetric key with the public key. The reason is
// that most cloud providers (ex. AWS KMS) force you to use envelope encryption because they can only
// encrypt a small amount of data directly. To match this behaviour we implement a similar pattern, even
// though it is not strictly necessary.

package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
)

// RsaKeyPair is a struct that holds the private and public keys
type RsaKeyPair struct {
	privateKey []byte
	publicKey  []byte
}

func (k *RsaKeyPair) Init(config map[string]string) error {
	publicKeyPEM, err := os.ReadFile(config["publicKeyPath"])
	if err != nil {
		return err
	}

	privateKeyPEM, err := os.ReadFile(config["privateKeyPath"])
	if err != nil {
		return err
	}

	k.privateKey = privateKeyPEM
	k.publicKey = publicKeyPEM

	return nil
}

func (k *RsaKeyPair) Encrypt(data []byte) ([]byte, []byte, error) {
	// Encrypt data with with a random AES key
	dataKey, err := generateRandomBytes(32)
	if err != nil {
		return nil, nil, err
	}

	aes, err := aes.NewCipher([]byte(dataKey))
	if err != nil {
		return nil, nil, err
	}

	gcm, err := cipher.NewGCM(aes)
	if err != nil {
		return nil, nil, err
	}

	nonce, err := generateRandomBytes(gcm.NonceSize())
	if err != nil {
		return nil, nil, err
	}

	envelopeText := gcm.Seal(nonce, nonce, data, nil)

	// Encrypt the AES key with the public key
	publicKeyBlock, _ := pem.Decode(k.publicKey)
	publicKey, err := x509.ParsePKIXPublicKey(publicKeyBlock.Bytes)
	if err != nil {
		return nil, nil, err
	}

	ciphertext, err := rsa.EncryptPKCS1v15(rand.Reader, publicKey.(*rsa.PublicKey), dataKey)
	if err != nil {
		return nil, nil, err
	}

	// Return the encrypted data and the encrypted AES key
	return envelopeText, ciphertext, nil
}

func (k *RsaKeyPair) Decrypt(data []byte, key []byte) ([]byte, error) {
	// Decrypt the AES key with the private key
	privateKeyBlock, _ := pem.Decode(k.privateKey)
	privateKey, err := x509.ParsePKCS1PrivateKey(privateKeyBlock.Bytes)
	if err != nil {
		return nil, err
	}

	dataKey, err := rsa.DecryptPKCS1v15(rand.Reader, privateKey, key)
	if err != nil {
		return nil, err
	}

	// Decrypt the data with the AES key
	aes, err := aes.NewCipher([]byte(dataKey))
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(aes)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	// Return the decrypted data
	return plaintext, nil
}

func generateRandomBytes(length int) ([]byte, error) {
	key := make([]byte, length)
	_, err := rand.Read(key)
	if err != nil {
		return nil, err
	}
	return key, nil

}
