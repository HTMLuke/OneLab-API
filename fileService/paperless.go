package fileService

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/HTMLuke/OneLab-API/secretProvider"
	"github.com/HTMLuke/OneLab-API/shellService"
)

// PaperlessService handles sending files to Paperless-ngx.
type PaperlessService struct {
	checkURL      string
	addUrl        string
	getUrl        string
	apiLookupUrl  string
	token         string
	dockerService shellService.ShellService
}

type PaperlessFileMetadata struct {
	ID                  int     `json:"id"`
	Correspondent       int     `json:"correspondent"`
	DocumentType        int     `json:"document_type"`
	StoragePath         *string `json:"storage_path"`
	Title               string  `json:"title"`
	Content             string  `json:"content"`
	Tags                []int   `json:"tags"`
	Created             string  `json:"created"`
	CreatedDate         string  `json:"created_date"`
	Modified            string  `json:"modified"`
	Added               string  `json:"added"`
	DeletedAt           *string `json:"deleted_at"`
	ArchiveSerialNumber *string `json:"archive_serial_number"`
	OriginalFileName    string  `json:"original_file_name"`
	ArchivedFileName    string  `json:"archived_file_name"`
	Owner               int     `json:"owner"`
	UserCanChange       bool    `json:"user_can_change"`
	IsSharedByRequester bool    `json:"is_shared_by_requester"`
	Notes               []any   `json:"notes"`
	CustomFields        []any   `json:"custom_fields"`
	PageCount           int     `json:"page_count"`
	MimeType            string  `json:"mime_type"`
}

type PaperlessFileResponse struct {
	ID               int    `json:"id"`
	Title            string `json:"title"`
	OriginalFileName string `json:"original_file_name"`
	Created          string `json:"created"`
	PageCount        int    `json:"page_count"`
}

type PaperlessSearchResponse struct {
	Count    int                     `json:"count"`
	Next     *string                 `json:"next"`
	Previous *string                 `json:"previous"`
	All      []int                   `json:"all"`
	Results  []PaperlessFileMetadata `json:"results"`
}

func init() {
	RegisterIntegration("paperless", func(baseURL string, s secretProvider.SecretService) (IntegrationService, error) {
		return NewPaperlessService(baseURL, s)
	})
}

func NewPaperlessService(baseURL string, secretService secretProvider.SecretService) (*PaperlessService, error) {
	token, err := secretService.GetSecret("ONELAB_PAPERLESS_TOKEN")
	if err != nil {
		return nil, err
	}
	if baseURL == "" {
		baseURL = "http://paperless.local/"
	}
	checkURL, _ := url.JoinPath(baseURL, "api/documents/")
	addUrl, _ := url.JoinPath(baseURL, "api/documents/post_document/")
	getUrl, _ := url.JoinPath(baseURL, "api/documents/")
	apiLookupUrl, _ := url.JoinPath(baseURL, "api/documents/")
	return &PaperlessService{
		checkURL:      checkURL,
		addUrl:        addUrl,
		getUrl:        getUrl,
		token:         token,
		apiLookupUrl:  apiLookupUrl,
		dockerService: shellService.NewDockerCmdService(),
	}, nil
}

func findPaperlessContainerNameFromDockerPsOutput(output string) (string, error) {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	var candidates []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}
		candidates = append(candidates, parts[0])
	}
	if len(candidates) == 0 {
		return "", fmt.Errorf("no docker containers found")
	}

	for _, name := range candidates {
		lower := strings.ToLower(name)
		if lower == "paperless-webserver" || strings.Contains(lower, "paperless-webserver") {
			return name, nil
		}
	}
	for _, name := range candidates {
		lower := strings.ToLower(name)
		if strings.Contains(lower, "paperless") {
			return name, nil
		}
	}
	return "", fmt.Errorf("could not determine paperless webserver container from docker ps output")
}

