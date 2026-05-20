package service

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"log/slog"
	"mime/multipart"
	"strconv"
	"strings"
	"time"

	"b2b-diagnostic-aggregator/apis/internal/apperrors"
	"b2b-diagnostic-aggregator/apis/internal/domain"
	"b2b-diagnostic-aggregator/apis/internal/dto"
	"b2b-diagnostic-aggregator/apis/internal/repository"
	"b2b-diagnostic-aggregator/apis/internal/timeutil"
	"b2b-diagnostic-aggregator/apis/pkg/utils"

	"gorm.io/gorm"
)

type LeadService interface {
	ListLeads(filter repository.LeadListFilter) ([]domain.Lead, int64, error)
	GetLeadByID(id int64) (*domain.LeadDetail, error)
	CreateLead(l *domain.Lead, createdBy int64) error
	UpdateLead(id int64, update *dto.LeadUpdateRequest, lastUpdatedBy int64) (*domain.Lead, error)
	DeleteLead(id int64, actorID int64) error
	BulkUpdateLeadStatus(leadIDs []int64, statusID int8, lastUpdatedBy int64, labID *int64, appointmentAt *time.Time) (int64, error)
	BulkImportFromCSV(csvContent []byte, clientID int64, packageID int, createdBy int64) (int, error)
	// UploadBloodTestReport validates a PDF, uploads to blob storage, then updates lead + history in one DB transaction.
	UploadBloodTestReport(ctx context.Context, leadID int64, uploadedBy int64, fh *multipart.FileHeader) (reportURL string, err error)
	// GetLeadReportDownloadURL returns a time-limited SAS URL for the lead's stored ReportURL blob.
	// Requires LeadStatusID >= LeadStatusIDReportUploaded (8) for all callers. For client JWTs (userType 2) with
	// LeadStatusID < LeadStatusIDClientDownloadNoFitGate (10): FIT may download; ON HOLD (IsFit=0) only if
	// IsReportDownloadable; UNFIT (IsFit=2) cannot download via IsReportDownloadable. At status >= 10, clients skip those checks.
	// jwtUserID scopes client (2) and lab (3) users to their own leads; employees ignore this value.
	GetLeadReportDownloadURL(ctx context.Context, leadID int64, jwtUserType int, jwtUserID int64) (downloadURL string, expiresAt time.Time, err error)
	// ApproveLeadReport sets tri-state IsFit, download flag, and approval remarks (see domain.LeadFit*).
	ApproveLeadReport(leadID int64, req *dto.ApproveLeadRequest, userID int64) error
}

type leadService struct {
	repo        repository.LeadRepository
	uow         repository.LeadUnitOfWork
	clientRepo  repository.ClientRepository
	packageRepo repository.PackageRepository
	labRepo     repository.LabRepository
	blobs       BlobService
}

func NewLeadService(repo repository.LeadRepository, uow repository.LeadUnitOfWork, clientRepo repository.ClientRepository, packageRepo repository.PackageRepository, labRepo repository.LabRepository, blobs BlobService) LeadService {
	return &leadService{repo: repo, uow: uow, clientRepo: clientRepo, packageRepo: packageRepo, labRepo: labRepo, blobs: blobs}
}

func (s *leadService) ListLeads(filter repository.LeadListFilter) ([]domain.Lead, int64, error) {
	return s.repo.List(filter)
}

func (s *leadService) GetLeadByID(id int64) (*domain.LeadDetail, error) {
	lead, err := s.repo.FindByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperrors.NewNotFound("Lead not found", err)
	}
	if err != nil {
		return nil, err
	}
	detail := &domain.LeadDetail{Lead: *lead}
	if lead.ClientID != 0 {
		if client, _ := s.clientRepo.FindByID(lead.ClientID); client != nil {
			detail.ClientName = client.ClientName
		}
	}
	if lead.PackageID != 0 {
		if pkg, _ := s.packageRepo.FindByID(int64(lead.PackageID)); pkg != nil {
			detail.PackageName = pkg.PackageName
		}
	}
	if lead.LabID != nil && *lead.LabID != 0 {
		if lab, err := s.labRepo.FindByID(*lead.LabID); err == nil && lab != nil {
			detail.LabName = lab.LabName
		}
	}
	return detail, nil
}

