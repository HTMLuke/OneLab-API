package config

import (
	"encoding/json"
	"os"
)

// ConfigService defines the interface for retrieving configuration values.
type ConfigService interface {
	GetNextcloudURL() string
	GetPaperlessURL() string
	GetNextcloudUser() string
	GetNextcloudPassword() string
	GetPaperlessToken() string
}

// AppConfig holds the internal representation of the configuration.
type AppConfig struct {
	NextcloudURL      string `json:"nextcloudUrl"`
	PaperlessURL      string `json:"paperlessUrl"`
	NextcloudUser     string `json:"-"` // Not in config.json
	NextcloudPassword string `json:"-"` // Not in config.json
	PaperlessToken    string `json:"-"` // Not in config.json
}

type configService struct {
	cfg *AppConfig
}

// NewConfigService initializes a new configuration service.
func NewConfigService(filePath string) (ConfigService, error) {
	// Set safe default values
	cfg := &AppConfig{
		NextcloudURL: "http://nextcloud.local/remote.php/webdav/",
		PaperlessURL: "http://paperless.local/api/documents/post_document/",
	}

	// 1. Try to read from the provided configuration file (e.g., config.json)
	if file, err := os.Open(filePath); err == nil {
		defer file.Close()
		json.NewDecoder(file).Decode(cfg)
	}

	// 2. Allow environment variable overrides.
	if url := os.Getenv("ONELAB_NEXTCLOUD_URL"); url != "" {
		cfg.NextcloudURL = url
	}
	if url := os.Getenv("ONELAB_PAPERLESS_URL"); url != "" {
		cfg.PaperlessURL = url
	}

	// 3. Load Credentials directly from Environment Variables
	cfg.NextcloudUser = os.Getenv("ONELAB_NEXTCLOUD_USER")
	cfg.NextcloudPassword = os.Getenv("ONELAB_NEXTCLOUD_PASSWORD")
	cfg.PaperlessToken = os.Getenv("ONELAB_PAPERLESS_TOKEN")

	return &configService{cfg: cfg}, nil
}

func (s *configService) GetNextcloudURL() string      { return s.cfg.NextcloudURL }
func (s *configService) GetPaperlessURL() string      { return s.cfg.PaperlessURL }
func (s *configService) GetNextcloudUser() string     { return s.cfg.NextcloudUser }
func (s *configService) GetNextcloudPassword() string { return s.cfg.NextcloudPassword }
func (s *configService) GetPaperlessToken() string    { return s.cfg.PaperlessToken }
