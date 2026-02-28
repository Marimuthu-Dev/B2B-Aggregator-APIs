package dto

import (
	"b2b-diagnostic-aggregator/apis/internal/domain"
)

type ClientRequest struct {
	ClientID                  int64   `binding:"omitempty"`
	ClientName                string  `binding:"required"`
	Address                   string  `binding:"required"`
	CityID                    int8    `binding:"required"`
	StateID                   int8    `binding:"required"`
	Pincode                   string  `binding:"required"`
	ContactPerson1Name        string  `binding:"required"`
	ContactPerson1Number      string  `binding:"required"`
	ContactPerson1EmailID     string  `binding:"required"`
	ContactPerson1Designation string  `binding:"required"`
	ContactPerson2Name        *string `binding:"omitempty"`
	ContactPerson2Number      *string `binding:"omitempty"`
	ContactPerson2EmailID     *string `binding:"omitempty"`
	ContactPerson2Designation *string `binding:"omitempty"`
	CategoryID                *int8   `binding:"omitempty"`
	GSTIN_UIN                 *string `binding:"omitempty"`
	PANNumber                 *string `binding:"omitempty"`
	BusinessVertical          string  `binding:"required"`
	BillingName               *string `binding:"omitempty"`
	BillingAdderss            *string `binding:"omitempty"`
	BillingPincode *string `binding:"omitempty"`
	IsAcitve bool `binding:"omitempty"`
}

// ClientUpdateRequest is for PUT; all fields optional. At least one must be set.
type ClientUpdateRequest struct {
	ClientName                *string `json:"ClientName"`
	Address                   *string `json:"Address"`
	CityID                    *int8   `json:"CityID"`
	StateID                   *int8   `json:"StateID"`
	Pincode                   *string `json:"Pincode"`
	ContactPerson1Name        *string `json:"ContactPerson1Name"`
	ContactPerson1Number      *string `json:"ContactPerson1Number"`
	ContactPerson1EmailID     *string `json:"ContactPerson1EmailID"`
	ContactPerson1Designation *string `json:"ContactPerson1Designation"`
	ContactPerson2Name        *string `json:"ContactPerson2Name"`
	ContactPerson2Number      *string `json:"ContactPerson2Number"`
	ContactPerson2EmailID     *string `json:"ContactPerson2EmailID"`
	ContactPerson2Designation *string `json:"ContactPerson2Designation"`
	CategoryID                *int8   `json:"CategoryID"`
	GSTIN_UIN                 *string `json:"GSTIN_UIN"`
	PANNumber                 *string `json:"PANNumber"`
	BusinessVertical          *string `json:"BusinessVertical"`
	BillingName               *string `json:"BillingName"`
	BillingAdderss            *string `json:"BillingAdderss"`
	BillingPincode            *string `json:"BillingPincode"`
	IsAcitve                  *bool   `json:"IsAcitve"`
}

func (r ClientUpdateRequest) HasAtLeastOneField() bool {
	return r.ClientName != nil || r.Address != nil || r.CityID != nil || r.StateID != nil || r.Pincode != nil ||
		r.ContactPerson1Name != nil || r.ContactPerson1Number != nil || r.ContactPerson1EmailID != nil || r.ContactPerson1Designation != nil ||
		r.ContactPerson2Name != nil || r.ContactPerson2Number != nil || r.ContactPerson2EmailID != nil || r.ContactPerson2Designation != nil ||
		r.CategoryID != nil || r.GSTIN_UIN != nil || r.PANNumber != nil || r.BusinessVertical != nil ||
		r.BillingName != nil || r.BillingAdderss != nil || r.BillingPincode != nil || r.IsAcitve != nil
}

type ClientLocationRequest struct {
	Address  string `json:"Address" binding:"omitempty"`
	Pincode  string `json:"Pincode" binding:"omitempty"`
	CityID   int8   `json:"CityID" binding:"required"`
	StateID  int8   `json:"StateID" binding:"required"`
	IsActive *bool  `json:"IsActive" binding:"omitempty"`
}

// ClientLocationUpdateRequest is for PUT; all fields optional. At least one must be set.
type ClientLocationUpdateRequest struct {
	Address  *string `json:"Address"`
	Pincode  *string `json:"Pincode"`
	CityID   *int8   `json:"CityID"`
	StateID  *int8   `json:"StateID"`
	IsActive *bool   `json:"IsActive"`
}

func (r ClientLocationUpdateRequest) HasAtLeastOneField() bool {
	return r.Address != nil || r.Pincode != nil || r.CityID != nil || r.StateID != nil || r.IsActive != nil
}

func (r ClientLocationRequest) ToDomain(clientID int64) domain.ClientLocation {
	isActive := true
	if r.IsActive != nil {
		isActive = *r.IsActive
	}
	return domain.ClientLocation{
		ClientID: clientID,
		Address:  r.Address,
		Pincode:  r.Pincode,
		CityID:   r.CityID,
		StateID:  r.StateID,
		IsActive: isActive,
	}
}

func (r ClientRequest) ToDomain() domain.Client {
	return domain.Client{
		ClientID:                  r.ClientID,
		ClientName:                r.ClientName,
		Address:                   r.Address,
		CityID:                    r.CityID,
		StateID:                   r.StateID,
		Pincode:                   r.Pincode,
		ContactPerson1Name:        r.ContactPerson1Name,
		ContactPerson1Number:      r.ContactPerson1Number,
		ContactPerson1EmailID:     r.ContactPerson1EmailID,
		ContactPerson1Designation: r.ContactPerson1Designation,
		ContactPerson2Name:        r.ContactPerson2Name,
		ContactPerson2Number:      r.ContactPerson2Number,
		ContactPerson2EmailID:     r.ContactPerson2EmailID,
		ContactPerson2Designation: r.ContactPerson2Designation,
		CategoryID:                r.CategoryID,
		GSTIN_UIN:                 r.GSTIN_UIN,
		PANNumber:                 r.PANNumber,
		BusinessVertical:          r.BusinessVertical,
		BillingName:               r.BillingName,
		BillingAdderss:            r.BillingAdderss,
		BillingPincode:            r.BillingPincode,
		IsAcitve:                  r.IsAcitve,
	}
}
