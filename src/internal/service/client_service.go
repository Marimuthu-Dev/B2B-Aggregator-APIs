package service

import (
	"errors"
	"time"

	"b2b-diagnostic-aggregator/apis/internal/apperrors"
	"b2b-diagnostic-aggregator/apis/internal/domain"
	"b2b-diagnostic-aggregator/apis/internal/repository"

	"gorm.io/gorm"
)

type ClientService interface {
	ListClients(filter repository.ClientListFilter) ([]domain.Client, int64, error)
	GetClientByID(id int64) (*domain.Client, error)
	GetClientByContactNumber(contactNumber string) (*domain.Client, error)
	CreateClient(c *domain.Client) error
	UpdateClient(id int64, c *domain.Client) error
	DeleteClient(id int64) error
	GetActiveClients() ([]domain.Client, error)
	GetClientsByCity(cityID int8) ([]domain.Client, error)
	GetClientsByState(stateID int8) ([]domain.Client, error)
}

type clientService struct {
	repo repository.ClientRepository
}

func NewClientService(repo repository.ClientRepository) ClientService {
	return &clientService{repo: repo}
}

func (s *clientService) ListClients(filter repository.ClientListFilter) ([]domain.Client, int64, error) {
	return s.repo.List(filter)
}

func (s *clientService) GetClientByID(id int64) (*domain.Client, error) {
	client, err := s.repo.FindByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperrors.NewNotFound("Client not found", err)
	}
	return client, err
}

func (s *clientService) GetClientByContactNumber(contactNumber string) (*domain.Client, error) {
	client, err := s.repo.FindByContactNumber(contactNumber)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperrors.NewNotFound("Client not found", err)
	}
	return client, err
}

func (s *clientService) CreateClient(c *domain.Client) error {
	return s.repo.Create(c)
}

func (s *clientService) UpdateClient(id int64, c *domain.Client) error {
	existing, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperrors.NewNotFound("Client not found", err)
		}
		return err
	}

	c.ClientID = id
	c.CreatedBy = existing.CreatedBy
	c.CreatedOn = existing.CreatedOn
	c.LastUpdatedOn = time.Now()
	return s.repo.Update(c)
}

func (s *clientService) DeleteClient(id int64) error {
	exists, err := s.repo.ExistsByID(id)
	if err != nil {
		return err
	}
	if !exists {
		return apperrors.NewNotFound("Client not found", gorm.ErrRecordNotFound)
	}
	return s.repo.Delete(id)
}

func (s *clientService) GetActiveClients() ([]domain.Client, error) {
	return s.repo.FindAllActive()
}

func (s *clientService) GetClientsByCity(cityID int8) ([]domain.Client, error) {
	return s.repo.FindByCity(cityID)
}

func (s *clientService) GetClientsByState(stateID int8) ([]domain.Client, error) {
	return s.repo.FindByState(stateID)
}
