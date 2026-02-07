package models

import "time"

type ClientLocation struct {
	ClientLocationID int64     `gorm:"primaryKey;column:ClientLocationID;autoIncrement"`
	ClientID         int64     `gorm:"column:ClientID;not null"`
	Address          string    `gorm:"column:Address;type:varchar(150)"`
	Pincode          string    `gorm:"column:Pincode;type:varchar(6)"`
	CityID           int8      `gorm:"column:CityID;not null"`
	StateID          int8      `gorm:"column:StateID;not null"`
	IsActive         bool      `gorm:"column:IsActive"`
	CreatedBy        int64     `gorm:"column:CreatedBy"`
	CreatedOn        time.Time `gorm:"column:CreatedOn;default:GETDATE()"`
	LastUpdatedBy    int64     `gorm:"column:LastUpdatedBy"`
	LastUpdatedOn    time.Time `gorm:"column:LastUpdatedOn;default:GETDATE()"`
}

func (ClientLocation) TableName() string {
	return "MediAdmin.tbl_ClientLocationMaster"
}
