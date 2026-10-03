package encryptionservice

import (
	"log/slog"

	"github.com/HTMLuke/OneLab-API/secretProvider"
)

// ServiceBuilder constructs an internal encryption service.
type ServiceBuilder func(secrets secretProvider.SecretService, logger *slog.Logger) (Service, error)

var builders = map[string]ServiceBuilder{}

// RegisterService records an internal encryption service implementation.
func RegisterService(name string, build ServiceBuilder) {
	builders[name] = build
}

// Builders returns all registered internal encryption services.
func Builders() map[string]ServiceBuilder { return builders }
