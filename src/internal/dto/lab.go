package dto

import (
	"b2b-diagnostic-aggregator/apis/internal/domain"
	"time"
)

type LabRequest struct {
	LabID                      int64      `binding:"omitempty"`
	LabName                    string     `binding:"required"`
	Address                    *string    `binding:"omitempty"`
	CityID                     *int8      `binding:"omitempty"`
	StateID                    *int8      `binding:"omitempty"`
	Pincode                    *string    `binding:"omitempty"`
	ContactPerson1Name         *string    `binding:"omitempty"`
	ContactPerson1Number       *string    `binding:"omitempty"`
	ContactPerson1EmailID      *string    `binding:"omitempty"`
	ContactPerson1Designation  *string    `binding:"omitempty"`
	ContactPerson1Name1        *string    `binding:"omitempty"`
	ContactPerson1Number1      *string    `binding:"omitempty"`
	ContactPerson1EmailID1     *string    `binding:"omitempty"`
	ContactPerson1Designation1 *string    `binding:"omitempty"`
	CategoryID                 *int8      `binding:"omitempty"`
	GSTIN_UIN                  *string    `binding:"omitempty"`
	PANNumber                  *string    `binding:"omitempty"`
	MOUStartDate               *time.Time `binding:"omitempty"`
	MOUEndDate                 *time.Time `binding:"omitempty"`
	AccreditationID            *int8      `binding:"omitempty"`
	CollectionTypes            *string    `binding:"omitempty"`
	ServicesID                 *string    `binding:"omitempty"`
	CollectionPincodes         *string    `binding:"omitempty"`
	IsActive                   *bool      `binding:"omitempty"`
	CreatedBy                  *int64     `binding:"omitempty"`
	CreatedOn                  *time.Time `binding:"omitempty"`
	LastUpdatedBy              *int64     `binding:"omitempty"`
	LastUpdatedOn              *time.Time `binding:"omitempty"`
}

func (r LabRequest) ToDomain() domain.Lab {
	return domain.Lab{
		LabID:                      r.LabID,
		LabName:                    r.LabName,
		Address:                    r.Address,
		CityID:                     r.CityID,
		StateID:                    r.StateID,
		Pincode:                    r.Pincode,
		ContactPerson1Name:         r.ContactPerson1Name,
		ContactPerson1Number:       r.ContactPerson1Number,
		ContactPerson1EmailID:      r.ContactPerson1EmailID,
		ContactPerson1Designation:  r.ContactPerson1Designation,
		ContactPerson1Name1:        r.ContactPerson1Name1,
		ContactPerson1Number1:      r.ContactPerson1Number1,
		ContactPerson1EmailID1:     r.ContactPerson1EmailID1,
		ContactPerson1Designation1: r.ContactPerson1Designation1,
		CategoryID:                 r.CategoryID,
		GSTIN_UIN:                  r.GSTIN_UIN,
		PANNumber:                  r.PANNumber,
		MOUStartDate:               r.MOUStartDate,
		MOUEndDate:                 r.MOUEndDate,
		AccreditationID:            r.AccreditationID,
		CollectionTypes:            r.CollectionTypes,
		ServicesID:                 r.ServicesID,
		CollectionPincodes:         r.CollectionPincodes,
		IsActive:                   r.IsActive,
		CreatedBy:                  r.CreatedBy,
		CreatedOn:                  r.CreatedOn,
		LastUpdatedBy:              r.LastUpdatedBy,
		LastUpdatedOn:              r.LastUpdatedOn,
	}
}
