package encryption

// NullEncryption is a dummy encryption backend that does not encrypt or decrypt anything.
type NullEncryption struct{}

// Init does nothing.
func (n NullEncryption) Init(config map[string]string) error {
	return nil
}

// Encrypt does not encrypt anything.
func (n NullEncryption) Encrypt(plaintext []byte) ([]byte, []byte, error) {
	return plaintext, nil, nil
}

// Decrypt does not decrypt anything.
func (n NullEncryption) Decrypt(ciphertext []byte, key []byte) ([]byte, error) {
	return ciphertext, nil
}
