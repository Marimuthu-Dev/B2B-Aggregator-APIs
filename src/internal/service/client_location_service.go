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

type ClientLocationService interface {
	GetByClientID(clientID int64) ([]domain.ClientLocation, error)
	GetByID(id int64) (*domain.ClientLocation, error)
	Create(l *domain.ClientLocation, createdBy int64) error
	Update(id int64, update *dto.ClientLocationUpdateRequest, lastUpdatedBy int64) (*domain.ClientLocation, error)
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

func (s *clientLocationService) Create(l *domain.ClientLocation, createdBy int64) error {
	now := time.Now()
	l.CreatedBy = createdBy
	l.CreatedOn = now
	l.LastUpdatedBy = createdBy
	l.LastUpdatedOn = now
	return s.repo.Create(l)
}

func (s *clientLocationService) Update(id int64, update *dto.ClientLocationUpdateRequest, lastUpdatedBy int64) (*domain.ClientLocation, error) {
	existing, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.NewNotFound("Client location not found", err)
		}
		return nil, err
	}
	l := *existing
	if update.Address != nil {
		l.Address = *update.Address
	}
	if update.Pincode != nil {
		l.Pincode = *update.Pincode
	}
	if update.CityID != nil {
		l.CityID = *update.CityID
	}
	if update.StateID != nil {
		l.StateID = *update.StateID
	}
	if update.IsActive != nil {
		l.IsActive = *update.IsActive
	}
	l.ClientLocationID = id
	l.LastUpdatedBy = lastUpdatedBy
	l.LastUpdatedOn = time.Now()
	if err := s.repo.Update(&l); err != nil {
		return nil, err
	}
	return &l, nil
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
