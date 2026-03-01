package dto

import (
	"time"

	"b2b-diagnostic-aggregator/apis/internal/domain"
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
	MOUStartDate               *FlexDate       `json:"MOUStartDate" binding:"omitempty"`   // accepts "YYYY-MM-DD" or RFC3339; nil when not sent
	MOUEndDate                 *FlexDate       `json:"MOUEndDate" binding:"omitempty"`
	AccreditationID            *int8           `binding:"omitempty"`
	CollectionTypes            *FlexArrayString `json:"CollectionTypes" binding:"omitempty"`    // accepts string or array e.g. [1,2]
	ServicesID                 *FlexArrayString `json:"ServicesID" binding:"omitempty"`
	CollectionPincodes         *FlexArrayString `json:"CollectionPincodes" binding:"omitempty"`
	IsActive                   *bool            `binding:"omitempty"`
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
	MOUStartDate               *FlexDate        `json:"MOUStartDate"`
	MOUEndDate                 *FlexDate       `json:"MOUEndDate"`
	AccreditationID            *int8           `json:"AccreditationID"`
	CollectionTypes            *FlexArrayString `json:"CollectionTypes"`
	ServicesID                 *FlexArrayString `json:"ServicesID"`
	CollectionPincodes         *FlexArrayString `json:"CollectionPincodes"`
	IsActive                   *bool           `json:"IsActive"`
}

func (r LabUpdateRequest) HasAtLeastOneField() bool {
	return r.LabName != nil || r.Address != nil || r.CityID != nil || r.StateID != nil || r.Pincode != nil ||
		r.ContactPerson1Name != nil || r.ContactPerson1Number != nil || r.ContactPerson1EmailID != nil || r.ContactPerson1Designation != nil ||
		r.ContactPerson1Name1 != nil || r.ContactPerson1Number1 != nil || r.ContactPerson1EmailID1 != nil || r.ContactPerson1Designation1 != nil ||
		r.CategoryID != nil || r.GSTIN_UIN != nil || r.PANNumber != nil || r.MOUStartDate != nil || r.MOUEndDate != nil ||
		r.AccreditationID != nil || r.CollectionTypes != nil || r.ServicesID != nil || r.CollectionPincodes != nil || r.IsActive != nil
}

// Getters for FlexDate/FlexArrayString so service receives *time.Time and *string without depending on flex types.
func (r LabUpdateRequest) GetMOUStartDate() *time.Time     { return r.MOUStartDate.ToTimePtr() }
func (r LabUpdateRequest) GetMOUEndDate() *time.Time       { return r.MOUEndDate.ToTimePtr() }
func (r LabUpdateRequest) GetCollectionTypes() *string    { return r.CollectionTypes.ToStringPtr() }
func (r LabUpdateRequest) GetServicesID() *string          { return r.ServicesID.ToStringPtr() }
func (r LabUpdateRequest) GetCollectionPincodes() *string   { return r.CollectionPincodes.ToStringPtr() }

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
		ContactPerson1Designation1:  r.ContactPerson1Designation1,
		CategoryID:                 r.CategoryID,
		GSTIN_UIN:                  r.GSTIN_UIN,
		PANNumber:                  r.PANNumber,
		MOUStartDate:               r.MOUStartDate.ToTimePtr(),
		MOUEndDate:                 r.MOUEndDate.ToTimePtr(),
		AccreditationID:            r.AccreditationID,
		CollectionTypes:            r.CollectionTypes.ToStringPtr(),
		ServicesID:                 r.ServicesID.ToStringPtr(),
		CollectionPincodes:         r.CollectionPincodes.ToStringPtr(),
		IsActive:                   r.IsActive,
	}
}
