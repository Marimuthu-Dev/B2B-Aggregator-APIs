package repository

import (
	"b2b-diagnostic-aggregator/apis/internal/models"
	"gorm.io/gorm"
)

type LabRepository interface {
	FindAll() ([]models.Lab, error)
	FindByID(id int64) (*models.Lab, error)
	Create(l *models.Lab) error
	Update(l *models.Lab) error
	Delete(id int64) error
	FindAllActive() ([]models.Lab, error)
	FindByContactNumber(contactNumber string) (*models.Lab, error)
	FindByCity(cityID int8) ([]models.Lab, error)
	FindByState(stateID int8) ([]models.Lab, error)
}

type labRepository struct {
	db *gorm.DB
}

func NewLabRepository(db *gorm.DB) LabRepository {
	return &labRepository{db: db}
}

func (r *labRepository) FindAll() ([]models.Lab, error) {
	var labs []models.Lab
	err := r.db.Find(&labs).Error
	return labs, err
}

func (r *labRepository) FindByID(id int64) (*models.Lab, error) {
	var l models.Lab
	err := r.db.First(&l, id).Error
	if err != nil {
		return nil, err
	}
	return &l, nil
}

func (r *labRepository) Create(l *models.Lab) error {
	return r.db.Create(l).Error
}

func (r *labRepository) Update(l *models.Lab) error {
	return r.db.Save(l).Error
}

func (r *labRepository) Delete(id int64) error {
	return r.db.Delete(&models.Lab{}, id).Error
}

func (r *labRepository) FindAllActive() ([]models.Lab, error) {
	var labs []models.Lab
	err := r.db.Where("IsActive = ?", true).Find(&labs).Error
	return labs, err
}

func (r *labRepository) FindByContactNumber(contactNumber string) (*models.Lab, error) {
	var l models.Lab
	err := r.db.Where("ContactPerson1Number = ? OR ContactPerson1Number1 = ?", contactNumber, contactNumber).First(&l).Error
	if err != nil {
		return nil, err
	}
	return &l, nil
}

func (r *labRepository) FindByCity(cityID int8) ([]models.Lab, error) {
	var labs []models.Lab
	err := r.db.Where("CityID = ?", cityID).Find(&labs).Error
	return labs, err
}

func (r *labRepository) FindByState(stateID int8) ([]models.Lab, error) {
	var labs []models.Lab
	err := r.db.Where("StateID = ?", stateID).Find(&labs).Error
	return labs, err
}
