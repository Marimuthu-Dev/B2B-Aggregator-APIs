package dto

import (
	"b2b-diagnostic-aggregator/apis/internal/domain"
	"time"
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
	LeadStatusID  int8      `binding:"required"`
	CreatedBy     int64     `binding:"required"`
	CreatedOn     *time.Time `binding:"omitempty"`
	LastUpdatedBy int64     `binding:"required"`
	LastUpdatedOn *time.Time `binding:"omitempty"`
}

type BulkUpdateLeadStatusRequest struct {
	LeadIDs       []int64 `json:"leadIds" binding:"required"`
	LeadStatusID  int8    `json:"leadStatusId" binding:"required"`
	LastUpdatedBy int64   `json:"lastUpdatedBy" binding:"required"`
}

func (r LeadRequest) ToDomain() domain.Lead {
	var createdOn time.Time
	if r.CreatedOn != nil {
		createdOn = *r.CreatedOn
	}
	var lastUpdatedOn time.Time
	if r.LastUpdatedOn != nil {
		lastUpdatedOn = *r.LastUpdatedOn
	}

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
		CreatedBy:     r.CreatedBy,
		CreatedOn:     createdOn,
		LastUpdatedBy: r.LastUpdatedBy,
		LastUpdatedOn: lastUpdatedOn,
	}
}
