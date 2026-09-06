package encryptionservice

import (
	"bytes"
	"errors"
	"testing"

	"github.com/HTMLuke/OneLab-API/secretProvider"
	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/armor"
)

type testSecrets struct {
	value string
}

func (s testSecrets) GetSecret(key string) (string, error) {
	if key != publicKeySecret {
		return "", errors.New("unexpected secret key")
	}
	return s.value, nil
}

var _ secretProvider.SecretService = testSecrets{}

func armoredKey(t *testing.T, entity *openpgp.Entity, private bool) string {
	t.Helper()
	var output bytes.Buffer
	blockType := "PGP PUBLIC KEY BLOCK"
	if private {
		blockType = "PGP PRIVATE KEY BLOCK"
	}
	armored, err := armor.Encode(&output, blockType, nil)
	if err != nil {
		t.Fatal(err)
	}
	if private {
		err = entity.SerializePrivate(armored, nil)
	} else {
		err = entity.Serialize(armored)
	}
	if err != nil {
		t.Fatal(err)
	}
	if err := armored.Close(); err != nil {
		t.Fatal(err)
	}
	return output.String()
}

func TestNewPGPServiceRejectsPrivateKey(t *testing.T) {
	entity, err := openpgp.NewEntity("test", "", "test@example.com", nil)
	if err != nil {
		t.Fatal(err)
	}

	_, err = NewPGPService(testSecrets{value: armoredKey(t, entity, true)}, nil)
	if err == nil {
		t.Fatal("expected private key to be rejected")
	}
}

func TestPGPServiceEncryptValuesAndFile(t *testing.T) {
	entity, err := openpgp.NewEntity("test", "", "test@example.com", nil)
	if err != nil {
		t.Fatal(err)
	}

	service, err := NewPGPService(testSecrets{value: armoredKey(t, entity, false)}, nil)
	if err != nil {
		t.Fatal(err)
	}

	ciphertext, err := service.Encrypt([]byte("secret"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(ciphertext, []byte("-----BEGIN PGP MESSAGE-----")) {
		t.Fatalf("expected armored message, got %q", ciphertext[:min(len(ciphertext), 32)])
	}
	if _, err := service.EncryptString("secret"); err != nil {
		t.Fatalf("encrypt string: %v", err)
	}
	if _, err := service.EncryptJSON(map[string]string{"value": "secret"}); err != nil {
		t.Fatalf("encrypt JSON: %v", err)
	}

	if _, err := service.EncryptFile([]byte("file contents")); err != nil {
		t.Fatalf("encrypt file: %v", err)
	}
}
