package models

import "time"

type Store struct {
	StoreID       int64     `gorm:"primaryKey;column:StoreID;autoIncrement"`
	ClientID      int64     `gorm:"column:ClientID;not null"`
	StoreName     string    `gorm:"column:StoreName;type:varchar(150);not null"`
	Address       string    `gorm:"column:Address;type:varchar(150);not null"`
	CityID        int8      `gorm:"column:CityID;not null"`
	StateID       int8      `gorm:"column:StateID;not null"`
	Pincode       string    `gorm:"column:Pincode;type:varchar(6);not null"`
	ContactNumber string    `gorm:"column:ContactNumber;type:varchar(10);not null"`
	EmailID       string    `gorm:"column:EmailID;type:varchar(75);not null"`
	IsActive      bool      `gorm:"column:IsActive;not null"`
	CreatedBy     int64     `gorm:"column:CreatedBy;not null"`
	CreatedOn     time.Time `gorm:"column:CreatedOn;not null;default:GETDATE()"`
	LastUpdatedBy int64     `gorm:"column:LastUpdatedBy;not null"`
	LastUpdatedOn time.Time `gorm:"column:LastUpdatedOn;not null;default:GETDATE()"`
}

func (Store) TableName() string {
	return Table("tbl_StoreMaster")
}
