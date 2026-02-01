package service

import (
	"time"

	"b2b-diagnostic-aggregator/apis/internal/models"
	"b2b-diagnostic-aggregator/apis/internal/repository"
)

type ClientService interface {
	GetAllClients() ([]models.Client, error)
	GetClientByID(id int64) (*models.Client, error)
	GetClientByContactNumber(contactNumber string) (*models.Client, error)
	CreateClient(c *models.Client) error
	UpdateClient(id int64, c *models.Client) error
	DeleteClient(id int64) error
	GetActiveClients() ([]models.Client, error)
	GetClientsByCity(cityID int8) ([]models.Client, error)
	GetClientsByState(stateID int8) ([]models.Client, error)
}

type clientService struct {
	repo repository.ClientRepository
}

func NewClientService(repo repository.ClientRepository) ClientService {
	return &clientService{repo: repo}
}

func (s *clientService) GetAllClients() ([]models.Client, error) {
	return s.repo.FindAll()
}

func (s *clientService) GetClientByID(id int64) (*models.Client, error) {
	return s.repo.FindByID(id)
}

func (s *clientService) GetClientByContactNumber(contactNumber string) (*models.Client, error) {
	return s.repo.FindByContactNumber(contactNumber)
}

func (s *clientService) CreateClient(c *models.Client) error {
	return s.repo.Create(c)
}

func (s *clientService) UpdateClient(id int64, c *models.Client) error {
	existing, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}

	c.ClientID = id
	c.CreatedBy = existing.CreatedBy
	c.CreatedOn = existing.CreatedOn
	c.LastUpdatedOn = time.Now()
	return s.repo.Update(c)
}

func (s *clientService) DeleteClient(id int64) error {
	return s.repo.Delete(id)
}

func (s *clientService) GetActiveClients() ([]models.Client, error) {
	return s.repo.FindAllActive()
}

func (s *clientService) GetClientsByCity(cityID int8) ([]models.Client, error) {
	return s.repo.FindByCity(cityID)
}

func (s *clientService) GetClientsByState(stateID int8) ([]models.Client, error) {
	return s.repo.FindByState(stateID)
}
