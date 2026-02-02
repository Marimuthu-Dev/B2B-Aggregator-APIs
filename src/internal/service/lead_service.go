package service

import (
	"errors"
	"fmt"
	"strings"

	"b2b-diagnostic-aggregator/apis/internal/apperrors"
	"b2b-diagnostic-aggregator/apis/internal/domain"
	"b2b-diagnostic-aggregator/apis/internal/repository"

	"gorm.io/gorm"
)

type LeadService interface {
	ListLeads(filter repository.LeadListFilter) ([]domain.Lead, int64, error)
	GetLeadByID(id int64) (*domain.Lead, error)
	CreateLead(l *domain.Lead) error
	UpdateLead(id int64, l *domain.Lead) error
	DeleteLead(id int64, actorID int64) error
	BulkUpdateLeadStatus(leadIDs []int64, statusID int8, lastUpdatedBy int64) error
}

type leadService struct {
	repo        repository.LeadRepository
	uow         repository.LeadUnitOfWork
}

func NewLeadService(repo repository.LeadRepository, uow repository.LeadUnitOfWork) LeadService {
	return &leadService{repo: repo, uow: uow}
}

func (s *leadService) ListLeads(filter repository.LeadListFilter) ([]domain.Lead, int64, error) {
	return s.repo.List(filter)
}

func (s *leadService) GetLeadByID(id int64) (*domain.Lead, error) {
	lead, err := s.repo.FindByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperrors.NewNotFound("Lead not found", err)
	}
	return lead, err
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

func (s *leadService) BulkUpdateLeadStatus(leadIDs []int64, statusID int8, lastUpdatedBy int64) error {
	return s.uow.WithinTransaction(func(leadRepo repository.LeadRepository, historyRepo repository.LeadHistoryRepository) error {
		if err := leadRepo.UpdateStatusForIDs(leadIDs, statusID, lastUpdatedBy); err != nil {
			return err
		}

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
