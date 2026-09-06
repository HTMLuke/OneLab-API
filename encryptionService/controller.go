package encryptionservice

import "fmt"

// Controller stores internal encryption services without exposing HTTP routes.
type Controller struct {
	services map[string]Service
}

// NewController creates an internal encryption service registry.
func NewController() *Controller {
	return &Controller{services: make(map[string]Service)}
}

// AddIntegration registers an encryption service for use by application code.
func (c *Controller) AddIntegration(name string, service Service) {
	c.services[name] = service
}

// GetIntegration returns a registered encryption service by name.
func (c *Controller) GetIntegration(name string) (Service, bool) {
	service, exists := c.services[name]
	return service, exists
}

// Encrypt encrypts data with a named registered encryption service.
func (c *Controller) Encrypt(method string, file []byte) ([]byte, error) {
	service, exists := c.GetIntegration(method)
	if !exists {
		return nil, fmt.Errorf("encryption method %q is not configured", method)
	}
	return service.Encrypt(file)
}
