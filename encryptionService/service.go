package encryptionservice

// Service exposes the internal OpenPGP encryption helper.
type Service interface {
	Encrypt(plaintext []byte) ([]byte, error)
	EncryptString(value string) ([]byte, error)
	EncryptJSON(value any) ([]byte, error)
	EncryptFile(file []byte) ([]byte, error)
}
