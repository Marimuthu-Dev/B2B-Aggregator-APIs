package repository

import (
	"b2b-diagnostic-aggregator/apis/internal/persistence/models"

	"gorm.io/gorm"
)

type EmploymentTypeRepository interface {
	GetActiveEmploymentTypes() ([]models.EmploymentTypeMaster, error)
	GetByID(id uint8) (*models.EmploymentTypeMaster, error)
}

type employmentTypeRepository struct {
	db *gorm.DB
}

func NewEmploymentTypeRepository(db *gorm.DB) EmploymentTypeRepository {
	return &employmentTypeRepository{db: db}
}

func (r *employmentTypeRepository) GetActiveEmploymentTypes() ([]models.EmploymentTypeMaster, error) {
	if !models.HasEmploymentTypeMasterTable() {
		return []models.EmploymentTypeMaster{}, nil
	}
	var types []models.EmploymentTypeMaster
	err := r.db.Table(models.Table("tbl_EmploymentTypeMaster")).
		Where("IsActive = ?", true).
		Order("EmploymentTypeID ASC").
		Find(&types).Error
	return types, err
}

func (r *employmentTypeRepository) GetByID(id uint8) (*models.EmploymentTypeMaster, error) {
	if !models.HasEmploymentTypeMasterTable() {
		return nil, gorm.ErrRecordNotFound
	}
	var et models.EmploymentTypeMaster
	err := r.db.Table(models.Table("tbl_EmploymentTypeMaster")).
		Where("EmploymentTypeID = ? AND IsActive = ?", id, true).
		First(&et).Error
	if err != nil {
		return nil, err
	}
	return &et, nil
}