func (s *leadService) CreateLead(l *domain.Lead, createdBy int64) error {
	now := time.Now()
	l.CreatedBy = createdBy
	l.CreatedOn = timeutil.FromTime(now)
	l.LastUpdatedBy = createdBy
	l.LastUpdatedOn = timeutil.FromTime(now)
	// IsFit omitted → persisted NULL until report approval; create payload does not send approval fields.
	l.IsReportDownloadable = false
	l.IsFitCertificateTobeGenerated = false
	l.PatientID = s.GeneratePatientID(l.PatientName, l.ContactNumber)
	if l.LeadStatusID == 0 {
		l.LeadStatusID = domain.LeadStatusIDDefault
	}
	ct, err := domain.ParseLeadCollectionType(l.CollectionType)
	if err != nil {
		return apperrors.NewBadRequest(err.Error(), err)
	}
	l.CollectionType = ct
	l.EmpID = strings.TrimSpace(l.EmpID)
	if err := domain.ValidateLeadEmpID(l.EmpID); err != nil {
		return apperrors.NewBadRequest(err.Error(), err)
	}

	return s.uow.WithinTransaction(func(leadRepo repository.LeadRepository, historyRepo repository.LeadHistoryRepository) error {
		if err := leadRepo.Create(l); err != nil {
			return err
		}

		history := &domain.LeadHistory{
			LeadID:    l.LeadID,
			Action:    domain.LeadActionCreate,
			CreatedBy: createdBy,
		}

		if err := historyRepo.LogAction(history); err != nil {
			return err
		}

		return nil
	})
}

func (s *leadService) UpdateLead(id int64, update *dto.LeadUpdateRequest, lastUpdatedBy int64) (*domain.Lead, error) {
	existing, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.NewNotFound("Lead not found", err)
		}
		return nil, err
	}

	// Merge: only overwrite when payload field is provided (ISNULL(input, existing))
	l := *existing
	if update.ClientID != nil {
		l.ClientID = *update.ClientID
	}
	if update.PatientName != nil {
		l.PatientName = *update.PatientName
	}
	if update.Age != nil {
		l.Age = *update.Age
	}
	if update.Gender != nil {
		l.Gender = *update.Gender
	}
	if update.PackageID != nil {
		l.PackageID = *update.PackageID
	}
	if update.ContactNumber != nil {
		l.ContactNumber = *update.ContactNumber
	}
	if update.Emailid != nil {
		l.Emailid = *update.Emailid
	}
	if update.Address != nil {
		l.Address = *update.Address
	}
	if update.CityID != nil {
		l.CityID = *update.CityID
	}
	if update.StateID != nil {
		l.StateID = *update.StateID
	}
	if update.Pincode != nil {
		l.Pincode = *update.Pincode
	}
	if update.EmpID != nil {
		empID := strings.TrimSpace(*update.EmpID)
		if err := domain.ValidateLeadEmpID(empID); err != nil {
			return nil, apperrors.NewBadRequest(err.Error(), err)
		}
		l.EmpID = empID
	}
	if update.LeadStatusID != nil {
		if domain.LeadStatusIDDowngradeForbiddenForPUTLead(existing.LeadStatusID, *update.LeadStatusID) {
			return nil, apperrors.NewBadRequest(
				"This lead is already in the report workflow. LeadStatusID cannot be changed to a value below report workflow.",
				nil,
			)
		}
		if domain.LeadStatusIDForbiddenForPUTLead(*update.LeadStatusID) {
			return nil, apperrors.NewBadRequest(
				"LeadStatusID cannot be set to 8 or higher via this endpoint; use the workflow APIs for report-related statuses",
				nil,
			)
		}
		l.LeadStatusID = *update.LeadStatusID
	}
	if update.AppointmentAt != nil {
		appt := *update.AppointmentAt
		if appt.Before(time.Now()) {
			return nil, apperrors.NewBadRequest(
				"AppointmentAt must be at or after the current time (India Standard Time)",
				nil,
			)
		}
		at := timeutil.StoredFromTime(appt)
		l.AppointmentAt = &at
	}
	if update.LabID != nil {
		l.LabID = update.LabID
	}
	if update.CollectionType != nil {
		ct, err := domain.ParseLeadCollectionType(*update.CollectionType)
		if err != nil {
			return nil, apperrors.NewBadRequest(err.Error(), err)
		}
		l.CollectionType = ct
	}

	l.LeadID = id
	l.LastUpdatedBy = lastUpdatedBy
	l.LastUpdatedOn = timeutil.FromTime(time.Now())
	l.PatientID = s.GeneratePatientID(l.PatientName, l.ContactNumber)

	err = s.uow.WithinTransaction(func(leadRepo repository.LeadRepository, historyRepo repository.LeadHistoryRepository) error {
		if err := leadRepo.Update(&l); err != nil {
			return err
		}

		history := &domain.LeadHistory{
			LeadID:    l.LeadID,
			Action:    domain.LeadActionUpdate,
			CreatedBy: lastUpdatedBy,
		}

		if err := historyRepo.LogAction(history); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}
	return &l, nil
}

