package domain

import "b2b-diagnostic-aggregator/apis/internal/timeutil"

type Employee struct {
	UID            int64
	FullName       string
	Address        string
	CityID         int8
	StateID        int8
	Pincode        string
	MobileNumber   string
	CompanyEmailID string
	Designation    string
	Department        string
	IsPriceViewAccess bool
	CreatedBy         int64
	CreatedOn      timeutil.ISTTime
	LastUpdatedBy  int64
	LastUpdatedOn  timeutil.ISTTime
}
