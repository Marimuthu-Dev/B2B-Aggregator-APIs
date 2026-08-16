package repository

import (
	"time"

	"b2b-diagnostic-aggregator/apis/internal/domain"
)

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
	CityID         *uint8
	StateID        *uint8
	IsActive       *bool
	MouStatuses    []string
	MouExpiryRange *MouExpiryDateRange
	Search         string
}

type LeadListFilter struct {
	Page             int
	PageSize         int
	SortBy           string
	SortOrder        string
	LeadID           *int64
	ClientID         *int64
	LabID            *int64
	StatusID         *int8
	PackageID        *int
	CollectionType   *string
	StoreID          *string
	StoreMasterID    *int64
	// RestrictToStoreID is set from a store JWT (userType 4). Matches StoreMasterID or StoreID varchar.
	RestrictToStoreID *int64
	Search           string
	FitnessStatus    domain.LeadListFitnessFilter
	// AppointmentAtMin is inclusive lower bound (appointmentAtFrom at 00:00:00 IST); nil = no lower filter.
	AppointmentAtMin *time.Time
	// AppointmentAtMax is inclusive upper bound (appointmentAtTo at 23:59:59.999999999 IST); nil = no upper filter.
	AppointmentAtMax *time.Time
}

type PackageListFilter struct {
	Page      int
	PageSize  int
	SortBy    string
	SortOrder string
	IsActive  *bool
	Search    string
}
