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

type ClientService interface {
	ListClients(filter repository.ClientListFilter) ([]domain.Client, int64, error)
	GetClientByID(id int64) (*domain.Client, error)
	GetClientByContactNumber(contactNumber string) (*domain.Client, error)
	CreateClient(c *domain.Client, createdBy int64) error
	UpdateClient(id int64, update *dto.ClientUpdateRequest, lastUpdatedBy int64) (*domain.Client, error)
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

func (s *clientService) CreateClient(c *domain.Client, createdBy int64) error {
	now := time.Now()
	c.CreatedBy = createdBy
	c.CreatedOn = now
	c.LastUpdatedBy = createdBy
	c.LastUpdatedOn = now
	return s.repo.Create(c)
}

func (s *clientService) UpdateClient(id int64, update *dto.ClientUpdateRequest, lastUpdatedBy int64) (*domain.Client, error) {
	existing, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.NewNotFound("Client not found", err)
		}
		return nil, err
	}

	c := *existing
	if update.ClientName != nil {
		c.ClientName = *update.ClientName
	}
	if update.Address != nil {
		c.Address = *update.Address
	}
	if update.CityID != nil {
		c.CityID = *update.CityID
	}
	if update.StateID != nil {
		c.StateID = *update.StateID
	}
	if update.Pincode != nil {
		c.Pincode = *update.Pincode
	}
	if update.ContactPerson1Name != nil {
		c.ContactPerson1Name = *update.ContactPerson1Name
	}
	if update.ContactPerson1Number != nil {
		c.ContactPerson1Number = *update.ContactPerson1Number
	}
	if update.ContactPerson1EmailID != nil {
		c.ContactPerson1EmailID = *update.ContactPerson1EmailID
	}
	if update.ContactPerson1Designation != nil {
		c.ContactPerson1Designation = *update.ContactPerson1Designation
	}
	if update.ContactPerson2Name != nil {
		c.ContactPerson2Name = update.ContactPerson2Name
	}
	if update.ContactPerson2Number != nil {
		c.ContactPerson2Number = update.ContactPerson2Number
	}
	if update.ContactPerson2EmailID != nil {
		c.ContactPerson2EmailID = update.ContactPerson2EmailID
	}
	if update.ContactPerson2Designation != nil {
		c.ContactPerson2Designation = update.ContactPerson2Designation
	}
	if update.CategoryID != nil {
		c.CategoryID = update.CategoryID
	}
	if update.GSTIN_UIN != nil {
		c.GSTIN_UIN = update.GSTIN_UIN
	}
	if update.PANNumber != nil {
		c.PANNumber = update.PANNumber
	}
	if update.BusinessVertical != nil {
		c.BusinessVertical = *update.BusinessVertical
	}
	if update.BillingName != nil {
		c.BillingName = update.BillingName
	}
	if update.BillingAdderss != nil {
		c.BillingAdderss = update.BillingAdderss
	}
	if update.BillingPincode != nil {
		c.BillingPincode = update.BillingPincode
	}
	if update.IsAcitve != nil {
		c.IsAcitve = *update.IsAcitve
	}

	c.ClientID = id
	c.LastUpdatedBy = lastUpdatedBy
	c.LastUpdatedOn = time.Now()
	if err := s.repo.Update(&c); err != nil {
		return nil, err
	}
	return &c, nil
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
