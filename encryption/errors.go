package encryption

import "errors"

var ErrInvalidCiphertext = errors.New("invalid ciphertext")

const (
	standardGCMNonceSize = 12
	standardGCMTagSize   = 16
)

func validateGCMCiphertext(ciphertext []byte) error {
	if len(ciphertext) < standardGCMNonceSize+standardGCMTagSize {
		return ErrInvalidCiphertext
	}
	return nil
}
