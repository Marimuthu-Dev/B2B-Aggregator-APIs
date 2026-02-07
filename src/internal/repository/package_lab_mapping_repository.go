package repository

import (
	persistencemodels "b2b-diagnostic-aggregator/apis/internal/persistence/models"

	"gorm.io/gorm"
)

type PackageLabMappingRepository interface {
	Create(m *persistencemodels.PackageLabMapping) error
	FindByPackageAndLab(packageID int, labID int64) (*persistencemodels.PackageLabMapping, error)
	FindByID(id int) (*persistencemodels.PackageLabMapping, error)
	FindAll() ([]persistencemodels.PackageLabMapping, error)
	FindByPackageID(packageID int) ([]persistencemodels.PackageLabMapping, error)
	Update(m *persistencemodels.PackageLabMapping) error
}

type packageLabMappingRepository struct {
	db *gorm.DB
}

func NewPackageLabMappingRepository(db *gorm.DB) PackageLabMappingRepository {
	return &packageLabMappingRepository{db: db}
}

func (r *packageLabMappingRepository) Create(m *persistencemodels.PackageLabMapping) error {
	return r.db.Create(m).Error
}

func (r *packageLabMappingRepository) FindByPackageAndLab(packageID int, labID int64) (*persistencemodels.PackageLabMapping, error) {
	var m persistencemodels.PackageLabMapping
	err := r.db.Where("PackageID = ? AND LabID = ? AND IsActive = ?", packageID, labID, true).First(&m).Error
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *packageLabMappingRepository) FindByID(id int) (*persistencemodels.PackageLabMapping, error) {
	var m persistencemodels.PackageLabMapping
	err := r.db.First(&m, id).Error
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *packageLabMappingRepository) FindAll() ([]persistencemodels.PackageLabMapping, error) {
	var list []persistencemodels.PackageLabMapping
	err := r.db.Find(&list).Error
	return list, err
}

func (r *packageLabMappingRepository) FindByPackageID(packageID int) ([]persistencemodels.PackageLabMapping, error) {
	var list []persistencemodels.PackageLabMapping
	err := r.db.Where("PackageID = ?", packageID).Find(&list).Error
	return list, err
}

func (r *packageLabMappingRepository) Update(m *persistencemodels.PackageLabMapping) error {
	return r.db.Save(m).Error
}
