package main

import (
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/HTMLuke/OneLab-API/auth"
	backupservice "github.com/HTMLuke/OneLab-API/backupService"
	encryptionservice "github.com/HTMLuke/OneLab-API/encryptionService"
	"github.com/HTMLuke/OneLab-API/fileService"
	"github.com/HTMLuke/OneLab-API/secretProvider"
)

// registerIntegrations wires every integration whose environment configuration is complete.
// Integrations register themselves from their own files via their init()
func registerIntegrations(s secretProvider.SecretService, fc *fileService.FileController, bc *backupservice.Controller, ac *auth.AuthController, ec *encryptionservice.Controller, logger *slog.Logger) {
	for name, build := range fileService.Builders() {
		if !integrationEnabled(name, logger) {
			continue
		}
		svc, err := build(serviceBaseURL(name), s, logger)
		if err != nil {
			logger.Warn("service integration skipped", "integration", name, "reason", err)
			continue
		}
		fc.AddIntegration(name, svc)
		if target, ok := svc.(backupservice.Target); ok {
			bc.AddTarget(name, target)
		}
		if name == "paperless" {
			bc.AddSource(name, backupservice.NewPaperlessService(logger))
		}
		logger.Info("service integration enabled", "integration", name)
	}

	for name, build := range auth.Builders() {
		if !integrationEnabled(name, logger) {
			continue
		}
		svc, err := build(authExpiry(logger), s, logger)
		if err != nil {
			logger.Warn("auth integration skipped", "integration", name, "reason", err)
			continue
		}
		ac.AddIntegration(name, svc)
		logger.Info("auth integration enabled", "integration", name)
	}

	for name, build := range encryptionservice.Builders() {
		if !integrationEnabled(name, logger) {
			continue
		}
		svc, err := build(s, logger)
		if err != nil {
			logger.Warn("encryption integration skipped", "integration", name, "reason", err)
			continue
		}
		ec.AddIntegration(name, svc)
		logger.Info("encryption integration enabled", "integration", name)
	}
}

func integrationEnabled(name string, logger *slog.Logger) bool {
	key := "ONELAB_" + envName(name) + "_ENABLED"
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return true
	}
	enabled, err := strconv.ParseBool(value)
	if err != nil {
		if logger != nil {
			logger.Warn("invalid integration enable flag; integration disabled", "integration", name, "env", key, "value", value)
		}
		return false
	}
	return enabled
}

func serviceBaseURL(name string) string {
	envName := envName(name)
	for _, key := range []string{
		"ONELAB_" + envName + "_BASE_URL",
		"ONELAB_" + envName + "_URL",
	} {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value
		}
	}
	return ""
}

func authExpiry(logger *slog.Logger) time.Duration {
	const defaultHours = 6
	value := strings.TrimSpace(os.Getenv("ONELAB_AUTH_EXPIRY_HOURS"))
	if value == "" {
		return defaultHours * time.Hour
	}
	hours, err := strconv.Atoi(value)
	if err != nil || hours <= 0 {
		logger.Warn("invalid auth expiry; using default", "env", "ONELAB_AUTH_EXPIRY_HOURS", "value", value, "default_hours", defaultHours)
		return defaultHours * time.Hour
	}
	return time.Duration(hours) * time.Hour
}

func envName(name string) string {
	return strings.ToUpper(strings.NewReplacer("-", "_", " ", "_", ".", "_").Replace(name))
}
