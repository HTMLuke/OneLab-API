package backupservice

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

type testSource struct {
	path string
}

func (s testSource) Backup(context.Context) (string, error) {
	return s.path, nil
}

type testTarget struct {
	file     []byte
	filename string
}

func (t *testTarget) AddFile(_ context.Context, file []byte, filename string) error {
	t.file = file
	t.filename = filename
	return nil
}

func TestCreateEncryptsAndUploadsBackup(t *testing.T) {
	path := filepath.Join(t.TempDir(), "paperless-export.zip")
	if err := os.WriteFile(path, []byte("zip contents"), 0o600); err != nil {
		t.Fatal(err)
	}

	target := &testTarget{}
	service := New(func(method string, file []byte) ([]byte, error) {
		return append([]byte(method+":"), file...), nil
	})
	result, err := service.Create(context.Background(), testSource{path: path}, target, "nextcloud", "pgp")
	if err != nil {
		t.Fatal(err)
	}

	if result.Filename != "paperless-export.zip.pgp" || target.filename != result.Filename {
		t.Fatalf("unexpected filename: result=%q target=%q", result.Filename, target.filename)
	}
	if string(target.file) != "pgp:zip contents" {
		t.Fatalf("unexpected encrypted content: %q", target.file)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected local backup to be removed, stat error: %v", err)
	}
}

func TestFindPaperlessContainerNameFromDockerPsOutput(t *testing.T) {
	got, err := findPaperlessContainerNameFromDockerPsOutput("paperless-webserver\tabc123\npaperless-db\tdef456\n")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got != "paperless-webserver" {
		t.Fatalf("expected paperless-webserver, got %q", got)
	}
}

func TestFindPaperlessContainerNameFromDockerPsOutputUsesFallback(t *testing.T) {
	got, err := findPaperlessContainerNameFromDockerPsOutput("some-other\tzzz\npaperless\tqqq\n")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got != "paperless" {
		t.Fatalf("expected paperless, got %q", got)
	}
}
