package backupservice

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

type controllerSource struct{ path string }

func (s controllerSource) Backup(context.Context) (string, error) { return s.path, nil }

type controllerTarget struct {
	filename string
}

func (t *controllerTarget) AddFile(context.Context, []byte, string) error { return nil }

func TestControllerHandlesBackupWithTargetAndEncryption(t *testing.T) {
	path := filepath.Join(t.TempDir(), "paperless-export.zip")
	if err := os.WriteFile(path, []byte("zip"), 0o600); err != nil {
		t.Fatal(err)
	}

	controller := NewController(nil, func(method string, file []byte) ([]byte, error) {
		return append([]byte(method+":"), file...), nil
	})
	controller.AddSource("paperless", controllerSource{path: path})
	controller.AddTarget("nextcloud", &controllerTarget{})

	request := httptest.NewRequest(http.MethodGet, "/api/v1/backup?source=paperless&target=nextcloud&encryption=pgp", nil)
	response := httptest.NewRecorder()
	controller.HandleBackup(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", response.Code, response.Body.String())
	}
}

func TestControllerRequiresSourceAndTarget(t *testing.T) {
	controller := NewController(nil, nil)

	for _, requestURL := range []string{
		"/api/v1/backup?target=nextcloud",
		"/api/v1/backup?source=paperless",
	} {
		request := httptest.NewRequest(http.MethodGet, requestURL, nil)
		response := httptest.NewRecorder()
		controller.HandleBackup(response, request)
		if response.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400 for %s, got %d", requestURL, response.Code)
		}
	}
}
