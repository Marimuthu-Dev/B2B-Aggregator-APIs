package domain

import "time"

type Lab struct {
	LabID                      int64
	LabName                    string
	Address                    *string
	CityID                     *int8
	StateID                    *int8
	Pincode                    *string
	ContactPerson1Name         *string
	ContactPerson1Number       *string
	ContactPerson1EmailID      *string
	ContactPerson1Designation  *string
	ContactPerson1Name1        *string
	ContactPerson1Number1      *string
	ContactPerson1EmailID1     *string
	ContactPerson1Designation1 *string
	CategoryID                 *int8
	GSTIN_UIN                  *string
	PANNumber                  *string
	MOUStartDate               *time.Time
	MOUEndDate                 *time.Time
	AccreditationID            *int8
	CollectionTypes            *string
	ServicesID                 *string
	CollectionPincodes         *string
	IsActive                   *bool
	CreatedBy                  *int64
	CreatedOn                  *time.Time
	LastUpdatedBy              *int64
	LastUpdatedOn              *time.Time
}
