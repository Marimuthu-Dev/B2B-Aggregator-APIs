package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/bloberror"
	"github.com/gabriel-vasile/mimetype"

	"b2b-diagnostic-aggregator/apis/internal/config"
)

const pdfContentType = "application/pdf"

// AzureMoUBlobService streams MoU and diagnostic-report PDFs to Azure Blob Storage.
type AzureMoUBlobService struct {
	client            *azblob.Client
	sharedKey         *azblob.SharedKeyCredential
	container         string
	reportsContainer  string
	mouMaxBytes       int64
	reportsMaxBytes   int64
	mouUploadTimeout  time.Duration
	reportsUploadTimeout time.Duration
	mouSASTTL         time.Duration
	reportsSASTTL     time.Duration
	endpointSuffix    string // e.g. core.windows.net — used for SAS URL host and signing consistency
	log               *slog.Logger
}

// NewAzureMoUBlobService creates a service client with SharedKeyCredential from AZURE_STORAGE_ACCOUNT /
// AZURE_STORAGE_KEY, or falls back to parsing AccountName/AccountKey from AZURE_STORAGE_CONNECTION_STRING.
// SAS tokens are always signed with SharedKeyCredential — never by passing a connection string into SAS APIs.
func NewAzureMoUBlobService(cfg config.AzureBlobConfig, log *slog.Logger) (*AzureMoUBlobService, error) {
	if strings.TrimSpace(cfg.ContainerName) == "" {
		return nil, errors.New("azure blob: empty container name")
	}
	acc := strings.ToLower(strings.TrimSpace(cfg.StorageAccountName))
	key := strings.TrimSpace(cfg.StorageAccountKey)
	if acc == "" || key == "" {
		var perr error
		acc, key, perr = ParseAzureConnectionString(strings.TrimSpace(cfg.ConnectionString))
		if perr != nil {
			return nil, fmt.Errorf("azure blob: set AZURE_STORAGE_ACCOUNT and AZURE_STORAGE_KEY, or fix AZURE_STORAGE_CONNECTION_STRING: %w", perr)
		}
		if acc == "" || key == "" {
			return nil, errors.New("azure blob: set AZURE_STORAGE_ACCOUNT and AZURE_STORAGE_KEY (or a connection string with AccountName and AccountKey)")
		}
		acc = strings.ToLower(strings.TrimSpace(acc))
		key = strings.TrimSpace(key)
	}
	suffix := strings.TrimSpace(cfg.BlobEndpointSuffix)
	if suffix == "" {
		suffix = "core.windows.net"
	}
	sk, err := azblob.NewSharedKeyCredential(acc, key)
	if err != nil {
		return nil, fmt.Errorf("azure blob: shared key credential: %w", err)
	}
	serviceURL := fmt.Sprintf("https://%s.blob.%s/", acc, suffix)
	cli, err := azblob.NewClientWithSharedKeyCredential(serviceURL, sk, nil)
	if err != nil {
		return nil, fmt.Errorf("azure blob: create client: %w", err)
	}
	mouSASTTL := cfg.MoUSASTTL
	if mouSASTTL <= 0 {
		mouSASTTL = 15 * time.Minute
	}
	reportsSASTTL := cfg.DiagnosticReportsSASTTL
	if reportsSASTTL <= 0 {
		reportsSASTTL = mouSASTTL
	}
	if log == nil {
		log = slog.Default()
	}
	reportsContainer := strings.TrimSpace(cfg.DiagnosticReportsContainer)
	if reportsContainer == "" {
		reportsContainer = "diagnostic-reports"
	}
	mouMax := cfg.MoUMaxBytes
	if mouMax <= 0 {
		mouMax = 5 * 1024 * 1024
	}
	reportMax := cfg.DiagnosticReportsMaxBytes
	if reportMax <= 0 {
		reportMax = mouMax
	}
	mouTimeout := cfg.UploadTimeout
	if mouTimeout <= 0 {
		mouTimeout = 60 * time.Second
	}
	reportTimeout := cfg.DiagnosticReportsUploadTimeout
	if reportTimeout <= 0 {
		reportTimeout = mouTimeout
	}
	s := &AzureMoUBlobService{
		client:               cli,
		sharedKey:            sk,
		container:            cfg.ContainerName,
		reportsContainer:     reportsContainer,
		mouMaxBytes:          mouMax,
		reportsMaxBytes:      reportMax,
		mouUploadTimeout:     mouTimeout,
		reportsUploadTimeout: reportTimeout,
		mouSASTTL:            mouSASTTL,
		reportsSASTTL:        reportsSASTTL,
		endpointSuffix:       suffix,
		log:                  log,
	}
	return s, nil
}

