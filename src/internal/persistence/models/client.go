package models

import "time"

type Client struct {
	ClientID                  int64     `gorm:"primaryKey;column:ClientID;autoIncrement"`
	ClientName                string    `gorm:"column:ClientName;type:varchar(150);not null"`
	Address                   string    `gorm:"column:Address;type:varchar(150);not null"`
	CityID                    int8      `gorm:"column:CityID;not null"`
	StateID                   int8      `gorm:"column:StateID;not null"`
	Pincode                   string    `gorm:"column:Pincode;type:varchar(6);not null"`
	ContactPerson1Name        string    `gorm:"column:ContactPerson1Name;type:varchar(75);not null"`
	ContactPerson1Number      string    `gorm:"column:ContactPerson1Number;type:varchar(10);not null"`
	ContactPerson1EmailID     string    `gorm:"column:ContactPerson1EmailID;type:varchar(75);not null"`
	ContactPerson1Designation string    `gorm:"column:ContactPerson1Designation;type:varchar(25);not null"`
	ContactPerson2Name        *string   `gorm:"column:ContactPerson2Name;type:varchar(75)"`
	ContactPerson2Number      *string   `gorm:"column:ContactPerson2Number;type:varchar(10)"`
	ContactPerson2EmailID     *string   `gorm:"column:ContactPerson2EmailID;type:varchar(75)"`
	ContactPerson2Designation *string   `gorm:"column:ContactPerson2Designation;type:varchar(25)"`
	CategoryID                *int8     `gorm:"column:CategoryID"`
	GSTIN_UIN                 *string   `gorm:"column:GSTIN_UIN;type:varchar(20)"`
	PANNumber                 *string   `gorm:"column:PANNumber;type:varchar(10)"`
	BusinessVertical          string    `gorm:"column:BusinessVertical;type:varchar(15);not null"`
	BillingName               *string   `gorm:"column:BillingName;type:varchar(75)"`
	BillingAdderss            *string   `gorm:"column:BillingAdderss;type:varchar(150)"`
	BillingPincode            *string   `gorm:"column:BillingPincode;type:varchar(6)"`
	IsAcitve                  bool      `gorm:"column:IsAcitve;not null"` // Note: typo in DB (IsAcitve)
	CreatedBy                 int64     `gorm:"column:CreatedBy;not null"`
	CreatedOn                 time.Time `gorm:"column:CreatedOn;not null;default:GETDATE()"`
	LastUpdatedBy             int64     `gorm:"column:LastUpdatedBy;not null"`
	LastUpdatedOn             time.Time `gorm:"column:LastUpdatedOn;not null;default:GETDATE()"`
}

func (Client) TableName() string {
	return "MediAdmin.tbl_ClientMaster"
}
