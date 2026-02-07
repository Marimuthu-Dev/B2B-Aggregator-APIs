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
	Designation    string `json:"Designation" binding:"required"`
	Department     string `json:"Department" binding:"required"`
	CreatedBy      int64  `json:"CreatedBy" binding:"required"`
	LastUpdatedBy  int64  `json:"LastUpdatedBy" binding:"required"`
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
		CreatedBy:      r.CreatedBy,
		LastUpdatedBy:  r.LastUpdatedBy,
	}
}