func (s *leadService) DeleteLead(id int64, actorID int64) error {
	exists, err := s.repo.ExistsByID(id)
	if err != nil {
		return err
	}
	if !exists {
		return apperrors.NewNotFound("Lead not found", gorm.ErrRecordNotFound)
	}
	return s.uow.WithinTransaction(func(leadRepo repository.LeadRepository, historyRepo repository.LeadHistoryRepository) error {
		if err := leadRepo.Delete(id); err != nil {
			return err
		}

		history := &domain.LeadHistory{
			LeadID:    id,
			Action:    domain.LeadActionDelete,
			CreatedBy: actorID,
		}

		if err := historyRepo.LogAction(history); err != nil {
			return err
		}

		return nil
	})
}

func (s *leadService) BulkUpdateLeadStatus(leadIDs []int64, statusID int8, lastUpdatedBy int64, labID *int64, appointmentAt *time.Time) (int64, error) {
	var appointmentPersist *time.Time
	if appointmentAt != nil {
		appt := *appointmentAt
		if appt.Before(time.Now()) {
			return 0, apperrors.NewBadRequest(
				"AppointmentAt must be at or after the current time (India Standard Time)",
				nil,
			)
		}
		st := timeutil.StoredFromTime(appt)
		appointmentPersist = timeutil.StoredToTimePtr(&st)
	}
	var affected int64
	err := s.uow.WithinTransaction(func(leadRepo repository.LeadRepository, historyRepo repository.LeadHistoryRepository) error {
		n, err := leadRepo.UpdateStatusForIDs(leadIDs, statusID, lastUpdatedBy, labID, appointmentPersist)
		if err != nil {
			return err
		}
		affected = n

		histories := make([]domain.LeadHistory, len(leadIDs))
		for i, id := range leadIDs {
			histories[i] = domain.LeadHistory{
				LeadID:    id,
				Action:    domain.LeadActionStatusUpdate,
				CreatedBy: lastUpdatedBy,
			}
		}

		if err := historyRepo.BulkLogActions(histories); err != nil {
			return err
		}

		return nil
	})
	return affected, err
}

func (s *leadService) GeneratePatientID(patientName, contactNumber string) string {
	parts := strings.Fields(patientName)
	var initials strings.Builder
	for _, part := range parts {
		if len(part) > 0 {
			initials.WriteByte(strings.ToUpper(string(part[0]))[0])
		}
	}
	return fmt.Sprintf("%s%s", initials.String(), contactNumber)
}

