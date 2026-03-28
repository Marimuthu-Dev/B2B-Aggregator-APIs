package service

import (
	"context"
	"mime/multipart"
)

// BlobService handles MoU PDF validation and upload to object storage.
// Implementations live outside this package (e.g. internal/storage).
type BlobService interface {
	UploadOrReplace(ctx context.Context, file multipart.File, clientID int64) (string, error)
	ValidatePDF(fileHeader *multipart.FileHeader) error
}
