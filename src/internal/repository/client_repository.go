package repository

import (
	"fmt"

	"b2b-diagnostic-aggregator/apis/internal/domain"
	persistencemodels "b2b-diagnostic-aggregator/apis/internal/persistence/models"

	"gorm.io/gorm"
)

type ClientRepository interface {
	FindAll() ([]domain.Client, error)
	List(filter ClientListFilter) ([]domain.Client, int64, error)
	FindByID(id int64) (*domain.Client, error)
	ExistsByID(id int64) (bool, error)
	Create(c *domain.Client) error
	Update(c *domain.Client) error
	Delete(id int64) error
	FindAllActive() ([]domain.Client, error)
	FindByContactNumber(contactNumber string) (*domain.Client, error)
	FindByCity(cityID int8) ([]domain.Client, error)
	FindByState(stateID int8) ([]domain.Client, error)
}

type clientRepository struct {
	db *gorm.DB
}

func NewClientRepository(db *gorm.DB) ClientRepository {
	return &clientRepository{db: db}
}

func (r *clientRepository) FindAll() ([]domain.Client, error) {
	var clients []persistencemodels.Client
	err := r.db.Find(&clients).Error
	return mapClientsToDomain(clients), err
}

func (r *clientRepository) List(filter ClientListFilter) ([]domain.Client, int64, error) {
	query := r.db.Model(&persistencemodels.Client{})
	if filter.CityID != nil {
		query = query.Where("CityID = ?", *filter.CityID)
	}
	if filter.StateID != nil {
		query = query.Where("StateID = ?", *filter.StateID)
	}
	if filter.IsActive != nil {
		query = query.Where("IsAcitve = ?", *filter.IsActive)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	sortColumn := mapClientSortColumn(filter.SortBy)
	order := normalizeSortOrder(filter.SortOrder)
	offset := (filter.Page - 1) * filter.PageSize

	var clients []persistencemodels.Client
	err := query.Order(sortColumn + " " + order).Limit(filter.PageSize).Offset(offset).Find(&clients).Error
	return mapClientsToDomain(clients), total, err
}

func mapClientSortColumn(sortBy string) string {
	switch sortBy {
	case "name":
		return "ClientName"
	case "cityId":
		return "CityID"
	case "stateId":
		return "StateID"
	case "createdOn":
		return "CreatedOn"
	default:
		return "ClientID"
	}
}

func (r *clientRepository) FindByID(id int64) (*domain.Client, error) {
	var c persistencemodels.Client
	err := r.db.First(&c, id).Error
	if err != nil {
		return nil, err
	}
	domainClient := mapClientToDomain(c)
	return &domainClient, nil
}

func (r *clientRepository) ExistsByID(id int64) (bool, error) {
	var count int64
	if err := r.db.Model(&persistencemodels.Client{}).Where("ClientID = ?", id).Limit(1).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *clientRepository) Create(c *domain.Client) error {
	persist := mapClientToPersistence(*c)
	if err := r.db.Create(&persist).Error; err != nil {
		return err
	}
	*c = mapClientToDomain(persist)
	return nil
}

func (r *clientRepository) Update(c *domain.Client) error {
	persist := mapClientToPersistence(*c)
	if err := r.db.Save(&persist).Error; err != nil {
		return err
	}
	*c = mapClientToDomain(persist)
	return nil
}

func (r *clientRepository) Delete(id int64) error {
	return r.db.Delete(&persistencemodels.Client{}, id).Error
}

func (r *clientRepository) FindAllActive() ([]domain.Client, error) {
	var clients []persistencemodels.Client
	err := r.db.Where("IsAcitve = ?", true).Find(&clients).Error
	return mapClientsToDomain(clients), err
}

func (r *clientRepository) FindByContactNumber(contactNumber string) (*domain.Client, error) {
	fmt.Printf("[LOGIN] Repository.Client.FindByContactNumber: entry contactNumber=%s\n", contactNumber)
	var c persistencemodels.Client
	err := r.db.Where("ContactPerson1Number = ? OR ContactPerson2Number = ?", contactNumber, contactNumber).First(&c).Error
	if err != nil {
		fmt.Printf("[LOGIN] Repository.Client.FindByContactNumber: not found err=%v\n", err)
		return nil, err
	}
	domainClient := mapClientToDomain(c)
	fmt.Printf("[LOGIN] Repository.Client.FindByContactNumber: found ClientID=%d\n", domainClient.ClientID)
	return &domainClient, nil
}

func (r *clientRepository) FindByCity(cityID int8) ([]domain.Client, error) {
	var clients []persistencemodels.Client
	err := r.db.Where("CityID = ?", cityID).Find(&clients).Error
	return mapClientsToDomain(clients), err
}

func (r *clientRepository) FindByState(stateID int8) ([]domain.Client, error) {
	var clients []persistencemodels.Client
	err := r.db.Where("StateID = ?", stateID).Find(&clients).Error
	return mapClientsToDomain(clients), err
}
