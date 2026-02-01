package service

import (
	"b2b-diagnostic-aggregator/apis/internal/models"
	"b2b-diagnostic-aggregator/apis/internal/repository"
	"fmt"
	"gorm.io/gorm"
	"strings"
)

type LeadService interface {
	GetAllLeads() ([]models.Lead, error)
	GetLeadByID(id int64) (*models.Lead, error)
	CreateLead(l *models.Lead) error
	UpdateLead(id int64, l *models.Lead) error
	DeleteLead(id int64, actorID int64) error
	BulkUpdateLeadStatus(leadIDs []int64, statusID int8, lastUpdatedBy int64) error
}

type leadService struct {
	repo        repository.LeadRepository
	historyRepo repository.LeadHistoryRepository
	db          *gorm.DB
}

func NewLeadService(repo repository.LeadRepository, historyRepo repository.LeadHistoryRepository, db *gorm.DB) LeadService {
	return &leadService{repo: repo, historyRepo: historyRepo, db: db}
}

func (s *leadService) GetAllLeads() ([]models.Lead, error) {
	return s.repo.FindAll()
}

func (s *leadService) GetLeadByID(id int64) (*models.Lead, error) {
	return s.repo.FindByID(id)
}

func (s *leadService) CreateLead(l *models.Lead) error {
	l.PatientID = s.GeneratePatientID(l.PatientName, l.ContactNumber)

	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(l).Error; err != nil {
			return err
		}

		history := &models.LeadHistory{
			LeadID:    l.LeadID,
			Action:    "CREATE",
			CreatedBy: l.CreatedBy,
		}

		if err := tx.Create(history).Error; err != nil {
			return err
		}

		return nil
	})
}

func (s *leadService) UpdateLead(id int64, l *models.Lead) error {
	existing, err := s.repo.FindByID(id)
	if err != nil {
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
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(l).Error; err != nil {
			return err
		}

		actor := l.LastUpdatedBy
		if actor == 0 {
			actor = l.CreatedBy
		}

		history := &models.LeadHistory{
			LeadID:    l.LeadID,
			Action:    "UPDATE",
			CreatedBy: actor,
		}

		if err := tx.Create(history).Error; err != nil {
			return err
		}

		return nil
	})
}

func (s *leadService) DeleteLead(id int64, actorID int64) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&models.Lead{}, id).Error; err != nil {
			return err
		}

		history := &models.LeadHistory{
			LeadID:    id,
			Action:    "DELETE",
			CreatedBy: actorID,
		}

		if err := tx.Create(history).Error; err != nil {
			return err
		}

		return nil
	})
}

func (s *leadService) BulkUpdateLeadStatus(leadIDs []int64, statusID int8, lastUpdatedBy int64) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.Lead{}).Where("LeadID IN ?", leadIDs).Updates(map[string]interface{}{
			"LeadStatusID":  statusID,
			"LastUpdatedBy": lastUpdatedBy,
		}).Error; err != nil {
			return err
		}

		histories := make([]models.LeadHistory, len(leadIDs))
		for i, id := range leadIDs {
			histories[i] = models.LeadHistory{
				LeadID:    id,
				Action:    "STATUS_UPDATE",
				CreatedBy: lastUpdatedBy,
			}
		}

		if err := tx.Create(&histories).Error; err != nil {
			return err
		}

		return nil
	})
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