// EnsureContainer creates the MoU container if it does not exist.
func (s *AzureMoUBlobService) EnsureContainer(ctx context.Context) error {
	return s.ensureContainer(ctx, s.container)
}

func (s *AzureMoUBlobService) ensureContainer(ctx context.Context, container string) error {
	_, err := s.client.CreateContainer(ctx, container, nil)
	if err == nil {
		return nil
	}
	if bloberror.HasCode(err, bloberror.ContainerAlreadyExists) {
		return nil
	}
	return fmt.Errorf("azure blob: ensure container: %w", err)
}

func mouBlobNameClient(clientID int64) string {
	return fmt.Sprintf("client-%d-mou.pdf", clientID)
}

func mouBlobNameLab(labID int64) string {
	return fmt.Sprintf("lab-%d-mou.pdf", labID)
}

// ValidatePDF checks extension, declared size, and file content (magic / MIME) for MoU uploads.
func (s *AzureMoUBlobService) ValidatePDF(fh *multipart.FileHeader) error {
	return s.validatePDFWithMaxBytes(fh, s.mouMaxBytes)
}

// ValidateDiagnosticReportPDF is the same checks as ValidatePDF but uses DIAGNOSTIC_REPORTS_MAX_UPLOAD_BYTES.
func (s *AzureMoUBlobService) ValidateDiagnosticReportPDF(fh *multipart.FileHeader) error {
	return s.validatePDFWithMaxBytes(fh, s.reportsMaxBytes)
}

func (s *AzureMoUBlobService) validatePDFWithMaxBytes(fh *multipart.FileHeader, maxBytes int64) error {
	if fh == nil {
		return errors.New("no file provided")
	}
	if maxBytes <= 0 {
		maxBytes = 5 * 1024 * 1024
	}
	ext := strings.ToLower(strings.TrimSpace(filepath.Ext(fh.Filename)))
	if ext != ".pdf" {
		return errors.New("invalid file type. Only PDF allowed")
	}
	ct := strings.ToLower(strings.TrimSpace(fh.Header.Get("Content-Type")))
	if ct != "" && ct != pdfContentType && !strings.HasPrefix(ct, pdfContentType+";") &&
		ct != "application/octet-stream" && !strings.HasPrefix(ct, "application/octet-stream;") {
		// Postman / browsers often send octet-stream; magic-byte check below is authoritative.
		return errors.New("invalid file type. Only PDF allowed")
	}
	if fh.Size > 0 && fh.Size > maxBytes {
		return fmt.Errorf("file exceeds maximum size of %d bytes", maxBytes)
	}

	rc, err := fh.Open()
	if err != nil {
		return fmt.Errorf("open upload: %w", err)
	}
	defer func() { _ = rc.Close() }()

	limited := io.LimitReader(rc, maxBytes+1)
	buf, err := io.ReadAll(limited)
	if err != nil {
		return fmt.Errorf("read file for validation: %w", err)
	}
	if int64(len(buf)) > maxBytes {
		return fmt.Errorf("file exceeds maximum size of %d bytes", maxBytes)
	}
	if len(buf) < 4 || string(buf[0:4]) != "%PDF" {
		return errors.New("invalid file type. Only PDF allowed")
	}
	detected := mimetype.Detect(buf)
	if detected.String() != pdfContentType {
		return errors.New("invalid file type. Only PDF allowed")
	}
	return nil
}

// UploadOrReplace uploads or overwrites client-{id}-mou.pdf with Content-Type application/pdf.
// The caller must close the provided file after this returns.
func (s *AzureMoUBlobService) UploadOrReplace(ctx context.Context, file multipart.File, clientID int64) (string, error) {
	return s.uploadMoUBlob(ctx, file, mouBlobNameClient(clientID), "client", clientID)
}

// UploadOrReplaceLabMoU uploads or overwrites lab-{id}-mou.pdf with Content-Type application/pdf.
func (s *AzureMoUBlobService) UploadOrReplaceLabMoU(ctx context.Context, file multipart.File, labID int64) (string, error) {
	return s.uploadMoUBlob(ctx, file, mouBlobNameLab(labID), "lab", labID)
}

