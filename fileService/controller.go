package fileService

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type FileController struct {
	// A map of available integration services (e.g., "nextcloud", "paperless")
	integrations map[string]IntegrationService
}

func NewFileController() *FileController {
	return &FileController{
		integrations: make(map[string]IntegrationService),
	}
}

// AddIntegration allows registering more services dynamically
func (c *FileController) AddIntegration(name string, service IntegrationService) {
	c.integrations[name] = service
}

// decodeJSON is a general helper to parse a JSON request body into a provided object
func decodeJSON(r *http.Request, v any) error {
	return json.NewDecoder(r.Body).Decode(v)
}
func (c *FileController) GetValueFromKey(r *http.Request, key string) (string, error) {
	var bodyMap map[string]interface{}
	if err := decodeJSON(r, &bodyMap); err != nil {
		return "", err
	}
	if value, exists := bodyMap[key]; exists {
		if str, ok := value.(string); ok {
			return str, nil
		}
	}
	return "", fmt.Errorf("key '%s' not found or not a string", key)
}

func (c *FileController) TransferHandler(w http.ResponseWriter, r *http.Request) {
	var targetSoft string
	var sourceSoft string

	targetSoft, err := c.GetValueFromKey(r, "target")
	if err != nil {
		http.Error(w, fmt.Sprintf("Error retrieving target: %v", err), http.StatusBadRequest)
		return
	}

	if targetSoft == "" {
		http.Error(w, "Target field 'target' is missing in the request body", http.StatusBadRequest)
		return
	}
	targetSoft = strings.ToLower(targetSoft)
	svc, exists := c.integrations[targetSoft]
	if !exists {
		http.Error(w, fmt.Sprintf("Target software '%s' is not supported", targetSoft), http.StatusBadRequest)
		return
	}

	sourceSoft, err = c.GetValueFromKey(r, "source")
	if err != nil {
		http.Error(w, fmt.Sprintf("Error retrieving source: %v", err), http.StatusBadRequest)
		return
	}
	if sourceSoft == "" {
		http.Error(w, "Source field 'source' is missing in the request body", http.StatusBadRequest)
		return
	}

	sourceSoft = strings.ToLower(sourceSoft)
	sourceExists := c.integrations[sourceSoft] != nil
	if !sourceExists {
		http.Error(w, fmt.Sprintf("Source software '%s' is not supported", sourceSoft), http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Error retrieving the file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Let the specific integration handle the logic
	if err := svc.TransferFile(r.Context(), file, header); err != nil {
		http.Error(w, fmt.Sprintf("Transfer to %s failed", targetSoft), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("File %s successfully transferred to %s\n", header.Filename, targetSoft)))
}

// CheckIntegrationsStatus verifies the connectivity of all registered integrations.
func (c *FileController) CheckIntegrationsStatus(ctx context.Context) map[string]string {
	statusMap := make(map[string]string)
	for name, svc := range c.integrations {
		if err := svc.CheckStatus(ctx); err != nil {
			statusMap[name] = fmt.Sprintf("DOWN (%v)", err.Error())
		} else {
			statusMap[name] = "OK"
		}
	}
	return statusMap
}

func (c *FileController) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/files/transfer", c.TransferHandler)
}
