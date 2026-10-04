package backupservice

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// Source creates a backup and returns its local path.
type Source interface {
	Backup(ctx context.Context) (string, error)
}

// Target accepts a backup using the normal file integration contract.
type Target interface {
	AddFile(ctx context.Context, file []byte, filename string) error
}

// Encryptor encrypts backup contents using a named encryption method.
type Encryptor func(method string, file []byte) ([]byte, error)

// Result describes the generated backup and its optional target.
type Result struct {
	Path     string
	Filename string
	Target   string
}

// Service processes backups, including encryption, upload, and cleanup.
type Service struct {
	encrypt Encryptor
}

// New creates a backup processing service.
func New(encrypt Encryptor) *Service {
	return &Service{encrypt: encrypt}
}

// Create generates a backup, optionally encrypts it, uploads it, and cleans up
// the local file after a successful upload.
func (s *Service) Create(ctx context.Context, source Source, target Target, targetName, encryption string) (Result, error) {
	path, err := source.Backup(ctx)
	if err != nil {
		return Result{}, fmt.Errorf("create backup: %w", err)
	}

	backup, err := os.ReadFile(path)
	if err != nil {
		return Result{}, fmt.Errorf("read backup: %w", err)
	}

	filename := filepath.Base(path)
	if encryption != "" {
		if s == nil || s.encrypt == nil {
			return Result{}, fmt.Errorf("encryption provider is not configured")
		}
		backup, err = s.encrypt(encryption, backup)
		if err != nil {
			return Result{}, fmt.Errorf("encrypt backup with %s: %w", encryption, err)
		}
		filename += ".pgp"
	}

	if target != nil {
		if err := target.AddFile(ctx, backup, filename); err != nil {
			return Result{}, fmt.Errorf("upload backup to %s: %w", targetName, err)
		}
		// The local copy is only needed until the target confirms a successful upload.
		if err := os.Remove(path); err != nil {
			return Result{}, fmt.Errorf("remove local backup after upload: %w", err)
		}
	}

	return Result{
		Path:     path,
		Filename: filename,
		Target:   targetName,
	}, nil
}
