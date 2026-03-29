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
	UploadOrReplaceLabMoU(ctx context.Context, file multipart.File, labID int64) (string, error)
	// UploadDiagnosticReportPDF stores a lead blood-test report PDF under diagnostic-reports/{leadId}/...
	UploadDiagnosticReportPDF(ctx context.Context, file multipart.File, leadID int64, originalFileName string) (url string, err error)
	ValidatePDF(fileHeader *multipart.FileHeader) error
	// ValidateDiagnosticReportPDF enforces DIAGNOSTIC_REPORTS_MAX_UPLOAD_BYTES (and PDF rules).
	ValidateDiagnosticReportPDF(fileHeader *multipart.FileHeader) error
	// DiagnosticReportDownloadSASURL builds a read-only SAS URL (DIAGNOSTIC_REPORTS_SAS_TTL_MINUTES).
	DiagnosticReportDownloadSASURL(ctx context.Context, container, blobName string) (url string, expiresAt time.Time, err error)
	// DiagnosticReportDownloadSASFromStoredURL parses a full blob HTTPS URL then signs it (same SAS TTL as above).
	DiagnosticReportDownloadSASFromStoredURL(ctx context.Context, reportBlobURL string) (url string, expiresAt time.Time, err error)
	// MoUDownloadSASURL returns a time-limited read-only URL for the client's MoU blob.
	MoUDownloadSASURL(ctx context.Context, clientID int64) (url string, expiresAt time.Time, err error)
	// LabMoUDownloadSASURL returns a time-limited read-only URL for the lab's MoU blob.
	LabMoUDownloadSASURL(ctx context.Context, labID int64) (url string, expiresAt time.Time, err error)
}
