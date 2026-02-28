package dto

import (
	"b2b-diagnostic-aggregator/apis/internal/domain"
)

type EmployeeRequest struct {
	FullName       string `json:"FullName" binding:"required"`
	Address        string `json:"Address" binding:"required"`
	CityID         int8   `json:"CityID" binding:"required"`
	StateID        int8   `json:"StateID" binding:"required"`
	Pincode        string `json:"Pincode" binding:"required"`
	MobileNumber   string `json:"MobileNumber" binding:"required"`
	CompanyEmailID string `json:"CompanyEmailID" binding:"required"`
	Designation  string `json:"Designation" binding:"required"`
	Department   string `json:"Department" binding:"required"`
}

// EmployeeUpdateRequest is for PUT; all fields optional. At least one must be set.
type EmployeeUpdateRequest struct {
	FullName       *string `json:"FullName"`
	Address        *string `json:"Address"`
	CityID         *int8   `json:"CityID"`
	StateID        *int8   `json:"StateID"`
	Pincode        *string `json:"Pincode"`
	MobileNumber   *string `json:"MobileNumber"`
	CompanyEmailID *string `json:"CompanyEmailID"`
	Designation    *string `json:"Designation"`
	Department     *string `json:"Department"`
}

func (r EmployeeUpdateRequest) HasAtLeastOneField() bool {
	return r.FullName != nil || r.Address != nil || r.CityID != nil || r.StateID != nil || r.Pincode != nil ||
		r.MobileNumber != nil || r.CompanyEmailID != nil || r.Designation != nil || r.Department != nil
}

func (r EmployeeRequest) ToDomain() domain.Employee {
	return domain.Employee{
		FullName:       r.FullName,
		Address:        r.Address,
		CityID:         r.CityID,
		StateID:        r.StateID,
		Pincode:        r.Pincode,
		MobileNumber:   r.MobileNumber,
		CompanyEmailID: r.CompanyEmailID,
		Designation:    r.Designation,
		Department:     r.Department,
	}
}