// UploadDiagnosticReportPDF uploads a lead blood-test report to the diagnostic-reports container.
// Blob path: {leadId}/{unixMillis}_{sanitizedOriginalFileName}.pdf
func (s *AzureMoUBlobService) UploadDiagnosticReportPDF(ctx context.Context, file multipart.File, leadID int64, originalFileName string) (string, error) {
	if file == nil {
		return "", errors.New("empty file reader")
	}
	safeName := sanitizeDiagnosticReportFileName(originalFileName)
	blobName := fmt.Sprintf("%d/%d_%s", leadID, time.Now().UTC().UnixMilli(), safeName)
	return s.uploadPDFStream(ctx, s.reportsContainer, blobName, file, "lead-report", leadID, s.reportsUploadTimeout)
}

var diagnosticReportFileSanitizer = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

func sanitizeDiagnosticReportFileName(original string) string {
	base := strings.TrimSpace(filepath.Base(original))
	base = diagnosticReportFileSanitizer.ReplaceAllString(base, "_")
	base = strings.Trim(base, "._-")
	if base == "" {
		return "report.pdf"
	}
	if !strings.HasSuffix(strings.ToLower(base), ".pdf") {
		base += ".pdf"
	}
	const maxLen = 200
	if len(base) > maxLen {
		base = base[:maxLen-len(".pdf")] + ".pdf"
	}
	return base
}

func (s *AzureMoUBlobService) uploadMoUBlob(ctx context.Context, file multipart.File, blobName, entity string, entityID int64) (string, error) {
	return s.uploadPDFStream(ctx, s.container, blobName, file, entity, entityID, s.mouUploadTimeout)
}

func (s *AzureMoUBlobService) uploadPDFStream(ctx context.Context, container, blobName string, file multipart.File, entity string, entityID int64, uploadTimeout time.Duration) (string, error) {
	if file == nil {
		return "", errors.New("empty file reader")
	}
	if uploadTimeout <= 0 {
		uploadTimeout = 60 * time.Second
	}

	if err := s.ensureContainer(ctx, container); err != nil {
		s.log.Error("blob ensure container failed",
			slog.String("container", container),
			slog.Any("err", err),
			slog.String("azure_error", formatAzureError(err)),
		)
		return "", err
	}

	uploadBase, cancel := context.WithTimeout(ctx, uploadTimeout)
	defer cancel()

	var lastErr error
	backoffs := []time.Duration{0, 200 * time.Millisecond, 600 * time.Millisecond}
attempts:
	for i, d := range backoffs {
		if d > 0 {
			select {
			case <-time.After(d):
			case <-uploadBase.Done():
				return "", uploadBase.Err()
			}
			seeker, ok := file.(io.Seeker)
			if !ok {
				break attempts
			}
			if _, err := seeker.Seek(0, io.SeekStart); err != nil {
				return "", fmt.Errorf("rewind file for retry: %w", err)
			}
		}

		_, err := s.client.UploadStream(uploadBase, container, blobName, file, &azblob.UploadStreamOptions{
			HTTPHeaders: &blob.HTTPHeaders{
				BlobContentType: ptr(pdfContentType),
			},
		})
		if err == nil {
			u := s.client.ServiceClient().NewContainerClient(container).NewBlobClient(blobName).URL()
			s.log.Info("pdf blob uploaded", slog.String("blob", blobName), slog.String("container", container), slog.String("entity", entity), slog.Int64("entityID", entityID))
			return u, nil
		}
		lastErr = err
		if !isRetriableBlobError(err) || i == len(backoffs)-1 {
			break
		}
		s.log.Warn("pdf blob upload retry", slog.Int("attempt", i+1), slog.Any("err", err))
	}
	s.log.Error("pdf blob upload failed",
		slog.String("blob", blobName),
		slog.String("entity", entity),
		slog.Int64("entityID", entityID),
		slog.String("container", container),
		slog.Any("err", lastErr),
		slog.String("azure_error", formatAzureError(lastErr)),
	)
	return "", fmt.Errorf("azure blob upload: %w", lastErr)
}

// MoUDownloadSASURL returns an HTTPS URL with read-only SAS query parameters for client-{clientID}-mou.pdf.
func (s *AzureMoUBlobService) MoUDownloadSASURL(ctx context.Context, clientID int64) (string, time.Time, error) {
	_ = ctx
	return s.sasURLForBlobInContainer(s.container, mouBlobNameClient(clientID), s.mouSASTTL)
}