func (s *PaperlessService) resolveWebserverContainer(ctx context.Context) (string, error) {
	if s.dockerService == nil {
		return "", fmt.Errorf("docker command service is not configured")
	}

	cmd := exec.CommandContext(ctx, "docker", "ps", "--format", "{{.Names}}\t{{.ID}}")
	output, err := cmd.CombinedOutput()
	if err != nil {
		if len(output) > 0 {
			return "", fmt.Errorf("docker ps failed: %w: %s", err, strings.TrimSpace(string(output)))
		}
		return "", fmt.Errorf("docker ps failed: %w", err)
	}
	containerName, err := findPaperlessContainerNameFromDockerPsOutput(string(output))
	if err != nil {
		return "", err
	}
	return containerName, nil
}

func (s *PaperlessService) ExportDocuments(ctx context.Context) (string, error) {
	containerName, err := s.resolveWebserverContainer(ctx)
	if err != nil {
		return "", err
	}

	if _, err := s.dockerService.Execute(ctx, containerName, "mkdir -p /tmp/export"); err != nil {
		return "", fmt.Errorf("failed to create temp export directory in container %s: %w", containerName, err)
	}
	if _, err := s.dockerService.Execute(ctx, containerName, `sh -c "rm -f /tmp/export/export-*.zip"`); err != nil {
		return "", fmt.Errorf("failed to remove previous export artifacts in container %s: %w", containerName, err)
	}

	output, err := s.dockerService.Execute(ctx, containerName, "document_exporter /tmp/export -z")
	if err != nil {
		if output != "" {
			return output, fmt.Errorf("paperless export failed in container %s: %w: %s", containerName, err, strings.TrimSpace(output))
		}
		return "", fmt.Errorf("paperless export failed in container %s: %w", containerName, err)
	}
	return output, nil
}

func findLatestZipInDockerExport(output string) (string, error) {
	lines := strings.FieldsFunc(output, func(r rune) bool { return r == '\n' || r == '\r' })
	var candidate string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasSuffix(line, ".zip") {
			candidate = line
		}
	}
	if candidate == "" {
		return "", fmt.Errorf("no zip export file found in container output")
	}
	return candidate, nil
}

func (s *PaperlessService) Backup(ctx context.Context) (string, error) {
	containerName, err := s.resolveWebserverContainer(ctx)
	if err != nil {
		return "", err
	}

	if _, err := s.dockerService.Execute(ctx, containerName, "mkdir -p /tmp/export"); err != nil {
		return "", fmt.Errorf("failed to create tmp export directory in container %s: %w", containerName, err)
	}
	if _, err := s.dockerService.Execute(ctx, containerName, `sh -c "rm -f /tmp/export/export-*.zip"`); err != nil {
		return "", fmt.Errorf("failed to remove stale export files in container %s: %w", containerName, err)
	}

	output, err := s.dockerService.Execute(ctx, containerName, "document_exporter /tmp/export -z")
	if err != nil {
		if output != "" {
			return "", fmt.Errorf("paperless export failed in container %s: %w: %s", containerName, err, strings.TrimSpace(output))
		}
		return "", fmt.Errorf("paperless export failed in container %s: %w", containerName, err)
	}

	listOutput, err := s.dockerService.Execute(ctx, containerName, "sh -lc 'ls -1 /tmp/export/*.zip 2>/dev/null | tail -n 1'")
	if err != nil {
		return "", fmt.Errorf("failed to locate exported zip in container %s: %w", containerName, err)
	}
	zipPath, err := findLatestZipInDockerExport(listOutput)
	if err != nil {
		return "", err
	}
	archiveName := filepath.Base(zipPath)
	localBackupDir := filepath.Join("/tmp", "onelab-paperless-backups")
	if err := os.MkdirAll(localBackupDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create local backup directory: %w", err)
	}
	localBackupPath := filepath.Join(localBackupDir, archiveName)
	if _, err := os.Stat(localBackupPath); err == nil {
		stamp := time.Now().UTC().Format("20060102-150405")
		ext := filepath.Ext(archiveName)
		name := strings.TrimSuffix(archiveName, ext)
		localBackupPath = filepath.Join(localBackupDir, fmt.Sprintf("%s-%s%s", name, stamp, ext))
	}

	cpCmd := exec.CommandContext(ctx, "docker", "cp", fmt.Sprintf("%s:%s", containerName, zipPath), localBackupPath)
	cpOutput, cpErr := cpCmd.CombinedOutput()
	if cpErr != nil {
		if len(cpOutput) > 0 {
			return "", fmt.Errorf("docker cp failed: %w: %s", cpErr, strings.TrimSpace(string(cpOutput)))
		}
		return "", fmt.Errorf("docker cp failed: %w", cpErr)
	}

	return localBackupPath, nil
}

