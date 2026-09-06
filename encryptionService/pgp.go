package encryptionservice

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/HTMLuke/OneLab-API/secretProvider"
	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/armor"
)

const publicKeySecret = "ONELAB_PGP_PUBLIC_KEY"

func init() {
	RegisterService("pgp", func(secrets secretProvider.SecretService, logger *slog.Logger) (Service, error) {
		return NewPGPService(secrets, logger)
	})
}

// PGPService encrypts data to the configured public key.
// It deliberately never retains or accepts private key material.
type PGPService struct {
	keyring openpgp.EntityList
}

// NewPGPService loads an armored public key from ONELAB_PGP_PUBLIC_KEY.
func NewPGPService(secrets secretProvider.SecretService, logger *slog.Logger) (*PGPService, error) {
	if secrets == nil {
		return nil, fmt.Errorf("secret service is required")
	}

	keyText, err := secrets.GetSecret(publicKeySecret)
	if err != nil {
		return nil, err
	}

	keyring, err := openpgp.ReadArmoredKeyRing(bytes.NewBufferString(keyText))
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", publicKeySecret, err)
	}
	if len(keyring) == 0 {
		return nil, fmt.Errorf("%s does not contain an OpenPGP key", publicKeySecret)
	}
	for _, entity := range keyring {
		if entity.PrivateKey != nil {
			return nil, fmt.Errorf("%s must contain public key material only", publicKeySecret)
		}
	}

	if logger != nil {
		logger.Info("PGP public key loaded", "secret_key", publicKeySecret, "keys", len(keyring))
	}
	return &PGPService{keyring: keyring}, nil
}

// Encrypt returns an ASCII-armored OpenPGP message encrypted to the configured key.
func (s *PGPService) Encrypt(plaintext []byte) ([]byte, error) {
	if s == nil || len(s.keyring) == 0 {
		return nil, fmt.Errorf("PGP service is not initialized")
	}

	var output bytes.Buffer
	armored, err := armor.Encode(&output, "PGP MESSAGE", nil)
	if err != nil {
		return nil, fmt.Errorf("create armored message: %w", err)
	}
	ciphertext, err := openpgp.Encrypt(armored, s.keyring, nil, nil, nil)
	if err != nil {
		_ = armored.Close()
		return nil, fmt.Errorf("encrypt message: %w", err)
	}
	if _, err := ciphertext.Write(plaintext); err != nil {
		_ = ciphertext.Close()
		_ = armored.Close()
		return nil, fmt.Errorf("write encrypted message: %w", err)
	}
	if err := ciphertext.Close(); err != nil {
		_ = armored.Close()
		return nil, fmt.Errorf("close encrypted message: %w", err)
	}
	if err := armored.Close(); err != nil {
		return nil, fmt.Errorf("close armored message: %w", err)
	}
	return output.Bytes(), nil
}

// EncryptString encrypts a string value to the configured public key.
func (s *PGPService) EncryptString(value string) ([]byte, error) {
	return s.Encrypt([]byte(value))
}

// EncryptJSON marshals a value as JSON and encrypts the resulting bytes.
func (s *PGPService) EncryptJSON(value any) ([]byte, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("marshal value as JSON: %w", err)
	}
	return s.Encrypt(data)
}

// EncryptFile encrypts file contents represented as bytes, matching the file service APIs.
func (s *PGPService) EncryptFile(file []byte) ([]byte, error) {
	return s.Encrypt(file)
}
