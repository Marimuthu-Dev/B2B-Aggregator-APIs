package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/bloberror"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/sas"
	"github.com/gabriel-vasile/mimetype"

	"b2b-diagnostic-aggregator/apis/internal/config"
)

const pdfContentType = "application/pdf"

// AzureMoUBlobService streams MoU PDFs to Azure Blob Storage using a fixed blob name per client.
type AzureMoUBlobService struct {
	client        *azblob.Client
	sharedKey     *azblob.SharedKeyCredential
	accountName   string
	container     string
	maxBytes      int64
	uploadTimeout time.Duration
	sasTTL        time.Duration
	log           *slog.Logger
}

// NewAzureMoUBlobService builds a blob client from the connection string and ensures the container exists.
func NewAzureMoUBlobService(cfg config.AzureBlobConfig, log *slog.Logger) (*AzureMoUBlobService, error) {
	if strings.TrimSpace(cfg.ConnectionString) == "" {
		return nil, errors.New("azure blob: empty connection string")
	}
	if strings.TrimSpace(cfg.ContainerName) == "" {
		return nil, errors.New("azure blob: empty container name")
	}
	cli, err := azblob.NewClientFromConnectionString(cfg.ConnectionString, nil)
	if err != nil {
		return nil, fmt.Errorf("azure blob: create client: %w", err)
	}
	acc, key, err := parseAzureConnectionString(cfg.ConnectionString)
	if err != nil {
		return nil, fmt.Errorf("azure blob: connection string: %w", err)
	}
	sk, err := azblob.NewSharedKeyCredential(acc, key)
	if err != nil {
		return nil, fmt.Errorf("azure blob: shared key credential: %w", err)
	}
	sasTTL := cfg.MoUSASTTL
	if sasTTL <= 0 {
		sasTTL = 15 * time.Minute
	}
	if log == nil {
		log = slog.Default()
	}
	s := &AzureMoUBlobService{
		client:        cli,
		sharedKey:     sk,
		accountName:   acc,
		container:     cfg.ContainerName,
		maxBytes:      cfg.MoUMaxBytes,
		uploadTimeout: cfg.UploadTimeout,
		sasTTL:        sasTTL,
		log:           log,
	}
	return s, nil
}

// EnsureContainer creates the container if it does not exist.
func (s *AzureMoUBlobService) EnsureContainer(ctx context.Context) error {
	_, err := s.client.CreateContainer(ctx, s.container, nil)
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

// ValidatePDF checks extension, declared size, and file content (magic / MIME).
func (s *AzureMoUBlobService) ValidatePDF(fh *multipart.FileHeader) error {
	if fh == nil {
		return errors.New("no file provided")
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
	if fh.Size > 0 && fh.Size > s.maxBytes {
		return fmt.Errorf("file exceeds maximum size of %d bytes", s.maxBytes)
	}

	rc, err := fh.Open()
	if err != nil {
		return fmt.Errorf("open upload: %w", err)
	}
	defer rc.Close()

	limited := io.LimitReader(rc, s.maxBytes+1)
	buf, err := io.ReadAll(limited)
	if err != nil {
		return fmt.Errorf("read file for validation: %w", err)
	}
	if int64(len(buf)) > s.maxBytes {
		return fmt.Errorf("file exceeds maximum size of %d bytes", s.maxBytes)
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

func (s *AzureMoUBlobService) uploadMoUBlob(ctx context.Context, file multipart.File, blobName, entity string, entityID int64) (string, error) {
	if file == nil {
		return "", errors.New("empty file reader")
	}

	if err := s.EnsureContainer(ctx); err != nil {
		s.log.Error("mou blob ensure container failed",
			slog.String("container", s.container),
			slog.Any("err", err),
			slog.String("azure_error", formatAzureError(err)),
		)
		return "", err
	}

	uploadBase, cancel := context.WithTimeout(ctx, s.uploadTimeout)
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

		_, err := s.client.UploadStream(uploadBase, s.container, blobName, file, &azblob.UploadStreamOptions{
			HTTPHeaders: &blob.HTTPHeaders{
				BlobContentType: ptr(pdfContentType),
			},
		})
		if err == nil {
			u := s.client.ServiceClient().NewContainerClient(s.container).NewBlobClient(blobName).URL()
			s.log.Info("mou blob uploaded", slog.String("blob", blobName), slog.String("entity", entity), slog.Int64("entityID", entityID))
			return u, nil
		}
		lastErr = err
		if !isRetriableBlobError(err) || i == len(backoffs)-1 {
			break
		}
		s.log.Warn("mou blob upload retry", slog.Int("attempt", i+1), slog.Any("err", err))
	}
	s.log.Error("mou blob upload failed",
		slog.String("blob", blobName),
		slog.String("entity", entity),
		slog.Int64("entityID", entityID),
		slog.String("container", s.container),
		slog.Any("err", lastErr),
		slog.String("azure_error", formatAzureError(lastErr)),
	)
	return "", fmt.Errorf("azure blob upload: %w", lastErr)
}

// MoUDownloadSASURL returns an HTTPS URL with read-only SAS query parameters for client-{clientID}-mou.pdf.
func (s *AzureMoUBlobService) MoUDownloadSASURL(ctx context.Context, clientID int64) (string, time.Time, error) {
	_ = ctx
	return s.sasURLForBlob(mouBlobNameClient(clientID))
}

// LabMoUDownloadSASURL returns an HTTPS URL with read-only SAS query parameters for lab-{labID}-mou.pdf.
func (s *AzureMoUBlobService) LabMoUDownloadSASURL(ctx context.Context, labID int64) (string, time.Time, error) {
	_ = ctx
	return s.sasURLForBlob(mouBlobNameLab(labID))
}

func (s *AzureMoUBlobService) sasURLForBlob(blobName string) (string, time.Time, error) {
	if s.sharedKey == nil || s.accountName == "" {
		return "", time.Time{}, errors.New("azure blob: SAS signing unavailable")
	}
	expires := time.Now().UTC().Add(s.sasTTL)
	start := time.Now().UTC().Add(-2 * time.Minute)
	qp, err := sas.BlobSignatureValues{
		Protocol:      sas.ProtocolHTTPS,
		StartTime:     start,
		ExpiryTime:    expires,
		Permissions:   (&sas.BlobPermissions{Read: true}).String(),
		ContainerName: s.container,
		BlobName:      blobName,
	}.SignWithSharedKey(s.sharedKey)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("azure blob: sign SAS: %w", err)
	}
	u := fmt.Sprintf("https://%s.blob.core.windows.net/%s/%s?%s",
		s.accountName,
		url.PathEscape(s.container),
		url.PathEscape(blobName),
		qp.Encode())
	return u, expires, nil
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
