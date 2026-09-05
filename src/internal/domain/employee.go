package domain

import "b2b-diagnostic-aggregator/apis/internal/timeutil"

type Employee struct {
	UID            int64
	FullName       string
	Address        string
	CityID         int16
	StateID        int16
	Pincode        string
	MobileNumber   string
	CompanyEmailID string
	Designation    string
	Department        string
	IsActive         bool
	IsPriceViewAccess bool
	CreatedBy         int64
	CreatedOn      timeutil.ISTTime
	LastUpdatedBy  int64
	LastUpdatedOn  timeutil.ISTTime
}
