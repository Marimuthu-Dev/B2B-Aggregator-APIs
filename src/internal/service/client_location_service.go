package service

import (
	"errors"
	"time"

	"b2b-diagnostic-aggregator/apis/internal/apperrors"
	"b2b-diagnostic-aggregator/apis/internal/domain"
	"b2b-diagnostic-aggregator/apis/internal/repository"

	"gorm.io/gorm"
)

type ClientLocationService interface {
	GetByClientID(clientID int64) ([]domain.ClientLocation, error)
	GetByID(id int64) (*domain.ClientLocation, error)
	Create(l *domain.ClientLocation) error
	Update(id int64, l *domain.ClientLocation) error
	Delete(id int64) error
}

type clientLocationService struct {
	repo repository.ClientLocationRepository
}

func NewClientLocationService(repo repository.ClientLocationRepository) ClientLocationService {
	return &clientLocationService{repo: repo}
}

func (s *clientLocationService) GetByClientID(clientID int64) ([]domain.ClientLocation, error) {
	return s.repo.FindByClientID(clientID)
}

func (s *clientLocationService) GetByID(id int64) (*domain.ClientLocation, error) {
	loc, err := s.repo.FindByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperrors.NewNotFound("Client location not found", err)
	}
	return loc, err
}

func (s *clientLocationService) Create(l *domain.ClientLocation) error {
	return s.repo.Create(l)
}

func (s *clientLocationService) Update(id int64, l *domain.ClientLocation) error {
	exists, err := s.repo.ExistsByID(id)
	if err != nil {
		return err
	}
	if !exists {
		return apperrors.NewNotFound("Client location not found", gorm.ErrRecordNotFound)
	}
	l.ClientLocationID = id
	l.LastUpdatedOn = time.Now()
	return s.repo.Update(l)
}

func (s *clientLocationService) Delete(id int64) error {
	exists, err := s.repo.ExistsByID(id)
	if err != nil {
		return err
	}
	if !exists {
		return apperrors.NewNotFound("Client location not found", gorm.ErrRecordNotFound)
	}
	return s.repo.Delete(id)
}
