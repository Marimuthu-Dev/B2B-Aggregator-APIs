package dto

type EmploymentTypeResponse struct {
	EmploymentTypeID   uint8  `json:"employmentTypeId"`
	EmploymentTypeName string `json:"employmentTypeName"`
	IsActive           bool   `json:"isActive"`
}
