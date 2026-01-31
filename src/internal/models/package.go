package models

import (
	"time"
)

type Package struct {
	PackageID     int       `gorm:"primaryKey;column:PackageID;autoIncrement"`
	PackageName   string    `gorm:"column:PackageName;type:varchar(500);not null"`
	Description   string    `gorm:"column:Description;type:text"`
	IsActive      bool      `gorm:"column:IsActive;not null;default:true"`
	CreatedBy     int64     `gorm:"column:CreatedBy;not null"`
	CreatedOn     time.Time `gorm:"column:CreatedOn;not null;default:GETDATE()"`
	LastUpdatedBy int64     `gorm:"column:LastUpdatedBy;not null"`
	LastUpdatedOn time.Time `gorm:"column:LastUpdatedOn;not null;default:GETDATE()"`
}

func (Package) TableName() string {
	return "MediAdmin.tbl_PackageMaster"
}
