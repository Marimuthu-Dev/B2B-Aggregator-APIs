package repository

import (
	"b2b-diagnostic-aggregator/apis/internal/domain"
	persistencemodels "b2b-diagnostic-aggregator/apis/internal/persistence/models"
	"b2b-diagnostic-aggregator/apis/internal/timeutil"
)

func mapStoreToDomain(p persistencemodels.Store) domain.Store {
	return domain.Store{
		StoreID:       p.StoreID,
		ClientID:      p.ClientID,
		StoreName:     p.StoreName,
		Address:       p.Address,
		CityID:        p.CityID,
		StateID:       p.StateID,
		Pincode:       p.Pincode,
		ContactNumber: p.ContactNumber,
		EmailID:       p.EmailID,
		IsActive:      p.IsActive,
		CreatedBy:     p.CreatedBy,
		CreatedOn:     timeutil.FromTime(p.CreatedOn),
		LastUpdatedBy: p.LastUpdatedBy,
		LastUpdatedOn: timeutil.FromTime(p.LastUpdatedOn),
	}
}

func mapStoreToPersistence(d domain.Store) persistencemodels.Store {
	return persistencemodels.Store{
		StoreID:       d.StoreID,
		ClientID:      d.ClientID,
		StoreName:     d.StoreName,
		Address:       d.Address,
		CityID:        d.CityID,
		StateID:       d.StateID,
		Pincode:       d.Pincode,
		ContactNumber: d.ContactNumber,
		EmailID:       d.EmailID,
		IsActive:      d.IsActive,
		CreatedBy:     d.CreatedBy,
		CreatedOn:     d.CreatedOn.ToTime(),
		LastUpdatedBy: d.LastUpdatedBy,
		LastUpdatedOn: d.LastUpdatedOn.ToTime(),
	}
}

func mapStoresToDomain(stores []persistencemodels.Store) []domain.Store {
	if len(stores) == 0 {
		return nil
	}
	mapped := make([]domain.Store, len(stores))
	for i, store := range stores {
		mapped[i] = mapStoreToDomain(store)
	}
	return mapped
}
