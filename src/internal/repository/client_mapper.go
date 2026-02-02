package repository

import (
	"b2b-diagnostic-aggregator/apis/internal/domain"
	persistencemodels "b2b-diagnostic-aggregator/apis/internal/persistence/models"
)

func mapClientToDomain(p persistencemodels.Client) domain.Client {
	return domain.Client{
		ClientID:                  p.ClientID,
		ClientName:                p.ClientName,
		Address:                   p.Address,
		CityID:                    p.CityID,
		StateID:                   p.StateID,
		Pincode:                   p.Pincode,
		ContactPerson1Name:        p.ContactPerson1Name,
		ContactPerson1Number:      p.ContactPerson1Number,
		ContactPerson1EmailID:     p.ContactPerson1EmailID,
		ContactPerson1Designation: p.ContactPerson1Designation,
		ContactPerson2Name:        p.ContactPerson2Name,
		ContactPerson2Number:      p.ContactPerson2Number,
		ContactPerson2EmailID:     p.ContactPerson2EmailID,
		ContactPerson2Designation: p.ContactPerson2Designation,
		CategoryID:                p.CategoryID,
		GSTIN_UIN:                 p.GSTIN_UIN,
		PANNumber:                 p.PANNumber,
		BusinessVertical:          p.BusinessVertical,
		BillingName:               p.BillingName,
		BillingAdderss:            p.BillingAdderss,
		BillingPincode:            p.BillingPincode,
		IsAcitve:                  p.IsAcitve,
		CreatedBy:                 p.CreatedBy,
		CreatedOn:                 p.CreatedOn,
		LastUpdatedBy:             p.LastUpdatedBy,
		LastUpdatedOn:             p.LastUpdatedOn,
	}
}

func mapClientToPersistence(d domain.Client) persistencemodels.Client {
	return persistencemodels.Client{
		ClientID:                  d.ClientID,
		ClientName:                d.ClientName,
		Address:                   d.Address,
		CityID:                    d.CityID,
		StateID:                   d.StateID,
		Pincode:                   d.Pincode,
		ContactPerson1Name:        d.ContactPerson1Name,
		ContactPerson1Number:      d.ContactPerson1Number,
		ContactPerson1EmailID:     d.ContactPerson1EmailID,
		ContactPerson1Designation: d.ContactPerson1Designation,
		ContactPerson2Name:        d.ContactPerson2Name,
		ContactPerson2Number:      d.ContactPerson2Number,
		ContactPerson2EmailID:     d.ContactPerson2EmailID,
		ContactPerson2Designation: d.ContactPerson2Designation,
		CategoryID:                d.CategoryID,
		GSTIN_UIN:                 d.GSTIN_UIN,
		PANNumber:                 d.PANNumber,
		BusinessVertical:          d.BusinessVertical,
		BillingName:               d.BillingName,
		BillingAdderss:            d.BillingAdderss,
		BillingPincode:            d.BillingPincode,
		IsAcitve:                  d.IsAcitve,
		CreatedBy:                 d.CreatedBy,
		CreatedOn:                 d.CreatedOn,
		LastUpdatedBy:             d.LastUpdatedBy,
		LastUpdatedOn:             d.LastUpdatedOn,
	}
}

func mapClientsToDomain(clients []persistencemodels.Client) []domain.Client {
	if len(clients) == 0 {
		return nil
	}
	mapped := make([]domain.Client, len(clients))
	for i, client := range clients {
		mapped[i] = mapClientToDomain(client)
	}
	return mapped
}
