package service

import (
	"encoding/csv"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"b2b-diagnostic-aggregator/apis/internal/apperrors"
	"b2b-diagnostic-aggregator/apis/internal/domain"
	"b2b-diagnostic-aggregator/apis/internal/repository"

	"gorm.io/gorm"
)

type LeadService interface {
	ListLeads(filter repository.LeadListFilter) ([]domain.Lead, int64, error)
	GetLeadByID(id int64) (*domain.LeadDetail, error)
	CreateLead(l *domain.Lead) error
	UpdateLead(id int64, l *domain.Lead) error
	DeleteLead(id int64, actorID int64) error
	BulkUpdateLeadStatus(leadIDs []int64, statusID int8, lastUpdatedBy int64) (int64, error)
	BulkImportFromCSV(csvContent []byte, clientID int64, packageID int) (int, error)
}

type leadService struct {
	repo        repository.LeadRepository
	uow         repository.LeadUnitOfWork
	clientRepo  repository.ClientRepository
	packageRepo repository.PackageRepository
}

func NewLeadService(repo repository.LeadRepository, uow repository.LeadUnitOfWork, clientRepo repository.ClientRepository, packageRepo repository.PackageRepository) LeadService {
	return &leadService{repo: repo, uow: uow, clientRepo: clientRepo, packageRepo: packageRepo}
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
		if pkg, _ := s.packageRepo.FindByID(lead.PackageID); pkg != nil {
			detail.PackageName = pkg.PackageName
		}
	}
	return detail, nil
}

func (s *leadService) CreateLead(l *domain.Lead) error {
	l.PatientID = s.GeneratePatientID(l.PatientName, l.ContactNumber)

	return s.uow.WithinTransaction(func(leadRepo repository.LeadRepository, historyRepo repository.LeadHistoryRepository) error {
		if err := leadRepo.Create(l); err != nil {
			return err
		}

		history := &domain.LeadHistory{
			LeadID:    l.LeadID,
			Action:    domain.LeadActionCreate,
			CreatedBy: l.CreatedBy,
		}

		if err := historyRepo.LogAction(history); err != nil {
			return err
		}

		return nil
	})
}

func (s *leadService) UpdateLead(id int64, l *domain.Lead) error {
	existing, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperrors.NewNotFound("Lead not found", err)
		}
		return err
	}

	if l.PatientName != "" || l.ContactNumber != "" {
		nameToUse := l.PatientName
		if nameToUse == "" {
			nameToUse = existing.PatientName
		}
		contactToUse := l.ContactNumber
		if contactToUse == "" {
			contactToUse = existing.ContactNumber
		}
		l.PatientID = s.GeneratePatientID(nameToUse, contactToUse)
	}

	l.LeadID = id
	return s.uow.WithinTransaction(func(leadRepo repository.LeadRepository, historyRepo repository.LeadHistoryRepository) error {
		if err := leadRepo.Update(l); err != nil {
			return err
		}

		actor := l.LastUpdatedBy
		if actor == 0 {
			actor = l.CreatedBy
		}

		history := &domain.LeadHistory{
			LeadID:    l.LeadID,
			Action:    domain.LeadActionUpdate,
			CreatedBy: actor,
		}

		if err := historyRepo.LogAction(history); err != nil {
			return err
		}

		return nil
	})
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

func (s *leadService) BulkUpdateLeadStatus(leadIDs []int64, statusID int8, lastUpdatedBy int64) (int64, error) {
	var affected int64
	err := s.uow.WithinTransaction(func(leadRepo repository.LeadRepository, historyRepo repository.LeadHistoryRepository) error {
		n, err := leadRepo.UpdateStatusForIDs(leadIDs, statusID, lastUpdatedBy)
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

func (s *leadService) BulkImportFromCSV(csvContent []byte, clientID int64, packageID int) (int, error) {
	if len(csvContent) == 0 {
		return 0, apperrors.NewBadRequest("CSV file is required", nil)
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
	atInt64 := func(row []string, name string) int64 {
		s := at(row, name)
		if s == "" {
			return 0
		}
		n, _ := strconv.ParseInt(s, 10, 64)
		return n
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

		lead := &domain.Lead{
			ClientID:      clientID,
			PatientID:     s.GeneratePatientID(patientName, contactNumber),
			PatientName:   patientName,
			Age:           atInt8(row, "Age"),
			Gender:        at(row, "Gender"),
			PackageID:     int(packageID),
			ContactNumber: contactNumber,
			Emailid:       at(row, "Emailid"),
			Address:       at(row, "Address"),
			CityID:        atInt8(row, "CityID"),
			StateID:       atInt8(row, "StateID"),
			Pincode:       at(row, "Pincode"),
			LeadStatusID:  atInt8(row, "LeadStatusID"),
			CreatedOn:     time.Now(),
			LastUpdatedOn: time.Now(),
		}
		createdBy := atInt64(row, "CreatedBy")
		if createdBy == 0 {
			createdBy = 1
		}
		lead.CreatedBy = createdBy
		lead.LastUpdatedBy = atInt64(row, "LastUpdatedBy")
		if lead.LastUpdatedBy == 0 {
			lead.LastUpdatedBy = 1
		}

		err := s.uow.WithinTransaction(func(leadRepo repository.LeadRepository, historyRepo repository.LeadHistoryRepository) error {
			if err := leadRepo.Create(lead); err != nil {
				return err
			}
			actor := lead.CreatedBy
			return historyRepo.LogAction(&domain.LeadHistory{
				LeadID:    lead.LeadID,
				Action:    domain.LeadActionCsvImport,
				CreatedBy: actor,
			})
		})
		if err != nil {
			return inserted, err
		}
		inserted++
	}

	return inserted, nil
}
