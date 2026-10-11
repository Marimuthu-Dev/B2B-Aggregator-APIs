package repository

import (
	"b2b-diagnostic-aggregator/apis/internal/persistence/models"

	"gorm.io/gorm"
)

type LocationRepository interface {
	GetAllStates() ([]models.StateMaster, error)
	GetAllCities() ([]models.CityMaster, error)
	GetCitiesByStateID(stateID int32) ([]models.CityMaster, error)
}

type locationRepository struct {
	db *gorm.DB
}

func NewLocationRepository(db *gorm.DB) LocationRepository {
	return &locationRepository{db: db}
}

func (r *locationRepository) GetAllStates() ([]models.StateMaster, error) {
	var states []models.StateMaster
	err := r.db.Table(models.Table("tbl_StateMaster")).
		Where("IsActive IS NULL OR IsActive = ?", true).
		Order("StateName ASC").
		Find(&states).Error
	return states, err
}

func (r *locationRepository) GetAllCities() ([]models.CityMaster, error) {
	var cities []models.CityMaster
	err := r.db.Table(models.Table("tbl_CityMaster")).
		Where("IsActive IS NULL OR IsActive = ?", true).
		Order("CityName ASC").
		Find(&cities).Error
	return cities, err
}

func (r *locationRepository) GetCitiesByStateID(stateID int32) ([]models.CityMaster, error) {
	var cities []models.CityMaster
	err := r.db.Table(models.Table("tbl_CityMaster")).
		Where("StateID = ? AND (IsActive IS NULL OR IsActive = ?)", stateID, true).
		Order("CityName ASC").
		Find(&cities).Error
	return cities, err
}
