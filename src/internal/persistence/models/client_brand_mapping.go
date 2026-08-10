package models

import "time"

// ClientBrandMapping maps MediAdmin.tbl_ClientBrandMapping.
type ClientBrandMapping struct {
	UID             int64     `gorm:"primaryKey;column:UID;autoIncrement"`
	ClientID        int64     `gorm:"column:ClientID;not null"`
	BrandName       string    `gorm:"column:BrandName;type:varchar(20);not null"`
	IsActive        bool      `gorm:"column:IsActive;not null"`
	CreatedBy       int64     `gorm:"column:CreatedBy;not null"`
	CreatedOn       time.Time `gorm:"column:CreatedOn;not null;default:GETDATE()"`
	LastUpdatedBy   int64     `gorm:"column:LastUpdatedBy;not null"`
	LastUpdatedOn   time.Time `gorm:"column:LastUpdatedOn;not null;default:GETDATE()"`
}

func (ClientBrandMapping) TableName() string {
	return Table("tbl_ClientBrandMapping")
}
