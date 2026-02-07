package repository

import (
	persistencemodels "b2b-diagnostic-aggregator/apis/internal/persistence/models"

	"gorm.io/gorm"
)

type PackageClientMappingRepository interface {
	Create(m *persistencemodels.PackageClientMapping) error
	FindByPackageAndClient(packageID int, clientID int64) (*persistencemodels.PackageClientMapping, error)
	FindByID(id int) (*persistencemodels.PackageClientMapping, error)
	FindAll() ([]persistencemodels.PackageClientMapping, error)
	FindByPackageID(packageID int) ([]persistencemodels.PackageClientMapping, error)
	Update(m *persistencemodels.PackageClientMapping) error
}

type packageClientMappingRepository struct {
	db *gorm.DB
}

func NewPackageClientMappingRepository(db *gorm.DB) PackageClientMappingRepository {
	return &packageClientMappingRepository{db: db}
}

func (r *packageClientMappingRepository) Create(m *persistencemodels.PackageClientMapping) error {
	return r.db.Create(m).Error
}

func (r *packageClientMappingRepository) FindByPackageAndClient(packageID int, clientID int64) (*persistencemodels.PackageClientMapping, error) {
	var m persistencemodels.PackageClientMapping
	err := r.db.Where("PackageID = ? AND ClientID = ? AND IsActive = ?", packageID, clientID, true).First(&m).Error
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *packageClientMappingRepository) FindByID(id int) (*persistencemodels.PackageClientMapping, error) {
	var m persistencemodels.PackageClientMapping
	err := r.db.First(&m, id).Error
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *packageClientMappingRepository) FindAll() ([]persistencemodels.PackageClientMapping, error) {
	var list []persistencemodels.PackageClientMapping
	err := r.db.Find(&list).Error
	return list, err
}

func (r *packageClientMappingRepository) FindByPackageID(packageID int) ([]persistencemodels.PackageClientMapping, error) {
	var list []persistencemodels.PackageClientMapping
	err := r.db.Where("PackageID = ?", packageID).Find(&list).Error
	return list, err
}

func (r *packageClientMappingRepository) Update(m *persistencemodels.PackageClientMapping) error {
	return r.db.Save(m).Error
}
