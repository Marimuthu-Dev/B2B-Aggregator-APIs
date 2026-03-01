package service

import (
	"errors"
	"time"

	"b2b-diagnostic-aggregator/apis/internal/apperrors"
	"b2b-diagnostic-aggregator/apis/internal/domain"
	"b2b-diagnostic-aggregator/apis/internal/dto"
	"b2b-diagnostic-aggregator/apis/internal/repository"

	"gorm.io/gorm"
)

type LabService interface {
	ListLabs(filter repository.LabListFilter) ([]domain.Lab, int64, error)
	GetLabByID(id int64) (*domain.Lab, error)
	GetLabByContactNumber(contactNumber string) (*domain.Lab, error)
	CreateLab(l *domain.Lab, createdBy int64) error
	UpdateLab(id int64, update *dto.LabUpdateRequest, lastUpdatedBy int64) (*domain.Lab, error)
	DeleteLab(id int64) error
	GetActiveLabs() ([]domain.Lab, error)
	GetLabsByCity(cityID int8) ([]domain.Lab, error)
	GetLabsByState(stateID int8) ([]domain.Lab, error)
}

type labService struct {
	repo repository.LabRepository
}

func NewLabService(repo repository.LabRepository) LabService {
	return &labService{repo: repo}
}

func (s *labService) ListLabs(filter repository.LabListFilter) ([]domain.Lab, int64, error) {
	return s.repo.List(filter)
}

func (s *labService) GetLabByID(id int64) (*domain.Lab, error) {
	lab, err := s.repo.FindByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperrors.NewNotFound("Lab not found", err)
	}
	return lab, err
}

func (s *labService) GetLabByContactNumber(contactNumber string) (*domain.Lab, error) {
	lab, err := s.repo.FindByContactNumber(contactNumber)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperrors.NewNotFound("Lab not found", err)
	}
	return lab, err
}

func (s *labService) CreateLab(l *domain.Lab, createdBy int64) error {
	now := time.Now()
	l.CreatedBy = &createdBy
	l.CreatedOn = &now
	l.LastUpdatedBy = &createdBy
	l.LastUpdatedOn = &now
	return s.repo.Create(l)
}

func (s *labService) UpdateLab(id int64, update *dto.LabUpdateRequest, lastUpdatedBy int64) (*domain.Lab, error) {
	existing, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.NewNotFound("Lab not found", err)
		}
		return nil, err
	}
	l := *existing
	if update.LabName != nil {
		l.LabName = *update.LabName
	}
	if update.Address != nil {
		l.Address = update.Address
	}
	if update.CityID != nil {
		l.CityID = update.CityID
	}
	if update.StateID != nil {
		l.StateID = update.StateID
	}
	if update.Pincode != nil {
		l.Pincode = update.Pincode
	}
	if update.ContactPerson1Name != nil {
		l.ContactPerson1Name = update.ContactPerson1Name
	}
	if update.ContactPerson1Number != nil {
		l.ContactPerson1Number = update.ContactPerson1Number
	}
	if update.ContactPerson1EmailID != nil {
		l.ContactPerson1EmailID = update.ContactPerson1EmailID
	}
	if update.ContactPerson1Designation != nil {
		l.ContactPerson1Designation = update.ContactPerson1Designation
	}
	if update.ContactPerson1Name1 != nil {
		l.ContactPerson1Name1 = update.ContactPerson1Name1
	}
	if update.ContactPerson1Number1 != nil {
		l.ContactPerson1Number1 = update.ContactPerson1Number1
	}
	if update.ContactPerson1EmailID1 != nil {
		l.ContactPerson1EmailID1 = update.ContactPerson1EmailID1
	}
	if update.ContactPerson1Designation1 != nil {
		l.ContactPerson1Designation1 = update.ContactPerson1Designation1
	}
	if update.CategoryID != nil {
		l.CategoryID = update.CategoryID
	}
	if update.GSTIN_UIN != nil {
		l.GSTIN_UIN = update.GSTIN_UIN
	}
	if update.PANNumber != nil {
		l.PANNumber = update.PANNumber
	}
	if t := update.GetMOUStartDate(); t != nil {
		l.MOUStartDate = t
	}
	if t := update.GetMOUEndDate(); t != nil {
		l.MOUEndDate = t
	}
	if update.AccreditationID != nil {
		l.AccreditationID = update.AccreditationID
	}
	if s := update.GetCollectionTypes(); s != nil {
		l.CollectionTypes = s
	}
	if s := update.GetServicesID(); s != nil {
		l.ServicesID = s
	}
	if s := update.GetCollectionPincodes(); s != nil {
		l.CollectionPincodes = s
	}
	if update.IsActive != nil {
		l.IsActive = update.IsActive
	}
	l.LabID = id
	l.LastUpdatedBy = &lastUpdatedBy
	now := time.Now()
	l.LastUpdatedOn = &now
	if err := s.repo.Update(&l); err != nil {
		return nil, err
	}
	return &l, nil
}

func (s *labService) DeleteLab(id int64) error {
	exists, err := s.repo.ExistsByID(id)
	if err != nil {
		return err
	}
	if !exists {
		return apperrors.NewNotFound("Lab not found", gorm.ErrRecordNotFound)
	}
	return s.repo.Delete(id)
}

func (s *labService) GetActiveLabs() ([]domain.Lab, error) {
	return s.repo.FindAllActive()
}

func (s *labService) GetLabsByCity(cityID int8) ([]domain.Lab, error) {
	return s.repo.FindByCity(cityID)
}

func (s *labService) GetLabsByState(stateID int8) ([]domain.Lab, error) {
	return s.repo.FindByState(stateID)
}
