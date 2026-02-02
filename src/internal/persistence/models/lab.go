package models

import "time"

type Lab struct {
	LabID                      int64      `gorm:"primaryKey;column:LabID;autoIncrement"`
	LabName                    string     `gorm:"column:LabName;type:varchar(75);not null"`
	Address                    *string    `gorm:"column:Address;type:varchar(150)"`
	CityID                     *int8      `gorm:"column:CityID"`
	StateID                    *int8      `gorm:"column:StateID"`
	Pincode                    *string    `gorm:"column:Pincode;type:varchar(6)"`
	ContactPerson1Name         *string    `gorm:"column:ContactPerson1Name;type:varchar(15)"`
	ContactPerson1Number       *string    `gorm:"column:ContactPerson1Number;type:varchar(10)"`
	ContactPerson1EmailID      *string    `gorm:"column:ContactPerson1EmailID;type:varchar(75)"`
	ContactPerson1Designation  *string    `gorm:"column:ContactPerson1Designation;type:varchar(15)"`
	ContactPerson1Name1        *string    `gorm:"column:ContactPerson1Name1;type:varchar(15)"`
	ContactPerson1Number1      *string    `gorm:"column:ContactPerson1Number1;type:varchar(10)"`
	ContactPerson1EmailID1     *string    `gorm:"column:ContactPerson1EmailID1;type:varchar(75)"`
	ContactPerson1Designation1 *string    `gorm:"column:ContactPerson1Designation1;type:varchar(15)"`
	CategoryID                 *int8      `gorm:"column:CategoryID"`
	GSTIN_UIN                  *string    `gorm:"column:GSTIN_UIN;type:varchar(20)"`
	PANNumber                  *string    `gorm:"column:PANNumber;type:varchar(10)"`
	MOUStartDate               *time.Time `gorm:"column:MOUStartDate;type:date"`
	MOUEndDate                 *time.Time `gorm:"column:MOUEndDate;type:date"`
	AccreditationID            *int8      `gorm:"column:AccreditationID"`
	CollectionTypes            *string    `gorm:"column:CollectionTypes;type:varchar(10)"` // JSON string
	ServicesID                 *string    `gorm:"column:ServicesID;type:varchar(10)"`      // JSON string
	CollectionPincodes         *string    `gorm:"column:CollectionPincodes;type:text"`     // JSON string
	IsActive                   *bool      `gorm:"column:IsActive"`
	CreatedBy                  *int64     `gorm:"column:CreatedBy"`
	CreatedOn                  *time.Time `gorm:"column:CreatedOn;default:GETDATE()"`
	LastUpdatedBy              *int64     `gorm:"column:LastUpdatedBy"`
	LastUpdatedOn              *time.Time `gorm:"column:LastUpdatedOn;default:GETDATE()"`
}

func (Lab) TableName() string {
	return "MediAdmin.tbl_LabMaster"
}
