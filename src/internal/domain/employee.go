package domain

import "time"

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
	Department     string
	CreatedBy      int64
	CreatedOn      time.Time
	LastUpdatedBy  int64
	LastUpdatedOn  time.Time
}
