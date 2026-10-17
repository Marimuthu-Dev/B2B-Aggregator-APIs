package service

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/textproto"
	"testing"

	"b2b-diagnostic-aggregator/apis/internal/apperrors"
	"b2b-diagnostic-aggregator/apis/internal/domain"
	"b2b-diagnostic-aggregator/apis/internal/repository"
)

type mockLeadRepoForUpload struct {
	repository.LeadRepository
	lead               *domain.Lead
	findErr            error
	updatedURL         string
	updatedStatusID    int8
	updatedUserID      int64
	activeStatusID     int8
	activeStatusName   string
	activeStatusIDErr  error
	updateReportURLErr error
}

func (m *mockLeadRepoForUpload) FindByID(id int64) (*domain.Lead, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	return m.lead, nil
}

func (m *mockLeadRepoForUpload) FindActiveLeadStatusIDByName(name string) (int8, error) {
	if m.activeStatusIDErr != nil {
		return 0, m.activeStatusIDErr
	}
	return m.activeStatusID, nil
}

func (m *mockLeadRepoForUpload) UpdateLeadReportURLAndStatus(leadID int64, reportURL string, statusID int8, userID int64) error {
	if m.updateReportURLErr != nil {
		return m.updateReportURLErr
	}
	m.updatedURL = reportURL
	m.updatedStatusID = statusID
	m.updatedUserID = userID
	return nil
}

type mockLeadHistoryRepoForUpload struct {
	repository.LeadHistoryRepository
	logged []*domain.LeadHistory
}

func (m *mockLeadHistoryRepoForUpload) LogAction(h *domain.LeadHistory) error {
	m.logged = append(m.logged, h)
	return nil
}

type mockLeadUowForUpload struct {
	leadRepo    repository.LeadRepository
	historyRepo repository.LeadHistoryRepository
}

func (u *mockLeadUowForUpload) WithinTransaction(fn func(repository.LeadRepository, repository.LeadHistoryRepository) error) error {
	return fn(u.leadRepo, u.historyRepo)
}

type mockBlobServiceForUpload struct {
	BlobService
	uploadURL string
	uploadErr error
}

func (b *mockBlobServiceForUpload) ValidateDiagnosticReportPDF(fh *multipart.FileHeader) error {
	return nil
}

func (b *mockBlobServiceForUpload) UploadDiagnosticReportPDF(ctx context.Context, file multipart.File, leadID int64, originalFileName string) (string, error) {
	if b.uploadErr != nil {
		return "", b.uploadErr
	}
	return b.uploadURL, nil
}

