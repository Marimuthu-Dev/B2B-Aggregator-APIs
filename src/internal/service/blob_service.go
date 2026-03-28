package service

import (
	"context"
	"mime/multipart"
	"time"
)

// BlobService handles MoU PDF validation and upload to object storage.
// Implementations live outside this package (e.g. internal/storage).
type BlobService interface {
	UploadOrReplace(ctx context.Context, file multipart.File, clientID int64) (string, error)
	ValidatePDF(fileHeader *multipart.FileHeader) error
	// MoUDownloadSASURL returns a time-limited read-only URL for the client's MoU blob.
	MoUDownloadSASURL(ctx context.Context, clientID int64) (url string, expiresAt time.Time, err error)
}
