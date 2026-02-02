package service

import (
	"errors"

	"b2b-diagnostic-aggregator/apis/internal/apperrors"
	"b2b-diagnostic-aggregator/apis/internal/domain"
	"b2b-diagnostic-aggregator/apis/internal/repository"

	"gorm.io/gorm"
)

type LabService interface {
	ListLabs(filter repository.LabListFilter) ([]domain.Lab, int64, error)
	GetLabByID(id int64) (*domain.Lab, error)
	GetLabByContactNumber(contactNumber string) (*domain.Lab, error)
	CreateLab(l *domain.Lab) error
	UpdateLab(id int64, l *domain.Lab) error
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

func (s *labService) CreateLab(l *domain.Lab) error {
	return s.repo.Create(l)
}

func (s *labService) UpdateLab(id int64, l *domain.Lab) error {
	exists, err := s.repo.ExistsByID(id)
	if err != nil {
		return err
	}
	if !exists {
		return apperrors.NewNotFound("Lab not found", gorm.ErrRecordNotFound)
	}
	l.LabID = id
	return s.repo.Update(l)
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
