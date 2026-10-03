package fileService

import (
	"context"
	"testing"
)

type transferTestService struct {
	uploadedFilename string
	uploadedFile     []byte
	originalFilename string
}

func (s *transferTestService) CheckStatus(context.Context) error {
	return nil
}

func (s *transferTestService) GetFile(context.Context, string) ([]byte, error) {
	return []byte("source file"), nil
}

func (s *transferTestService) AddFile(_ context.Context, file []byte, filename string) error {
	s.uploadedFile = file
	s.uploadedFilename = filename
	return nil
}

func (s *transferTestService) LookupFile(context.Context, string) (any, error) {
	return []PaperlessFileResponse{{ID: 1, OriginalFileName: s.originalFilename}}, nil
}

func TestTransferHandlerAddsArmoredExtensionWhenEncrypted(t *testing.T) {
	source := &transferTestService{originalFilename: "report.pdf"}
	target := &transferTestService{}
	controller := NewFileController(nil, func(_ string, file []byte) ([]byte, error) {
		return append([]byte("-----BEGIN PGP MESSAGE-----\n"), file...), nil
	})

	err, status := controller.TransferHandler(source, target, "report.pdf", "pgp", context.Background())
	if err != nil {
		t.Fatalf("expected transfer to succeed, got %v", err)
	}
	if status != 200 {
		t.Fatalf("expected status 200, got %d", status)
	}
	if target.uploadedFilename != "report.pdf.pgp" {
		t.Fatalf("expected encrypted filename report.pdf.pgp, got %q", target.uploadedFilename)
	}
	if len(target.uploadedFile) == 0 {
		t.Fatal("expected encrypted file contents to be uploaded")
	}
}

func TestTransferHandlerKeepsFilenameWhenUnencrypted(t *testing.T) {
	source := &transferTestService{originalFilename: "report.pdf"}
	target := &transferTestService{}
	controller := NewFileController(nil, nil)

	err, status := controller.TransferHandler(source, target, "report.pdf", "", context.Background())
	if err != nil {
		t.Fatalf("expected transfer to succeed, got %v", err)
	}
	if status != 200 {
		t.Fatalf("expected status 200, got %d", status)
	}
	if target.uploadedFilename != "report.pdf" {
		t.Fatalf("expected filename report.pdf, got %q", target.uploadedFilename)
	}
}

func TestTransferHandlerUsesPaperlessOriginalFilename(t *testing.T) {
	source := &transferTestService{originalFilename: "invoice.pdf"}
	target := &transferTestService{}
	controller := NewFileController(nil, nil)

	err, status := controller.TransferHandler(source, target, "user-provided-name", "", context.Background())
	if err != nil {
		t.Fatalf("expected transfer to succeed, got %v", err)
	}
	if status != 200 {
		t.Fatalf("expected status 200, got %d", status)
	}
	if target.uploadedFilename != "invoice.pdf" {
		t.Fatalf("expected filename invoice.pdf, got %q", target.uploadedFilename)
	}
}
