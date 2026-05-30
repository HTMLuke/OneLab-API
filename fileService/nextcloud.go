package fileService

import (
	"context"
	"fmt"
	"mime/multipart"
)

// NextcloudService handles sending files to Nextcloud.
type NextcloudService struct {
	apiURL   string
	username string
	password string
}

func NewNextcloudService(apiURL, username, password string) *NextcloudService {
	return &NextcloudService{
		apiURL:   apiURL,
		username: username,
		password: password,
	}
}

func (s *NextcloudService) TransferFile(ctx context.Context, file multipart.File, header *multipart.FileHeader) error {
	// TODO: Implement Nextcloud WebDAV or API upload logic here.
	fmt.Printf("Transferring file '%s' to Nextcloud at %s\n", header.Filename, s.apiURL)
	return nil
}
