package encryption

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/kms/types"
)

type AwsKmsEncryption struct {
	client   *kms.Client
	kmsKeyId string
}

func (a *AwsKmsEncryption) Init(c map[string]string) error {
	if c["kms_key_id"] == "" {
		return fmt.Errorf("kms_key_id is required")
	}

	a.kmsKeyId = c["kms_key_id"]

	if c["region"] == "" {
		return fmt.Errorf("region is required")
	}

	cfg, err := config.LoadDefaultConfig(
		context.TODO(),
		config.WithRegion(c["region"]))

	if err != nil {
		return err
	}

	a.client = kms.NewFromConfig(cfg, func(o *kms.Options) {
		if c["endpoint"] != "" {
			endpoint := c["endpoint"]
			o.BaseEndpoint = &endpoint
		}
	})

	return nil
}

func (a *AwsKmsEncryption) Encrypt(plaintext []byte) ([]byte, []byte, error) {
	result, err := a.client.GenerateDataKey(context.TODO(), &kms.GenerateDataKeyInput{
		KeyId:   &a.kmsKeyId,
		KeySpec: types.DataKeySpecAes256,
	})

	if err != nil {
		return nil, nil, err
	}

	aes, err := aes.NewCipher([]byte(result.Plaintext))
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

	encryptedText := gcm.Seal(nonce, nonce, plaintext, nil)

	return encryptedText, result.CiphertextBlob, nil

}

func (a *AwsKmsEncryption) Decrypt(ciphertext []byte, key []byte) ([]byte, error) {
	// Decrypt the CipherTextBlob with the KMS key
	result, err := a.client.Decrypt(context.TODO(), &kms.DecryptInput{
		CiphertextBlob: key,
	})

	if err != nil {
		return nil, err
	}

	aes, err := aes.NewCipher([]byte(result.Plaintext))
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(aes)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}
