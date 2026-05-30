package fileService

import (
	"context"
	"fmt"
	"mime/multipart"
)

// PaperlessService handles sending files to Paperless-ngx.
type PaperlessService struct {
	apiURL string
	token  string
}

func NewPaperlessService(apiURL, token string) *PaperlessService {
	return &PaperlessService{
		apiURL: apiURL,
		token:  token,
	}
}

func (s *PaperlessService) TransferFile(ctx context.Context, file multipart.File, header *multipart.FileHeader) error {
	// TODO: Implement Paperless-ngx document consumption logic here.
	fmt.Printf("Transferring file '%s' to Paperless-ngx at %s\n", header.Filename, s.apiURL)
	return nil
}
