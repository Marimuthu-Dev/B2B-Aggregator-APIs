package repository

import (
	"b2b-diagnostic-aggregator/apis/internal/domain"
	persistencemodels "b2b-diagnostic-aggregator/apis/internal/persistence/models"

	"gorm.io/gorm"
)

type ClientLocationRepository interface {
	FindByClientID(clientID int64) ([]domain.ClientLocation, error)
	FindByID(id int64) (*domain.ClientLocation, error)
	ExistsByID(id int64) (bool, error)
	Create(l *domain.ClientLocation) error
	Update(l *domain.ClientLocation) error
	Delete(id int64) error
}

type clientLocationRepository struct {
	db *gorm.DB
}

func NewClientLocationRepository(db *gorm.DB) ClientLocationRepository {
	return &clientLocationRepository{db: db}
}

func (r *clientLocationRepository) FindByClientID(clientID int64) ([]domain.ClientLocation, error) {
	var list []persistencemodels.ClientLocation
	if err := r.db.Where("ClientID = ?", clientID).Find(&list).Error; err != nil {
		return nil, err
	}
	return mapClientLocationsToDomain(list), nil
}

func (r *clientLocationRepository) FindByID(id int64) (*domain.ClientLocation, error) {
	var m persistencemodels.ClientLocation
	if err := r.db.First(&m, id).Error; err != nil {
		return nil, err
	}
	d := mapClientLocationToDomain(m)
	return &d, nil
}

func (r *clientLocationRepository) ExistsByID(id int64) (bool, error) {
	var count int64
	if err := r.db.Model(&persistencemodels.ClientLocation{}).Where("ClientLocationID = ?", id).Limit(1).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *clientLocationRepository) Create(l *domain.ClientLocation) error {
	p := mapClientLocationToPersistence(*l)
	if err := r.db.Create(&p).Error; err != nil {
		return err
	}
	*l = mapClientLocationToDomain(p)
	return nil
}

func (r *clientLocationRepository) Update(l *domain.ClientLocation) error {
	p := mapClientLocationToPersistence(*l)
	if err := r.db.Save(&p).Error; err != nil {
		return err
	}
	*l = mapClientLocationToDomain(p)
	return nil
}

func (r *clientLocationRepository) Delete(id int64) error {
	return r.db.Delete(&persistencemodels.ClientLocation{}, id).Error
}

func mapClientLocationToDomain(p persistencemodels.ClientLocation) domain.ClientLocation {
	return domain.ClientLocation{
		ClientLocationID: p.ClientLocationID,
		ClientID:         p.ClientID,
		Address:          p.Address,
		Pincode:          p.Pincode,
		CityID:           p.CityID,
		StateID:          p.StateID,
		IsActive:         p.IsActive,
		CreatedBy:        p.CreatedBy,
		CreatedOn:        p.CreatedOn,
		LastUpdatedBy:    p.LastUpdatedBy,
		LastUpdatedOn:    p.LastUpdatedOn,
	}
}

func mapClientLocationToPersistence(d domain.ClientLocation) persistencemodels.ClientLocation {
	return persistencemodels.ClientLocation{
		ClientLocationID: d.ClientLocationID,
		ClientID:         d.ClientID,
		Address:          d.Address,
		Pincode:          d.Pincode,
		CityID:           d.CityID,
		StateID:          d.StateID,
		IsActive:         d.IsActive,
		CreatedBy:        d.CreatedBy,
		CreatedOn:        d.CreatedOn,
		LastUpdatedBy:    d.LastUpdatedBy,
		LastUpdatedOn:    d.LastUpdatedOn,
	}
}

func mapClientLocationsToDomain(list []persistencemodels.ClientLocation) []domain.ClientLocation {
	if len(list) == 0 {
		return nil
	}
	out := make([]domain.ClientLocation, len(list))
	for i := range list {
		out[i] = mapClientLocationToDomain(list[i])
	}
	return out
}
