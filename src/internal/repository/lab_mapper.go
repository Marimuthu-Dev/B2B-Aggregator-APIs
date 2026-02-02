package repository

import (
	"b2b-diagnostic-aggregator/apis/internal/domain"
	persistencemodels "b2b-diagnostic-aggregator/apis/internal/persistence/models"
)

func mapLabToDomain(p persistencemodels.Lab) domain.Lab {
	return domain.Lab{
		LabID:                      p.LabID,
		LabName:                    p.LabName,
		Address:                    p.Address,
		CityID:                     p.CityID,
		StateID:                    p.StateID,
		Pincode:                    p.Pincode,
		ContactPerson1Name:         p.ContactPerson1Name,
		ContactPerson1Number:       p.ContactPerson1Number,
		ContactPerson1EmailID:      p.ContactPerson1EmailID,
		ContactPerson1Designation:  p.ContactPerson1Designation,
		ContactPerson1Name1:        p.ContactPerson1Name1,
		ContactPerson1Number1:      p.ContactPerson1Number1,
		ContactPerson1EmailID1:     p.ContactPerson1EmailID1,
		ContactPerson1Designation1: p.ContactPerson1Designation1,
		CategoryID:                 p.CategoryID,
		GSTIN_UIN:                  p.GSTIN_UIN,
		PANNumber:                  p.PANNumber,
		MOUStartDate:               p.MOUStartDate,
		MOUEndDate:                 p.MOUEndDate,
		AccreditationID:            p.AccreditationID,
		CollectionTypes:            p.CollectionTypes,
		ServicesID:                 p.ServicesID,
		CollectionPincodes:         p.CollectionPincodes,
		IsActive:                   p.IsActive,
		CreatedBy:                  p.CreatedBy,
		CreatedOn:                  p.CreatedOn,
		LastUpdatedBy:              p.LastUpdatedBy,
		LastUpdatedOn:              p.LastUpdatedOn,
	}
}

func mapLabToPersistence(d domain.Lab) persistencemodels.Lab {
	return persistencemodels.Lab{
		LabID:                      d.LabID,
		LabName:                    d.LabName,
		Address:                    d.Address,
		CityID:                     d.CityID,
		StateID:                    d.StateID,
		Pincode:                    d.Pincode,
		ContactPerson1Name:         d.ContactPerson1Name,
		ContactPerson1Number:       d.ContactPerson1Number,
		ContactPerson1EmailID:      d.ContactPerson1EmailID,
		ContactPerson1Designation:  d.ContactPerson1Designation,
		ContactPerson1Name1:        d.ContactPerson1Name1,
		ContactPerson1Number1:      d.ContactPerson1Number1,
		ContactPerson1EmailID1:     d.ContactPerson1EmailID1,
		ContactPerson1Designation1: d.ContactPerson1Designation1,
		CategoryID:                 d.CategoryID,
		GSTIN_UIN:                  d.GSTIN_UIN,
		PANNumber:                  d.PANNumber,
		MOUStartDate:               d.MOUStartDate,
		MOUEndDate:                 d.MOUEndDate,
		AccreditationID:            d.AccreditationID,
		CollectionTypes:            d.CollectionTypes,
		ServicesID:                 d.ServicesID,
		CollectionPincodes:         d.CollectionPincodes,
		IsActive:                   d.IsActive,
		CreatedBy:                  d.CreatedBy,
		CreatedOn:                  d.CreatedOn,
		LastUpdatedBy:              d.LastUpdatedBy,
		LastUpdatedOn:              d.LastUpdatedOn,
	}
}

func mapLabsToDomain(labs []persistencemodels.Lab) []domain.Lab {
	if len(labs) == 0 {
		return nil
	}
	mapped := make([]domain.Lab, len(labs))
	for i, lab := range labs {
		mapped[i] = mapLabToDomain(lab)
	}
	return mapped
}
