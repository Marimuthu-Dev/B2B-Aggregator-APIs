package models

import (
	"time"
)

type PackageTestMapping struct {
	PackageTestID int       `gorm:"primaryKey;column:PackageTestID;autoIncrement"`
	PackageID     int       `gorm:"column:PackageID;not null"`
	TestID        int       `gorm:"column:TestID;not null"`
	IsActive      bool      `gorm:"column:IsActive;not null;default:true"`
	CreatedBy     int64     `gorm:"column:CreatedBy;not null"`
	CreatedOn     time.Time `gorm:"column:CreatedOn;not null;default:GETDATE()"`
	LastUpdatedBy int64     `gorm:"column:LastUpdatedBy;not null"`
	LastUpdatedOn time.Time `gorm:"column:LastUpdatedOn;not null;default:GETDATE()"`
}

func (PackageTestMapping) TableName() string {
	return "MediAdmin.tbl_PackageTestMapping"
}

type PackageClientMapping struct {
	PackageClientID int       `gorm:"primaryKey;column:PackageClientID;autoIncrement"`
	PackageID       int       `gorm:"column:PackageID;not null"`
	ClientID        int64     `gorm:"column:ClientID;not null"`
	Price           float64   `gorm:"column:Price;not null"`
	IsActive        bool      `gorm:"column:IsActive;not null;default:true"`
	CreatedBy       int64     `gorm:"column:CreatedBy;not null"`
	CreatedOn       time.Time `gorm:"column:CreatedOn;not null;default:GETDATE()"`
	LastUpdatedBy   int64     `gorm:"column:LastUpdatedBy;not null"`
	LastUpdatedOn   time.Time `gorm:"column:LastUpdatedOn;not null;default:GETDATE()"`
}

func (PackageClientMapping) TableName() string {
	return "MediAdmin.tbl_PackageClientMapping"
}

type PackageLabMapping struct {
	PackageLabID  int       `gorm:"primaryKey;column:PackageLabID;autoIncrement"`
	PackageID     int       `gorm:"column:PackageID;not null"`
	LabID         int64     `gorm:"column:LabID;not null"`
	Price         float64   `gorm:"column:Price;not null"`
	IsActive      bool      `gorm:"column:IsActive;not null;default:true"`
	CreatedBy     int64     `gorm:"column:CreatedBy;not null"`
	CreatedOn     time.Time `gorm:"column:CreatedOn;not null;default:GETDATE()"`
	LastUpdatedBy int64     `gorm:"column:LastUpdatedBy;not null"`
	LastUpdatedOn time.Time `gorm:"column:LastUpdatedOn;not null;default:GETDATE()"`
}

func (PackageLabMapping) TableName() string {
	return "MediAdmin.tbl_PackageLabMapping"
}
