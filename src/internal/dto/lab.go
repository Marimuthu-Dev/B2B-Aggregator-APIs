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
	CollectionPincodes *string `binding:"omitempty"`
	IsActive *bool `binding:"omitempty"`
}

// LabUpdateRequest is for PUT; all fields optional. At least one must be set.
type LabUpdateRequest struct {
	LabName                    *string    `json:"LabName"`
	Address                    *string   `json:"Address"`
	CityID                     *int8     `json:"CityID"`
	StateID                    *int8     `json:"StateID"`
	Pincode                    *string   `json:"Pincode"`
	ContactPerson1Name         *string   `json:"ContactPerson1Name"`
	ContactPerson1Number       *string   `json:"ContactPerson1Number"`
	ContactPerson1EmailID      *string   `json:"ContactPerson1EmailID"`
	ContactPerson1Designation  *string   `json:"ContactPerson1Designation"`
	ContactPerson1Name1        *string   `json:"ContactPerson1Name1"`
	ContactPerson1Number1      *string   `json:"ContactPerson1Number1"`
	ContactPerson1EmailID1     *string   `json:"ContactPerson1EmailID1"`
	ContactPerson1Designation1 *string   `json:"ContactPerson1Designation1"`
	CategoryID                 *int8     `json:"CategoryID"`
	GSTIN_UIN                  *string   `json:"GSTIN_UIN"`
	PANNumber                  *string   `json:"PANNumber"`
	MOUStartDate               *time.Time `json:"MOUStartDate"`
	MOUEndDate                 *time.Time `json:"MOUEndDate"`
	AccreditationID            *int8     `json:"AccreditationID"`
	CollectionTypes            *string   `json:"CollectionTypes"`
	ServicesID                 *string   `json:"ServicesID"`
	CollectionPincodes         *string   `json:"CollectionPincodes"`
	IsActive                   *bool     `json:"IsActive"`
}

func (r LabUpdateRequest) HasAtLeastOneField() bool {
	return r.LabName != nil || r.Address != nil || r.CityID != nil || r.StateID != nil || r.Pincode != nil ||
		r.ContactPerson1Name != nil || r.ContactPerson1Number != nil || r.ContactPerson1EmailID != nil || r.ContactPerson1Designation != nil ||
		r.ContactPerson1Name1 != nil || r.ContactPerson1Number1 != nil || r.ContactPerson1EmailID1 != nil || r.ContactPerson1Designation1 != nil ||
		r.CategoryID != nil || r.GSTIN_UIN != nil || r.PANNumber != nil || r.MOUStartDate != nil || r.MOUEndDate != nil ||
		r.AccreditationID != nil || r.CollectionTypes != nil || r.ServicesID != nil || r.CollectionPincodes != nil || r.IsActive != nil
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
		ContactPerson1Number:      r.ContactPerson1Number,
		ContactPerson1EmailID:     r.ContactPerson1EmailID,
		ContactPerson1Designation:  r.ContactPerson1Designation,
		ContactPerson1Name1:        r.ContactPerson1Name1,
		ContactPerson1Number1:     r.ContactPerson1Number1,
		ContactPerson1EmailID1:    r.ContactPerson1EmailID1,
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
	}
}
