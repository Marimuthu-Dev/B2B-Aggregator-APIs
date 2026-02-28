package dto

import (
	"b2b-diagnostic-aggregator/apis/internal/domain"
)

type LeadRequest struct {
	LeadID        int64     `binding:"omitempty"`
	ClientID      int64     `binding:"required"`
	PatientID     string    `binding:"omitempty"`
	PatientName   string    `binding:"required"`
	Age           int8      `binding:"required"`
	Gender        string    `binding:"required"`
	PackageID     int       `binding:"required"`
	ContactNumber string    `binding:"required"`
	Emailid       string    `binding:"required"`
	Address       string    `binding:"required"`
	CityID        int8      `binding:"required"`
	StateID       int8      `binding:"required"`
	Pincode       string    `binding:"required"`
	LeadStatusID int8 // defaults to 0 when omitted in POST payload
}

type BulkUpdateLeadStatusRequest struct {
	LeadIDs      []int64 `json:"leadIds" binding:"required"`
	LeadStatusID int8    `json:"leadStatusId" binding:"required"`
}

// LeadUpdateRequest is for PUT; all fields optional. At least one must be set.
type LeadUpdateRequest struct {
	ClientID      *int64  `json:"ClientID"`
	PatientName   *string `json:"PatientName"`
	Age           *int8   `json:"Age"`
	Gender        *string `json:"Gender"`
	PackageID     *int    `json:"PackageID"`
	ContactNumber *string `json:"ContactNumber"`
	Emailid       *string `json:"Emailid"`
	Address       *string `json:"Address"`
	CityID        *int8   `json:"CityID"`
	StateID       *int8   `json:"StateID"`
	Pincode       *string `json:"Pincode"`
	LeadStatusID  *int8   `json:"LeadStatusID"`
}

func (r LeadUpdateRequest) HasAtLeastOneField() bool {
	return r.ClientID != nil || r.PatientName != nil || r.Age != nil || r.Gender != nil ||
		r.PackageID != nil || r.ContactNumber != nil || r.Emailid != nil || r.Address != nil ||
		r.CityID != nil || r.StateID != nil || r.Pincode != nil || r.LeadStatusID != nil
}

func (r LeadRequest) ToDomain() domain.Lead {
	return domain.Lead{
		LeadID:        r.LeadID,
		ClientID:      r.ClientID,
		PatientID:     r.PatientID,
		PatientName:   r.PatientName,
		Age:           r.Age,
		Gender:        r.Gender,
		PackageID:     r.PackageID,
		ContactNumber: r.ContactNumber,
		Emailid:       r.Emailid,
		Address:       r.Address,
		CityID:        r.CityID,
		StateID:       r.StateID,
		Pincode:       r.Pincode,
		LeadStatusID:  r.LeadStatusID,
	}
}