func (s *leadService) BulkImportFromCSV(csvContent []byte, clientID int64, packageID int, createdBy int64) (int, error) {
	if len(csvContent) == 0 {
		return 0, apperrors.NewBadRequest("CSV file is required", nil)
	}
	if createdBy == 0 {
		createdBy = 1
	}
	if clientID == 0 || packageID == 0 {
		return 0, apperrors.NewBadRequest("ClientID and PackageID are required", nil)
	}

	reader := csv.NewReader(strings.NewReader(string(csvContent)))
	reader.FieldsPerRecord = -1
	rows, err := reader.ReadAll()
	if err != nil {
		return 0, apperrors.NewBadRequest("Invalid CSV: "+err.Error(), err)
	}
	if len(rows) < 2 {
		return 0, apperrors.NewBadRequest("CSV contains no data rows", nil)
	}

	headers := rows[0]
	colIndex := func(name string) int {
		for i, h := range headers {
			if strings.TrimSpace(strings.ToLower(h)) == strings.ToLower(name) {
				return i
			}
		}
		return -1
	}
	at := func(row []string, name string) string {
		if i := colIndex(name); i >= 0 && i < len(row) {
			return strings.TrimSpace(row[i])
		}
		return ""
	}
	atInt8 := func(row []string, name string) int8 {
		s := at(row, name)
		if s == "" {
			return 0
		}
		n, _ := strconv.ParseInt(s, 10, 8)
		return int8(n)
	}
	atInt32 := func(row []string, name string) int32 {
		s := at(row, name)
		if s == "" {
			return 0
		}
		n, _ := strconv.ParseInt(s, 10, 32)
		return int32(n)
	}

	requiredCols := []string{"PatientName", "ContactNumber", "Age", "Gender", "Emailid", "Address", "CityID", "StateID", "Pincode"}
	for _, name := range requiredCols {
		if colIndex(name) < 0 {
			return 0, apperrors.NewBadRequest("CSV missing required column: "+name, nil)
		}
	}

	inserted := 0
	for rowIdx := 1; rowIdx < len(rows); rowIdx++ {
		row := rows[rowIdx]
		if len(row) == 0 {
			continue
		}
		patientName := at(row, "PatientName")
		contactNumber := at(row, "ContactNumber")
		if patientName == "" || contactNumber == "" {
			return inserted, apperrors.NewBadRequest(fmt.Sprintf("Row %d: PatientName and ContactNumber are required", rowIdx+1), nil)
		}

		leadStatusID := atInt8(row, "LeadStatusID")
		if leadStatusID == 0 {
			leadStatusID = domain.LeadStatusIDDefault
		}

		ctRaw := at(row, "CollectionType")
		if ctRaw == "" {
			ctRaw = domain.LeadCollectionCenter
		}
		collectionType, errParse := domain.ParseLeadCollectionType(ctRaw)
		if errParse != nil {
			return inserted, apperrors.NewBadRequest(fmt.Sprintf("Row %d: %s", rowIdx+1, errParse.Error()), errParse)
		}

		empID := at(row, "EmpID")
		if err := domain.ValidateLeadEmpID(empID); err != nil {
			return inserted, apperrors.NewBadRequest(fmt.Sprintf("Row %d: %s", rowIdx+1, err.Error()), err)
		}
		empID = strings.TrimSpace(empID)

		now := time.Now()
		lead := &domain.Lead{
			ClientID:       clientID,
			PatientID:      s.GeneratePatientID(patientName, contactNumber),
			PatientName:    patientName,
			Age:            atInt8(row, "Age"),
			Gender:         at(row, "Gender"),
			PackageID:      int(packageID),
			ContactNumber:  contactNumber,
			Emailid:        at(row, "Emailid"),
			Address:        at(row, "Address"),
			CityID:         atInt32(row, "CityID"),
			StateID:        atInt32(row, "StateID"),
			Pincode:        at(row, "Pincode"),
			EmpID:          empID,
			CollectionType: collectionType,
			LeadStatusID:   leadStatusID,
			CreatedBy:      createdBy,
			CreatedOn:      timeutil.FromTime(now),
			LastUpdatedBy:  createdBy,
			LastUpdatedOn:  timeutil.FromTime(now),
		}

		err := s.uow.WithinTransaction(func(leadRepo repository.LeadRepository, historyRepo repository.LeadHistoryRepository) error {
			if err := leadRepo.Create(lead); err != nil {
				return err
			}
			return historyRepo.LogAction(&domain.LeadHistory{
				LeadID:    lead.LeadID,
				Action:    domain.LeadActionCsvImport,
				CreatedBy: createdBy,
			})
		})
		if err != nil {
			return inserted, err
		}
		inserted++
	}

	return inserted, nil
}

