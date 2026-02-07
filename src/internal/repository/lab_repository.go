package repository

import (
	"fmt"

	"b2b-diagnostic-aggregator/apis/internal/domain"
	persistencemodels "b2b-diagnostic-aggregator/apis/internal/persistence/models"

	"gorm.io/gorm"
)

type LabRepository interface {
	FindAll() ([]domain.Lab, error)
	List(filter LabListFilter) ([]domain.Lab, int64, error)
	FindByID(id int64) (*domain.Lab, error)
	ExistsByID(id int64) (bool, error)
	Create(l *domain.Lab) error
	Update(l *domain.Lab) error
	Delete(id int64) error
	FindAllActive() ([]domain.Lab, error)
	FindByContactNumber(contactNumber string) (*domain.Lab, error)
	FindByCity(cityID int8) ([]domain.Lab, error)
	FindByState(stateID int8) ([]domain.Lab, error)
}

type labRepository struct {
	db *gorm.DB
}

func NewLabRepository(db *gorm.DB) LabRepository {
	return &labRepository{db: db}
}

func (r *labRepository) FindAll() ([]domain.Lab, error) {
	var labs []persistencemodels.Lab
	err := r.db.Find(&labs).Error
	return mapLabsToDomain(labs), err
}

func (r *labRepository) List(filter LabListFilter) ([]domain.Lab, int64, error) {
	query := r.db.Model(&persistencemodels.Lab{})
	if filter.CityID != nil {
		query = query.Where("CityID = ?", *filter.CityID)
	}
	if filter.StateID != nil {
		query = query.Where("StateID = ?", *filter.StateID)
	}
	if filter.IsActive != nil {
		query = query.Where("IsActive = ?", *filter.IsActive)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	sortColumn := mapLabSortColumn(filter.SortBy)
	order := normalizeSortOrder(filter.SortOrder)
	offset := (filter.Page - 1) * filter.PageSize

	var labs []persistencemodels.Lab
	err := query.Order(sortColumn + " " + order).Limit(filter.PageSize).Offset(offset).Find(&labs).Error
	return mapLabsToDomain(labs), total, err
}

func mapLabSortColumn(sortBy string) string {
	switch sortBy {
	case "name":
		return "LabName"
	case "cityId":
		return "CityID"
	case "stateId":
		return "StateID"
	case "createdOn":
		return "CreatedOn"
	default:
		return "LabID"
	}
}

func (r *labRepository) FindByID(id int64) (*domain.Lab, error) {
	var l persistencemodels.Lab
	err := r.db.First(&l, id).Error
	if err != nil {
		return nil, err
	}
	domainLab := mapLabToDomain(l)
	return &domainLab, nil
}

func (r *labRepository) ExistsByID(id int64) (bool, error) {
	var count int64
	if err := r.db.Model(&persistencemodels.Lab{}).Where("LabID = ?", id).Limit(1).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *labRepository) Create(l *domain.Lab) error {
	persist := mapLabToPersistence(*l)
	if err := r.db.Create(&persist).Error; err != nil {
		return err
	}
	*l = mapLabToDomain(persist)
	return nil
}

func (r *labRepository) Update(l *domain.Lab) error {
	persist := mapLabToPersistence(*l)
	if err := r.db.Save(&persist).Error; err != nil {
		return err
	}
	*l = mapLabToDomain(persist)
	return nil
}

func (r *labRepository) Delete(id int64) error {
	return r.db.Delete(&persistencemodels.Lab{}, id).Error
}

func (r *labRepository) FindAllActive() ([]domain.Lab, error) {
	var labs []persistencemodels.Lab
	err := r.db.Where("IsActive = ?", true).Find(&labs).Error
	return mapLabsToDomain(labs), err
}

func (r *labRepository) FindByContactNumber(contactNumber string) (*domain.Lab, error) {
	fmt.Printf("[LOGIN] Repository.Lab.FindByContactNumber: entry contactNumber=%s\n", contactNumber)
	var l persistencemodels.Lab
	err := r.db.Where("ContactPerson1Number = ? OR ContactPerson1Number1 = ?", contactNumber, contactNumber).First(&l).Error
	if err != nil {
		fmt.Printf("[LOGIN] Repository.Lab.FindByContactNumber: not found err=%v\n", err)
		return nil, err
	}
	domainLab := mapLabToDomain(l)
	fmt.Printf("[LOGIN] Repository.Lab.FindByContactNumber: found LabID=%d\n", domainLab.LabID)
	return &domainLab, nil
}

func (r *labRepository) FindByCity(cityID int8) ([]domain.Lab, error) {
	var labs []persistencemodels.Lab
	err := r.db.Where("CityID = ?", cityID).Find(&labs).Error
	return mapLabsToDomain(labs), err
}

func (r *labRepository) FindByState(stateID int8) ([]domain.Lab, error) {
	var labs []persistencemodels.Lab
	err := r.db.Where("StateID = ?", stateID).Find(&labs).Error
	return mapLabsToDomain(labs), err
}
