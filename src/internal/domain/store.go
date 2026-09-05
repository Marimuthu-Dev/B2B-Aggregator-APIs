package domain

import "b2b-diagnostic-aggregator/apis/internal/timeutil"

type Store struct {
	StoreID       int64
	ClientID      int64
	ClientName    string `json:"ClientName,omitempty"`
	StoreName     string
	Address       string
	CityID        int16
	StateID       int16
	Pincode       string
	ContactNumber string
	EmailID       string
	IsActive      bool
	CreatedBy     int64
	CreatedOn     timeutil.ISTTime
	LastUpdatedBy int64
	LastUpdatedOn timeutil.ISTTime
}