func (s *leadService) UploadBloodTestReport(ctx context.Context, leadID int64, uploadedBy int64, fh *multipart.FileHeader) (string, error) {
	if fh == nil {
		return "", apperrors.NewBadRequest("PDF file is required", nil)
	}
	if s.blobs == nil {
		return "", apperrors.NewInternal("Report storage is not configured", nil)
	}
	if err := s.blobs.ValidateDiagnosticReportPDF(fh); err != nil {
		return "", apperrors.NewBadRequest(err.Error(), err)
	}

	lead, err := s.repo.FindByID(leadID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", apperrors.NewNotFound("Lead not found", err)
		}
		return "", err
	}
	if strings.TrimSpace(lead.ReportURL) != "" {
		return "", apperrors.NewConflict("A report is already uploaded for this lead", nil)
	}

	rc, err := fh.Open()
	if err != nil {
		return "", apperrors.NewBadRequest("Failed to read file", err)
	}
	defer func() { _ = rc.Close() }()

	reportURL, err := s.blobs.UploadDiagnosticReportPDF(ctx, rc, leadID, fh.Filename)
	if err != nil {
		slog.Error("UploadBloodTestReport: blob upload failed", slog.Int64("leadID", leadID), slog.Any("err", err))
		return "", apperrors.NewInternal("Failed to upload report", err)
	}
	if len(reportURL) > 500 {
		slog.Error("UploadBloodTestReport: report URL exceeds column length", slog.Int64("leadID", leadID), slog.Int("len", len(reportURL)))
		return "", apperrors.NewInternal("Report URL is too long", nil)
	}

	err = s.uow.WithinTransaction(func(leadRepo repository.LeadRepository, historyRepo repository.LeadHistoryRepository) error {
		// Resolve status at commit time — never hardcode LeadStatusID.
		statusID, err := leadRepo.FindActiveLeadStatusIDByName(domain.LeadActionReportUploaded)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return apperrors.NewInternal("Lead status '"+domain.LeadActionReportUploaded+"' is not configured", err)
			}
			return err
		}
		if err := leadRepo.UpdateLeadReportURLAndStatus(leadID, reportURL, statusID, uploadedBy); err != nil {
			return err
		}
		return historyRepo.LogAction(&domain.LeadHistory{
			LeadID:    leadID,
			Action:    domain.LeadActionReportUploaded,
			CreatedBy: uploadedBy,
		})
	})
	if err != nil {
		slog.Error("UploadBloodTestReport: DB failed after successful blob upload; blob may be orphaned",
			slog.Int64("leadID", leadID),
			slog.String("reportURL", reportURL),
			slog.Any("err", err),
		)
		return "", err
	}
	return reportURL, nil
}

func (s *leadService) GetLeadReportDownloadURL(ctx context.Context, leadID int64, jwtUserType int, jwtUserID int64) (string, time.Time, error) {
	if s.blobs == nil {
		return "", time.Time{}, apperrors.NewInternal("Report storage is not configured", nil)
	}
	lead, err := s.repo.FindByID(leadID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", time.Time{}, apperrors.NewNotFound("Lead not found", err)
		}
		return "", time.Time{}, err
	}
	if lead.LeadStatusID < domain.LeadStatusIDReportUploaded {
		return "", time.Time{}, apperrors.NewBadRequest("Report download is only allowed when lead status is report uploaded or later", nil)
	}
	if jwtUserType == utils.UserTypeClient {
		if jwtUserID <= 0 {
			return "", time.Time{}, apperrors.NewUnauthorized("Authentication required", nil)
		}
		if lead.ClientID != jwtUserID {
			return "", time.Time{}, apperrors.NewNotFound("Lead not found", nil)
		}
	}
	if jwtUserType == utils.UserTypeLab {
		if jwtUserID <= 0 {
			return "", time.Time{}, apperrors.NewUnauthorized("Authentication required", nil)
		}
		if lead.LabID == nil || *lead.LabID != jwtUserID {
			return "", time.Time{}, apperrors.NewNotFound("Lead not found", nil)
		}
	}
	// Client portal: below status 10, FIT may download; HOLD (IsFit=0) only if IsReportDownloadable; UNFIT (IsFit=2) never via flag; NULL / not assessed same as forbidden until approved.
	if jwtUserType == utils.UserTypeClient && lead.LeadStatusID < domain.LeadStatusIDClientDownloadNoFitGate {
		if lead.IsFit == nil {
			return "", time.Time{}, apperrors.NewForbidden("Report download is not allowed for this lead", nil)
		}
		switch *lead.IsFit {
		case domain.LeadFitFit:
			// ok
		case domain.LeadFitHold:
			if !lead.IsReportDownloadable {
				return "", time.Time{}, apperrors.NewForbidden("Report download is not allowed for this lead", nil)
			}
		case domain.LeadFitUnfit:
			return "", time.Time{}, apperrors.NewForbidden("Report download is not allowed for this lead", nil)
		default:
			return "", time.Time{}, apperrors.NewForbidden("Report download is not allowed for this lead", nil)
		}
	}
	raw := strings.TrimSpace(lead.ReportURL)
	if raw == "" {
		return "", time.Time{}, apperrors.NewNotFound("No report uploaded for this lead", gorm.ErrRecordNotFound)
	}
	u, exp, err := s.blobs.DiagnosticReportDownloadSASFromStoredURL(ctx, raw)
	if err != nil {
		return "", time.Time{}, apperrors.NewInternal("Failed to generate report download link", err)
	}
	return u, exp, nil
}

