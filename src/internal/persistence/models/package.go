package models

import "time"

type Package struct {
	PackageID     int64     `gorm:"primaryKey;column:PackageID;autoIncrement"`
	PackageName   string    `gorm:"column:PackageName;type:varchar(75);not null"`
	Description   *string   `gorm:"column:Description;type:varchar(250)"`
	IsActive      bool      `gorm:"column:IsActive;not null;default:true"`
	CreatedBy     int64     `gorm:"column:CreatedBy;not null"`
	CreatedOn     time.Time `gorm:"column:CreatedOn;not null;default:GETDATE()"`
	LastUpdatedBy int64     `gorm:"column:LastUpdatedBy;not null"`
	LastUpdatedOn time.Time `gorm:"column:LastUpdatedOn;not null;default:GETDATE()"`
}

func (Package) TableName() string {
	return Table("tbl_PackageMaster")
}
