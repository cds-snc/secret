package encryption

type EncryptionBackend interface {
	Init(map[string]string) error
	Encrypt([]byte) ([]byte, []byte, error)
	Decrypt([]byte, []byte) ([]byte, error)
}
