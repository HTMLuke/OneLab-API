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

func (c *FileController) GetValueFromBody(r *http.Request, key string) (string, error) {
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

func (c *FileController) GetValueFromQuery(r *http.Request, key string) (string, error) {
	value := r.URL.Query().Get(key)
	if value == "" {
		return "", fmt.Errorf("missing '%s' in query parameters", key)
	}
	return value, nil
}
func (c *FileController) HTTPTransferHandler(w http.ResponseWriter, r *http.Request) {
	source, err := c.GetValueFromQuery(r, "source")
	if err != nil {
		http.Error(w, fmt.Sprintf("source query parameter is required: %v", err), http.StatusBadRequest)
		return
	}
	target, err := c.GetValueFromQuery(r, "target")
	if err != nil {
		http.Error(w, fmt.Sprintf("target query parameter is required: %v", err), http.StatusBadRequest)
		return
	}

	filename, err := c.GetValueFromQuery(r, "filename")
	if err != nil {
		http.Error(w, fmt.Sprintf("filename query parameter is required: %v", err), http.StatusBadRequest)
		return
	}

	tvc, exists := c.integrations[target]
	if !exists {
		http.Error(w, fmt.Sprintf("Target software '%s' is not supported", target), http.StatusBadRequest)
		return
	}

	svc, exists := c.integrations[source]
	if !exists {
		http.Error(w, fmt.Sprintf("Source software '%s' is not supported", source), http.StatusBadRequest)
		return
	}
	err, statusCode := c.TransferHandler(svc, tvc, filename, r.Context())
	if err != nil {
		http.Error(w, fmt.Sprintf("Error transferring file: %v", err), statusCode)
		return
	} else {
		w.WriteHeader(statusCode)
		return
	}
}
func (c *FileController) TransferHandler(svc IntegrationService, tvc IntegrationService, filename string, ctx context.Context) (error, int) {

	_, err, statusCode := c.LookupHandler(svc, filename, ctx) // Reuse the lookup handler to validate the file exists before transfer
	if err != nil {
		return err, statusCode
	}
	return nil, http.StatusOK
	// Let the specific integration handle the logic
	// if transferer, ok := svc.(FileTransferer); ok {
	// 	if err := transferer.TransferFile(ctx, nil, nil); err != nil {
	// 		return fmt.Errorf("Transfer failed: %w", err), http.StatusInternalServerError
	// 	}else {
	// 		return nil, http.StatusOK
	// 	}

	// } else {
	// 	return fmt.Errorf("File transfer not supported for target"), http.StatusBadRequest
	// }
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
func (c *FileController) HTTPLookupHandler(w http.ResponseWriter, r *http.Request) {
	source, err := c.GetValueFromQuery(r, "source")
	if err != nil {
		http.Error(w, fmt.Sprintf("source query parameter is required: %v", err), http.StatusBadRequest)
		return
	}

	filename, err := c.GetValueFromQuery(r, "filename")
	if err != nil {
		http.Error(w, fmt.Sprintf("filename query parameter is required: %v", err), http.StatusBadRequest)
		return
	}

	sourceSoft := strings.ToLower(source)
	svc, exists := c.integrations[sourceSoft]
	if !exists {
		http.Error(w, fmt.Sprintf("source software '%s' is not supported", sourceSoft), http.StatusBadRequest)
		return
	}

	result, err, statusCode := c.LookupHandler(svc, filename, r.Context())
	if err != nil {
		http.Error(w, fmt.Sprintf("Error looking up file from %s: %v", sourceSoft, err), statusCode)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(result); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
		return
	}

}
func (c *FileController) LookupHandler(svc IntegrationService, filename string, ctx context.Context) (any, error, int) {

	// Check if the service implements the FileLookuper interface
	if lookuper, ok := svc.(FileLookuper); ok {
		result, err := lookuper.LookupFile(ctx, filename)
		if err != nil {
			return nil, fmt.Errorf("Error looking up file from: %v", err), http.StatusInternalServerError
		}

		return result, nil, http.StatusOK

	} else {
		return nil, fmt.Errorf("lookup not supported for source"), http.StatusBadRequest
	}
}
func (c *FileController) RegisterRoutes(mux *http.ServeMux) {
	// e.g. POST /api/v1/files/transfer/nextcloud
	mux.HandleFunc("POST /api/v1/files/transfer/{target}", c.HTTPLookupHandler)
	mux.HandleFunc("GET /api/v1/files/lookup/", c.HTTPLookupHandler)
}
