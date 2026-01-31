package repository

import (
	"b2b-diagnostic-aggregator/apis/internal/models"
	"gorm.io/gorm"
)

type ClientRepository interface {
	FindAll() ([]models.Client, error)
	FindByID(id int64) (*models.Client, error)
	Create(c *models.Client) error
	Update(c *models.Client) error
	Delete(id int64) error
	FindAllActive() ([]models.Client, error)
	FindByContactNumber(contactNumber string) (*models.Client, error)
	FindByCity(cityID int8) ([]models.Client, error)
	FindByState(stateID int8) ([]models.Client, error)
}

type clientRepository struct {
	db *gorm.DB
}

func NewClientRepository(db *gorm.DB) ClientRepository {
	return &clientRepository{db: db}
}

func (r *clientRepository) FindAll() ([]models.Client, error) {
	var clients []models.Client
	err := r.db.Find(&clients).Error
	return clients, err
}

func (r *clientRepository) FindByID(id int64) (*models.Client, error) {
	var c models.Client
	err := r.db.First(&c, id).Error
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *clientRepository) Create(c *models.Client) error {
	return r.db.Create(c).Error
}

func (r *clientRepository) Update(c *models.Client) error {
	return r.db.Save(c).Error
}

func (r *clientRepository) Delete(id int64) error {
	return r.db.Delete(&models.Client{}, id).Error
}

func (r *clientRepository) FindAllActive() ([]models.Client, error) {
	var clients []models.Client
	err := r.db.Where("IsAcitve = ?", true).Find(&clients).Error
	return clients, err
}

func (r *clientRepository) FindByContactNumber(contactNumber string) (*models.Client, error) {
	var c models.Client
	err := r.db.Where("ContactPerson1Number = ? OR ContactPerson2Number = ?", contactNumber, contactNumber).First(&c).Error
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *clientRepository) FindByCity(cityID int8) ([]models.Client, error) {
	var clients []models.Client
	err := r.db.Where("CityID = ?", cityID).Find(&clients).Error
	return clients, err
}

func (r *clientRepository) FindByState(stateID int8) ([]models.Client, error) {
	var clients []models.Client
	err := r.db.Where("StateID = ?", stateID).Find(&clients).Error
	return clients, err
}
