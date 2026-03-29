package repository

import "time"

// MouExpiryDateRange filters MOUEndDate inclusive; bounds are date-only (UTC midnight). Use only when both are set.
type MouExpiryDateRange struct {
	From time.Time
	To   time.Time
}

type ClientListFilter struct {
	Page           int
	PageSize       int
	SortBy         string
	SortOrder      string
	CityID         *int8
	StateID        *int8
	IsActive       *bool
	MouStatuses    []string // active, expired, expiringSoon — OR semantics
	MouExpiryRange *MouExpiryDateRange
	Search         string
}

type LabListFilter struct {
	Page           int
	PageSize       int
	SortBy         string
	SortOrder      string
	CityID         *int8
	StateID        *int8
	IsActive       *bool
	MouStatuses    []string
	MouExpiryRange *MouExpiryDateRange
	Search         string
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
