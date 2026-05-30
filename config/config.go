package config

import (
	"os"
)

// ConfigService defines the interface for retrieving configuration values.
type ConfigService interface {
	GetNextcloudBaseUrl() string
	GetPaperlessBaseUrl() string
	GetNextcloudUser() string
	GetNextcloudPassword() string
	GetPaperlessToken() string
}

// AppConfig holds the internal representation of the configuration.
type AppConfig struct {
	NextcloudBaseUrl  string `json:"-"`
	PaperlessBaseUrl  string `json:"-"`
	NextcloudUser     string `json:"-"` // Not in config.json
	NextcloudPassword string `json:"-"` // Not in config.json
	PaperlessToken    string `json:"-"` // Not in config.json
}

type configService struct {
	cfg *AppConfig
}

// NewConfigService initializes a new configuration service.
func NewConfigService() (ConfigService, error) {
	// Set safe default values
	cfg := &AppConfig{
		NextcloudBaseUrl: "http://nextcloud.local/",
		PaperlessBaseUrl: "http://paperless.local/",
	}

	if url := os.Getenv("ONELAB_NEXTCLOUD_BASE_URL"); url != "" {
		cfg.NextcloudBaseUrl = url
	}
	if url := os.Getenv("ONELAB_PAPERLESS_BASE_URL"); url != "" {
		cfg.PaperlessBaseUrl = url
	}

	// 3. Load Credentials directly from Environment Variables
	cfg.NextcloudUser = os.Getenv("ONELAB_NEXTCLOUD_USER")
	cfg.NextcloudPassword = os.Getenv("ONELAB_NEXTCLOUD_PASSWORD")
	cfg.PaperlessToken = os.Getenv("ONELAB_PAPERLESS_TOKEN")

	return &configService{cfg: cfg}, nil
}

func (s *configService) GetNextcloudBaseUrl() string  { return s.cfg.NextcloudBaseUrl }
func (s *configService) GetPaperlessBaseUrl() string  { return s.cfg.PaperlessBaseUrl }
func (s *configService) GetNextcloudUser() string     { return s.cfg.NextcloudUser }
func (s *configService) GetNextcloudPassword() string { return s.cfg.NextcloudPassword }
func (s *configService) GetPaperlessToken() string    { return s.cfg.PaperlessToken }
