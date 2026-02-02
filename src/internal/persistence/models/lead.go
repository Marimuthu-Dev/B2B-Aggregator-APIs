package models

import "time"

type Lead struct {
	LeadID        int64     `gorm:"primaryKey;column:LeadID;autoIncrement"`
	ClientID      int64     `gorm:"column:ClientID;not null"`
	PatientID     string    `gorm:"column:PatientID;type:varchar(20);not null"`
	PatientName   string    `gorm:"column:PatientName;type:varchar(100);not null"`
	Age           int8      `gorm:"column:Age;not null"`
	Gender        string    `gorm:"column:Gender;type:varchar(1);not null"`
	PackageID     int       `gorm:"column:PackageID;not null"`
	ContactNumber string    `gorm:"column:ContactNumber;type:varchar(10);not null"`
	Emailid       string    `gorm:"column:Emailid;type:varchar(75);not null"`
	Address       string    `gorm:"column:Address;type:varchar(150);not null"`
	CityID        int8      `gorm:"column:CityID;not null"`
	StateID       int8      `gorm:"column:StateID;not null"`
	Pincode       string    `gorm:"column:Pincode;type:varchar(6);not null"`
	LeadStatusID  int8      `gorm:"column:LeadStatusID;not null"`
	CreatedBy     int64     `gorm:"column:CreatedBy;not null"`
	CreatedOn     time.Time `gorm:"column:CreatedOn;not null;default:GETDATE()"`
	LastUpdatedBy int64     `gorm:"column:LastUpdatedBy;not null"`
	LastUpdatedOn time.Time `gorm:"column:LastUpdatedOn;not null;default:GETDATE()"`
}

func (Lead) TableName() string {
	return "MediAdmin.tbl_Leads"
}

type LeadHistory struct {
	UID       int64     `gorm:"primaryKey;column:UID;autoIncrement"`
	LeadID    int64     `gorm:"column:LeadID;not null"`
	Action    string    `gorm:"column:Action;type:varchar(25);not null"`
	CreatedBy int64     `gorm:"column:CreatedBy;not null"`
	CreatedOn time.Time `gorm:"column:CreatedOn;not null;default:GETDATE()"`
}

func (LeadHistory) TableName() string {
	return "MediAdmin.tbl_LeadsHistory"
}