func parseLeadApprovalStatus(s string) (int8, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "fit":
		return domain.LeadFitFit, nil
	case "unfit":
		return domain.LeadFitUnfit, nil
	case "hold":
		return domain.LeadFitHold, nil
	default:
		return 0, apperrors.NewBadRequest(`status must be "fit", "unfit", or "hold"`, nil)
	}
}

// leadBrandIDUnchanged is true when the request does not ask to change BrandID, or the value matches the lead row.
func leadBrandIDUnchanged(leadBrandID, reqBrandID *int64) bool {
	if reqBrandID == nil {
		return true
	}
	if leadBrandID == nil {
		return false
	}
	return *leadBrandID == *reqBrandID
}

func truncateLeadApprovalRemarks(s string, maxRunes int) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) <= maxRunes {
		return string(r)
	}
	return string(r[:maxRunes])
}

func (s *leadService) ApproveLeadReport(leadID int64, req *dto.ApproveLeadRequest, userID int64) error {
	if req == nil {
		return apperrors.NewBadRequest("Request body is required", nil)
	}
	isFit, err := parseLeadApprovalStatus(req.Status)
	if err != nil {
		return err
	}
	lead, err := s.repo.FindByID(leadID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperrors.NewNotFound("Lead not found", err)
		}
		return err
	}
	if lead.LeadStatusID != domain.LeadStatusIDReportApproval {
		return apperrors.NewBadRequest("Report approval is only allowed when lead status ID is 8", nil)
	}
	remarksNorm := truncateLeadApprovalRemarks(req.Remarks, 250)
	var remarksPtr *string
	if remarksNorm != "" {
		remarksPtr = &remarksNorm
	}
	certTobe := *req.IsFitCertificateToBeGenerated
	// Skip DB only when nothing changes; if status is still "uploaded" (8) but decision is FIT, we must run once to set LeadStatusID = 9.
	isFitMatches := lead.IsFit != nil && *lead.IsFit == isFit
	same := isFitMatches && lead.IsReportDownloadable == req.AllowDownload && lead.IsFitCertificateTobeGenerated == certTobe && strings.TrimSpace(lead.ApprovalRemarks) == remarksNorm && leadBrandIDUnchanged(lead.BrandID, req.BrandID)
	if same && !(isFit == domain.LeadFitFit && lead.LeadStatusID == domain.LeadStatusIDReportUploaded) {
		return nil
	}
	return s.uow.WithinTransaction(func(leadRepo repository.LeadRepository, historyRepo repository.LeadHistoryRepository) error {
		n, err := leadRepo.UpdateLeadReportApproval(leadID, userID, isFit, req.AllowDownload, certTobe, remarksPtr, req.BrandID)
		if err != nil {
			return err
		}
		if n == 0 {
			return apperrors.NewConflict("Lead could not be updated; it may no longer be in status 8", nil)
		}
		return historyRepo.LogAction(&domain.LeadHistory{
			LeadID:    leadID,
			Action:    domain.LeadActionReportApproval,
			CreatedBy: userID,
		})
	})
}
