package repository

import (
	"b2b-diagnostic-aggregator/apis/internal/domain"
	persistencemodels "b2b-diagnostic-aggregator/apis/internal/persistence/models"
)

func mapLeadToDomain(p persistencemodels.Lead) domain.Lead {
	return domain.Lead{
		LeadID:        p.LeadID,
		ClientID:      p.ClientID,
		PatientID:     p.PatientID,
		PatientName:   p.PatientName,
		Age:           p.Age,
		Gender:        p.Gender,
		PackageID:     p.PackageID,
		ContactNumber: p.ContactNumber,
		Emailid:       p.Emailid,
		Address:       p.Address,
		CityID:        p.CityID,
		StateID:       p.StateID,
		Pincode:       p.Pincode,
		LeadStatusID:  p.LeadStatusID,
		CreatedBy:     p.CreatedBy,
		CreatedOn:     p.CreatedOn,
		LastUpdatedBy: p.LastUpdatedBy,
		LastUpdatedOn: p.LastUpdatedOn,
	}
}

func mapLeadToPersistence(d domain.Lead) persistencemodels.Lead {
	return persistencemodels.Lead{
		LeadID:        d.LeadID,
		ClientID:      d.ClientID,
		PatientID:     d.PatientID,
		PatientName:   d.PatientName,
		Age:           d.Age,
		Gender:        d.Gender,
		PackageID:     d.PackageID,
		ContactNumber: d.ContactNumber,
		Emailid:       d.Emailid,
		Address:       d.Address,
		CityID:        d.CityID,
		StateID:       d.StateID,
		Pincode:       d.Pincode,
		LeadStatusID:  d.LeadStatusID,
		CreatedBy:     d.CreatedBy,
		CreatedOn:     d.CreatedOn,
		LastUpdatedBy: d.LastUpdatedBy,
		LastUpdatedOn: d.LastUpdatedOn,
	}
}

func mapLeadsToDomain(leads []persistencemodels.Lead) []domain.Lead {
	if len(leads) == 0 {
		return nil
	}
	mapped := make([]domain.Lead, len(leads))
	for i, lead := range leads {
		mapped[i] = mapLeadToDomain(lead)
	}
	return mapped
}

func mapLeadHistoryToDomain(p persistencemodels.LeadHistory) domain.LeadHistory {
	return domain.LeadHistory{
		UID:       p.UID,
		LeadID:    p.LeadID,
		Action:    p.Action,
		CreatedBy: p.CreatedBy,
		CreatedOn: p.CreatedOn,
	}
}

func mapLeadHistoryToPersistence(d domain.LeadHistory) persistencemodels.LeadHistory {
	return persistencemodels.LeadHistory{
		UID:       d.UID,
		LeadID:    d.LeadID,
		Action:    d.Action,
		CreatedBy: d.CreatedBy,
		CreatedOn: d.CreatedOn,
	}
}

func mapLeadHistoriesToDomain(histories []persistencemodels.LeadHistory) []domain.LeadHistory {
	if len(histories) == 0 {
		return nil
	}
	mapped := make([]domain.LeadHistory, len(histories))
	for i, history := range histories {
		mapped[i] = mapLeadHistoryToDomain(history)
	}
	return mapped
}

func mapLeadHistoriesToPersistence(histories []domain.LeadHistory) []persistencemodels.LeadHistory {
	if len(histories) == 0 {
		return nil
	}
	mapped := make([]persistencemodels.LeadHistory, len(histories))
	for i, history := range histories {
		mapped[i] = mapLeadHistoryToPersistence(history)
	}
	return mapped
}