func (s *PaperlessService) AddFile(ctx context.Context, file []byte, filename string) error {
	// Initialize buffer and multipart writer
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Create the form file field (Paperless strictly requires the field name to be "document")
	part, err := writer.CreateFormFile("document", filename)
	if err != nil {
		return fmt.Errorf("failed to create multipart form file: %v", err)
	}

	// Stream the []byte into the multipart field
	_, err = io.Copy(part, bytes.NewReader(file))
	if err != nil {
		return fmt.Errorf("failed to copy file bytes to multipart: %v", err)
	}

	// CRITICAL: Close the writer BEFORE creating the request.
	// This appends the final multipart boundary to the buffer.
	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close multipart writer: %v", err)
	}

	// Prepare the HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", s.addUrl, body)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	// The Content-Type header MUST include the specific boundary generated by the writer!
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Set the Authorization Header for Paperless
	if s.token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Token %s", s.token))
	}

	// 5. Execute the request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request to paperless: %v", err)
	}
	defer resp.Body.Close()

	// 6. Validate the response
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("paperless upload failed: status %d, response: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}
func (s *PaperlessService) GetFile(ctx context.Context, fileID string) ([]byte, error) {
	// Build the download URL: .../api/documents/{id}/download/
	downloadPath, err := url.JoinPath(s.apiLookupUrl, fileID, "download/")
	if err != nil {
		return nil, fmt.Errorf("failed to build download URL: %v", err)
	}
	//create request with context
	req, err := http.NewRequestWithContext(ctx, "GET", downloadPath, nil)
	if err != nil {
		return nil, err
	}

	// Set required headers
	if s.token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Token %s", s.token))
	}

	//  Send the request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send download request to paperless: %v", err)
	}
	defer resp.Body.Close()

	// Handle non-success HTTP statuses
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("paperless download failed: status %d, response: %s", resp.StatusCode, string(bodyBytes))
	}

	// Read the raw file binary data into a byte slice
	fileData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read paperless file content: %v", err)
	}

	return fileData, nil
}
func (s *PaperlessService) LookupFile(ctx context.Context, filename string) (any, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", s.apiLookupUrl, nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Add("title__icontains", filename)
	req.URL.RawQuery = q.Encode()
	req.Header.Set("Authorization", fmt.Sprintf("Token %s", s.token))
	req.Header.Set("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed: received status code %d, response: %s", resp.StatusCode, string(bodyBytes))
	}
	var searchResp PaperlessSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, fmt.Errorf("failed to decode paperless lookup response: %v", err)
	}
	var results []PaperlessFileResponse
	for _, entry := range searchResp.Results {
		results = append(results, PaperlessFileResponse{
			ID:               entry.ID,
			Title:            entry.Title,
			OriginalFileName: entry.OriginalFileName,
			Created:          entry.Created,
			PageCount:        entry.PageCount,
		})
	}
	return results, nil
}
func (s *PaperlessService) CheckStatus(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", s.checkURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Token %s", s.token))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("paperless authentication failed with status: %d", resp.StatusCode)
	}

	return nil
}
