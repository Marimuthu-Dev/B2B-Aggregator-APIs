package dto

type ClientListQuery struct {
	PaginationQuery
	CityID   *int8 `form:"cityId" binding:"omitempty,min=1"`
	StateID  *int8 `form:"stateId" binding:"omitempty,min=1"`
	IsActive *bool `form:"isActive" binding:"omitempty"`
}

type LabListQuery struct {
	PaginationQuery
	CityID   *int8 `form:"cityId" binding:"omitempty,min=1"`
	StateID  *int8 `form:"stateId" binding:"omitempty,min=1"`
	IsActive *bool `form:"isActive" binding:"omitempty"`
}

type LeadListQuery struct {
	PaginationQuery
	ClientID  *int64 `form:"clientId" binding:"omitempty,min=1"`
	StatusID  *int8  `form:"statusId" binding:"omitempty,min=1"`
	PackageID *int   `form:"packageId" binding:"omitempty,min=1"`
}

type PackageListQuery struct {
	PaginationQuery
	IsActive *bool  `form:"isActive" binding:"omitempty"`
	Search   string `form:"search" binding:"omitempty"`
}
