package dto

import (
	"b2b-diagnostic-aggregator/apis/internal/domain"
	"time"
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
	BillingPincode            *string `binding:"omitempty"`
	IsAcitve                  bool    `binding:"omitempty"`
	CreatedBy                 int64   `binding:"required"`
	CreatedOn                 *time.Time `binding:"omitempty"`
	LastUpdatedBy             int64   `binding:"required"`
	LastUpdatedOn             *time.Time `binding:"omitempty"`
}

func (r ClientRequest) ToDomain() domain.Client {
	var createdOn time.Time
	if r.CreatedOn != nil {
		createdOn = *r.CreatedOn
	}
	var lastUpdatedOn time.Time
	if r.LastUpdatedOn != nil {
		lastUpdatedOn = *r.LastUpdatedOn
	}

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
		CreatedBy:                 r.CreatedBy,
		CreatedOn:                 createdOn,
		LastUpdatedBy:             r.LastUpdatedBy,
		LastUpdatedOn:             lastUpdatedOn,
	}
}
