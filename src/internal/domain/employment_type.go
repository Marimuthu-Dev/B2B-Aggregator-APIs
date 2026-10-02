package domain

import "time"

type EmploymentType struct {
	EmploymentTypeID   uint8     `json:"employmentTypeId"`
	EmploymentTypeName string    `json:"employmentTypeName"`
	IsActive           bool      `json:"isActive"`
	CreatedOn          time.Time `json:"createdOn"`
}
