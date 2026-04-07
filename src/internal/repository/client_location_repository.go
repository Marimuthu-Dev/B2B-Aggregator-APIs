package repository

import (
	"b2b-diagnostic-aggregator/apis/internal/domain"
	"b2b-diagnostic-aggregator/apis/internal/timeutil"
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
		Address:          derefString(p.Address),
		Pincode:          derefString(p.Pincode),
		CityID:           p.CityID,
		StateID:          p.StateID,
		IsActive:         derefBoolDefaultTrue(p.IsActive),
		CreatedBy:        derefInt64Zero(p.CreatedBy),
		CreatedOn:        timeutil.FromTime(p.CreatedOn),
		LastUpdatedBy:    derefInt64Zero(p.LastUpdatedBy),
		LastUpdatedOn:    timeutil.FromTime(p.LastUpdatedOn),
	}
}

func mapClientLocationToPersistence(d domain.ClientLocation) persistencemodels.ClientLocation {
	return persistencemodels.ClientLocation{
		ClientLocationID: d.ClientLocationID,
		ClientID:         d.ClientID,
		Address:          stringPtrOrNil(d.Address),
		Pincode:          stringPtrOrNil(d.Pincode),
		CityID:           d.CityID,
		StateID:          d.StateID,
		IsActive:         boolPtr(d.IsActive),
		CreatedBy:        int64PtrOrNil(d.CreatedBy),
		CreatedOn:        d.CreatedOn.ToTime(),
		LastUpdatedBy:    int64PtrOrNil(d.LastUpdatedBy),
		LastUpdatedOn:    d.LastUpdatedOn.ToTime(),
	}
}

func derefBoolDefaultTrue(p *bool) bool {
	if p == nil {
		return true
	}
	return *p
}

func derefInt64Zero(p *int64) int64 {
	if p == nil {
		return 0
	}
	return *p
}

func boolPtr(b bool) *bool {
	x := b
	return &x
}

func int64PtrOrNil(v int64) *int64 {
	if v == 0 {
		return nil
	}
	x := v
	return &x
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
