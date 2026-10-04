package backupservice

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// Controller exposes the HTTP API for creating and routing backups.
type Controller struct {
	sources   map[string]Source
	targets   map[string]Target
	processor *Service
	logger    *slog.Logger
}

// NewController creates a backup controller with the provided encryptor.
func NewController(logger *slog.Logger, encrypt Encryptor) *Controller {
	return &Controller{
		sources:   make(map[string]Source),
		targets:   make(map[string]Target),
		processor: New(encrypt),
		logger:    logger,
	}
}

// AddSource registers a backup source by name.
func (c *Controller) AddSource(name string, source Source) {
	c.sources[name] = source
}

// AddTarget registers a file integration as a backup target.
func (c *Controller) AddTarget(name string, target Target) {
	c.targets[name] = target
}

// HandleBackup creates, optionally encrypts, and uploads a backup.
func (c *Controller) HandleBackup(w http.ResponseWriter, r *http.Request) {
	started := time.Now()
	sourceName := r.URL.Query().Get("source")
	if sourceName == "" {
		http.Error(w, "source query parameter is required", http.StatusBadRequest)
		return
	}
	source, ok := c.sources[sourceName]
	if !ok {
		http.Error(w, fmt.Sprintf("backup source %q is not supported", sourceName), http.StatusBadRequest)
		return
	}

	targetName := r.URL.Query().Get("target")
	if targetName == "" {
		http.Error(w, "target query parameter is required", http.StatusBadRequest)
		return
	}
	target, ok := c.targets[targetName]
	if !ok {
		http.Error(w, fmt.Sprintf("target %q is not supported", targetName), http.StatusBadRequest)
		return
	}

	result, err := c.processor.Create(r.Context(), source, target, targetName, r.URL.Query().Get("encryption"))
	if err != nil {
		c.log(r, slog.LevelError, "backup failed", "source", sourceName, "target", targetName, "error", err)
		http.Error(w, fmt.Sprintf("failed to process backup: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{
		"status":   "ok",
		"filename": result.Filename,
		"source":   sourceName,
		"target":   result.Target,
	}); err != nil {
		c.log(r, slog.LevelError, "backup response encoding failed", "error", err)
		return
	}
	c.log(r, slog.LevelInfo, "backup completed", "source", sourceName, "target", targetName, "duration_ms", time.Since(started).Seconds()*1000)
}

// RegisterRoutes registers the authenticated backup endpoints.
func (c *Controller) RegisterRoutes(mux *http.ServeMux, authMiddleware func(http.Handler) http.Handler) {
	mux.Handle("GET /api/v1/backup", authMiddleware(http.HandlerFunc(c.HandleBackup)))
}

func (c *Controller) log(r *http.Request, level slog.Level, message string, args ...any) {
	if c.logger != nil {
		c.logger.Log(r.Context(), level, message, args...)
	}
}
