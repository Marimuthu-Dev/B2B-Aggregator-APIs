package repository

type ClientListFilter struct {
	Page      int
	PageSize  int
	SortBy    string
	SortOrder string
	CityID    *int8
	StateID   *int8
	IsActive  *bool
}

type LabListFilter struct {
	Page      int
	PageSize  int
	SortBy    string
	SortOrder string
	CityID    *int8
	StateID   *int8
	IsActive  *bool
}

type LeadListFilter struct {
	Page      int
	PageSize  int
	SortBy    string
	SortOrder string
	ClientID  *int64
	StatusID  *int8
	PackageID *int
}

type PackageListFilter struct {
	Page      int
	PageSize  int
	SortBy    string
	SortOrder string
	IsActive  *bool
	Search    string
}
