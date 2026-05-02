package dto

import (
	"strings"
	"time"

	"b2b-diagnostic-aggregator/apis/internal/domain"
	"b2b-diagnostic-aggregator/apis/internal/timeutil"
)

type LeadRequest struct {
	LeadID         int64      `binding:"omitempty"`
	ClientID       int64      `binding:"required"`
	PatientID      string     `binding:"omitempty"`
	PatientName    string     `binding:"required"`
	Age            int8       `binding:"required"`
	Gender         string     `binding:"required"`
	PackageID      int        `binding:"required"`
	ContactNumber  string     `binding:"required"`
	Emailid        string     `binding:"required"`
	Address        string     `binding:"required"`
	CityID         int32      `binding:"required"`
	StateID        int32      `binding:"required"`
	Pincode        string     `binding:"required"`
	CollectionType string     `json:"CollectionType" binding:"required"`
	LeadStatusID   int8       // omitted or 0 → service uses domain.LeadStatusIDDefault (1)
	AppointmentAt  *time.Time `json:"AppointmentAt"`
	LabID          *int64     `json:"LabID"`
}

type BulkUpdateLeadStatusRequest struct {
	LeadIDs      []int64 `json:"leadIds" binding:"required"`
	LeadStatusID int8    `json:"leadStatusId" binding:"required"`
	// LabID when set is applied to every lead in the batch (same as PUT /leads/{id} with LabID).
	LabID *int64 `json:"LabID,omitempty"`
	// AppointmentAt when set is applied to every lead; validated like PUT /leads/{id} (not in the past).
	AppointmentAt *time.Time `json:"AppointmentAt,omitempty"`
}

// ApproveLeadRequest is the body for POST /api/v1/leads/{id}/reports/approve (report fit / hold / unfit).
type ApproveLeadRequest struct {
	Status        string `json:"status" binding:"required"`
	Remarks       string `json:"remarks"`
	AllowDownload bool   `json:"allowDownload"`
	// IsFitCertificateToBeGenerated is whether a fitness certificate PDF should be generated for this lead (MediAdmin.tbl_Leads.IsFitCertificateTobeGenerated). Pointer + required so JSON must include true or false explicitly.
	IsFitCertificateToBeGenerated *bool `json:"isFitCertificateToBeGenerated" binding:"required"`
}

// LeadUpdateRequest is for PUT; all fields optional. At least one must be set.
type LeadUpdateRequest struct {
	ClientID       *int64     `json:"ClientID"`
	PatientName    *string    `json:"PatientName"`
	Age            *int8      `json:"Age"`
	Gender         *string    `json:"Gender"`
	PackageID      *int       `json:"PackageID"`
	ContactNumber  *string    `json:"ContactNumber"`
	Emailid        *string    `json:"Emailid"`
	Address        *string    `json:"Address"`
	CityID         *int32     `json:"CityID"`
	StateID        *int32     `json:"StateID"`
	Pincode        *string    `json:"Pincode"`
	CollectionType *string    `json:"CollectionType"`
	LeadStatusID   *int8      `json:"LeadStatusID"`
	AppointmentAt  *time.Time `json:"AppointmentAt"`
	LabID          *int64     `json:"LabID"`
}

func (r LeadUpdateRequest) HasAtLeastOneField() bool {
	return r.ClientID != nil || r.PatientName != nil || r.Age != nil || r.Gender != nil ||
		r.PackageID != nil || r.ContactNumber != nil || r.Emailid != nil || r.Address != nil ||
		r.CityID != nil || r.StateID != nil || r.Pincode != nil || r.CollectionType != nil || r.LeadStatusID != nil ||
		r.AppointmentAt != nil || r.LabID != nil
}

func (r LeadRequest) ToDomain() domain.Lead {
	l := domain.Lead{
		LeadID:         r.LeadID,
		ClientID:       r.ClientID,
		PatientID:      r.PatientID,
		PatientName:    r.PatientName,
		Age:            r.Age,
		Gender:         r.Gender,
		PackageID:      r.PackageID,
		ContactNumber:  r.ContactNumber,
		Emailid:        r.Emailid,
		Address:        r.Address,
		CityID:         r.CityID,
		StateID:        r.StateID,
		Pincode:        r.Pincode,
		CollectionType: strings.TrimSpace(r.CollectionType),
		LeadStatusID:   r.LeadStatusID,
		LabID:          r.LabID,
	}
	if r.AppointmentAt != nil {
		at := timeutil.StoredFromTime(*r.AppointmentAt)
		l.AppointmentAt = &at
	}
	return l
}