// LabMoUDownloadSASURL returns an HTTPS URL with read-only SAS query parameters for lab-{labID}-mou.pdf.
func (s *AzureMoUBlobService) LabMoUDownloadSASURL(ctx context.Context, labID int64) (string, time.Time, error) {
	_ = ctx
	return s.sasURLForBlobInContainer(s.container, mouBlobNameLab(labID), s.mouSASTTL)
}

// DiagnosticReportDownloadSASURL returns a read-only SAS URL for a blob in the given container (uses DIAGNOSTIC_REPORTS_SAS_TTL_MINUTES).
func (s *AzureMoUBlobService) DiagnosticReportDownloadSASURL(ctx context.Context, container, blobName string) (string, time.Time, error) {
	_ = ctx
	if strings.TrimSpace(container) == "" || strings.TrimSpace(blobName) == "" {
		return "", time.Time{}, errors.New("azure blob: container and blob name required")
	}
	return s.sasURLForBlobInContainer(container, blobName, s.reportsSASTTL)
}

// DownloadBlob reads an entire blob into memory (used by the fitness certificate worker).
func (s *AzureMoUBlobService) DownloadBlob(ctx context.Context, container, blobName string) ([]byte, error) {
	if strings.TrimSpace(container) == "" || strings.TrimSpace(blobName) == "" {
		return nil, errors.New("azure blob: container and blob name required")
	}
	resp, err := s.client.DownloadStream(ctx, container, blobName, nil)
	if err != nil {
		return nil, fmt.Errorf("azure blob download: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("azure blob read body: %w", err)
	}
	if int64(len(data)) > s.reportsMaxBytes*10 {
		s.log.Warn("downloaded blob exceeds soft limit", slog.Int("bytes", len(data)), slog.String("blob", blobName))
	}
	return data, nil
}

// UploadDiagnosticReportPDFBytes overwrites an existing diagnostic report blob (same container/path as stored ReportURL).
func (s *AzureMoUBlobService) UploadDiagnosticReportPDFBytes(ctx context.Context, container, blobName string, pdf []byte) error {
	if len(pdf) == 0 {
		return errors.New("azure blob: empty pdf")
	}
	// Merged report (certificate + original) can exceed single-upload limit.
	const mergeSizeMultiplier int64 = 5
	if int64(len(pdf)) > s.reportsMaxBytes*mergeSizeMultiplier {
		return fmt.Errorf("azure blob: pdf exceeds maximum size %d bytes", s.reportsMaxBytes*mergeSizeMultiplier)
	}
	if uploadTimeout := s.reportsUploadTimeout; uploadTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, uploadTimeout)
		defer cancel()
	}
	_, err := s.client.UploadBuffer(ctx, container, blobName, pdf, &azblob.UploadBufferOptions{
		HTTPHeaders: &blob.HTTPHeaders{
			BlobContentType: ptr(pdfContentType),
		},
	})
	if err != nil {
		return fmt.Errorf("azure blob upload: %w", err)
	}
	s.log.Info("diagnostic report pdf overwritten", slog.String("container", container), slog.String("blob", blobName))
	return nil
}

// DiagnosticReportDownloadSASFromStoredURL parses a blob HTTPS URL from tbl_Leads.ReportURL and returns a read-only SAS link.
func (s *AzureMoUBlobService) DiagnosticReportDownloadSASFromStoredURL(ctx context.Context, reportBlobURL string) (string, time.Time, error) {
	_ = ctx
	container, blobName, err := ParseAzureBlobContainerAndBlob(reportBlobURL)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("azure blob: parse report url: %w", err)
	}
	return s.DiagnosticReportDownloadSASURL(ctx, container, blobName)
}

func (s *AzureMoUBlobService) sasURLForBlobInContainer(container, blobName string, ttl time.Duration) (string, time.Time, error) {
	if s.sharedKey == nil {
		return "", time.Time{}, errors.New("azure blob: SAS signing unavailable")
	}
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}
	return buildBlobReadOnlySASURLRelativeToNow(s.sharedKey, s.endpointSuffix, container, blobName, ttl, s.log)
}

func formatAzureError(err error) string {
	if err == nil {
		return ""
	}
	var re *azcore.ResponseError
	if errors.As(err, &re) {
		return fmt.Sprintf("status=%d code=%s", re.StatusCode, re.ErrorCode)
	}
	return err.Error()
}

func isRetriableBlobError(err error) bool {
	var re *azcore.ResponseError
	if errors.As(err, &re) {
		if re.StatusCode == 429 || re.StatusCode >= 500 {
			return true
		}
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	return false
}

func ptr[T any](v T) *T {
	return &v
}
