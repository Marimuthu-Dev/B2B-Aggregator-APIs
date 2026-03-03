package domain

import "b2b-diagnostic-aggregator/apis/internal/timeutil"

type Client struct {
	ClientID                  int64
	ClientName                string
	Address                   string
	CityID                    int8
	StateID                   int8
	Pincode                   string
	ContactPerson1Name        string
	ContactPerson1Number      string
	ContactPerson1EmailID     string
	ContactPerson1Designation string
	ContactPerson2Name        *string
	ContactPerson2Number      *string
	ContactPerson2EmailID     *string
	ContactPerson2Designation *string
	CategoryID                *int8
	GSTIN_UIN                 *string
	PANNumber                 *string
	BusinessVertical          string
	BillingName               *string
	BillingAdderss            *string
	BillingPincode            *string
	ClientTypeID              *int8
	IsAcitve                  bool
	MOUStartDate              *timeutil.ISTTime
	MOUEndDate                *timeutil.ISTTime
	CreatedBy                 int64
	CreatedOn                 timeutil.ISTTime
	LastUpdatedBy             int64
	LastUpdatedOn             timeutil.ISTTime
}

type ClientLocation struct {
	ClientLocationID int64
	ClientID         int64
	Address          string
	Pincode          string
	CityID           int8
	StateID          int8
	IsActive         bool
	CreatedBy        int64
	CreatedOn        timeutil.ISTTime
	LastUpdatedBy    int64
	LastUpdatedOn    timeutil.ISTTime
}
