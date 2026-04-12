package domain

import (
	"b2b-diagnostic-aggregator/apis/internal/timeutil"
	"time"
)

type Lab struct {
	LabID                      int64
	LabName                    string
	Address                    *string
	CityID                     *uint8
	StateID                    *uint8
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
	MoUDocumentURL             *string
	AccreditationID            *int8
	AccreditationExpirationDate *time.Time
	CollectionTypes            *string
	ServicesID                 *string
	CollectionPincodes         *string
	LabGrade                   *string
	IsActive                   *bool
	CreatedBy                  *int64
	CreatedOn                  *timeutil.ISTTime
	LastUpdatedBy              *int64
	LastUpdatedOn              *timeutil.ISTTime
}