func createTestFileHeader(t *testing.T, filename string, content []byte) *multipart.FileHeader {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="file"; filename="`+filename+`"`)
	h.Set("Content-Type", "application/pdf")
	part, err := w.CreatePart(h)
	if err != nil {
		t.Fatalf("failed to create part: %v", err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatalf("failed to write part: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("failed to close writer: %v", err)
	}

	reader := multipart.NewReader(&buf, w.Boundary())
	form, err := reader.ReadForm(int64(buf.Len()) + 1024)
	if err != nil {
		t.Fatalf("failed to read form: %v", err)
	}
	files := form.File["file"]
	if len(files) == 0 {
		t.Fatalf("no file header returned")
	}
	return files[0]
}

func TestUploadBloodTestReport_OverwriteAllowedWhenStatusReportUploaded(t *testing.T) {
	fh := createTestFileHeader(t, "revised_report.pdf", []byte("%PDF-1.4 test report"))

	leadRepo := &mockLeadRepoForUpload{
		lead: &domain.Lead{
			LeadID:       75,
			LeadStatusID: domain.LeadStatusIDReportUploaded, // 8
			ReportURL:    "https://storage.blob.core.windows.net/reports/75/old_report.pdf",
		},
		activeStatusID: domain.LeadStatusIDReportUploaded,
	}
	histRepo := &mockLeadHistoryRepoForUpload{}
	uow := &mockLeadUowForUpload{leadRepo: leadRepo, historyRepo: histRepo}
	blob := &mockBlobServiceForUpload{
		uploadURL: "https://storage.blob.core.windows.net/reports/75/revised_report.pdf",
	}

	svc := &leadService{
		repo:  leadRepo,
		uow:   uow,
		blobs: blob,
	}

	url, overwritten, err := svc.UploadBloodTestReport(context.Background(), 75, 1, fh)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !overwritten {
		t.Fatalf("expected overwritten to be true when lead is in Report Uploaded status with existing report, got false")
	}
	if url != "https://storage.blob.core.windows.net/reports/75/revised_report.pdf" {
		t.Fatalf("unexpected report URL: %s", url)
	}
	if leadRepo.updatedURL != url {
		t.Fatalf("expected repo to be updated with new URL %s, got %s", url, leadRepo.updatedURL)
	}
	if leadRepo.updatedStatusID != domain.LeadStatusIDReportUploaded {
		t.Fatalf("expected status ID %d, got %d", domain.LeadStatusIDReportUploaded, leadRepo.updatedStatusID)
	}
	if len(histRepo.logged) != 1 || histRepo.logged[0].Action != domain.LeadActionReportUploaded {
		t.Fatalf("expected 1 history entry with action 'Report Uploaded'")
	}
}

func TestUploadBloodTestReport_FreshUpload(t *testing.T) {
	fh := createTestFileHeader(t, "fresh_report.pdf", []byte("%PDF-1.4 test report"))

	leadRepo := &mockLeadRepoForUpload{
		lead: &domain.Lead{
			LeadID:       76,
			LeadStatusID: 6, // Sample Collected
			ReportURL:    "",
		},
		activeStatusID: domain.LeadStatusIDReportUploaded,
	}
	histRepo := &mockLeadHistoryRepoForUpload{}
	uow := &mockLeadUowForUpload{leadRepo: leadRepo, historyRepo: histRepo}
	blob := &mockBlobServiceForUpload{
		uploadURL: "https://storage.blob.core.windows.net/reports/76/fresh_report.pdf",
	}

	svc := &leadService{
		repo:  leadRepo,
		uow:   uow,
		blobs: blob,
	}

	url, overwritten, err := svc.UploadBloodTestReport(context.Background(), 76, 1, fh)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if overwritten {
		t.Fatalf("expected overwritten to be false for fresh upload, got true")
	}
	if url != "https://storage.blob.core.windows.net/reports/76/fresh_report.pdf" {
		t.Fatalf("unexpected report URL: %s", url)
	}
	if leadRepo.updatedStatusID != domain.LeadStatusIDReportUploaded {
		t.Fatalf("expected status ID %d, got %d", domain.LeadStatusIDReportUploaded, leadRepo.updatedStatusID)
	}
}

func TestUploadBloodTestReport_ConflictWhenLeadPastReportUploaded(t *testing.T) {
	fh := createTestFileHeader(t, "report.pdf", []byte("%PDF-1.4 test report"))

	statuses := []int8{
		domain.LeadStatusIDReportApproved,          // 9
		domain.LeadStatusIDReportReadyForDownload, // 10
		domain.LeadStatusIDReportDownloaded,       // 11
	}

	for _, st := range statuses {
		leadRepo := &mockLeadRepoForUpload{
			lead: &domain.Lead{
				LeadID:       77,
				LeadStatusID: st,
				ReportURL:    "https://storage.blob.core.windows.net/reports/77/old.pdf",
			},
			activeStatusID: domain.LeadStatusIDReportUploaded,
		}
		histRepo := &mockLeadHistoryRepoForUpload{}
		uow := &mockLeadUowForUpload{leadRepo: leadRepo, historyRepo: histRepo}
		blob := &mockBlobServiceForUpload{}

		svc := &leadService{
			repo:  leadRepo,
			uow:   uow,
			blobs: blob,
		}

		_, _, err := svc.UploadBloodTestReport(context.Background(), 77, 1, fh)
		if err == nil {
			t.Fatalf("expected conflict error for status %d, got nil", st)
		}
		appErr, ok := err.(*apperrors.AppError)
		if !ok || appErr.Kind != apperrors.KindConflict {
			t.Fatalf("expected conflict AppError for status %d, got %v", st, err)
		}
	}
}
