package models

import "time"

type Test struct {
	TestID        int64     `gorm:"primaryKey;column:TestID;autoIncrement"`
	TestName      string    `gorm:"column:TestName;type:varchar(50);not null"`
	Category      string    `gorm:"column:Category;type:varchar(20);not null"`
	IsActive      bool      `gorm:"column:IsActive;not null;default:true"`
	CreatedBy     int64     `gorm:"column:CreatedBy;not null"`
	CreatedOn     time.Time `gorm:"column:CreatedOn;not null;default:GETDATE()"`
	LastUpdatedBy int64     `gorm:"column:LastUpdatedBy;not null"`
	LastUpdatedOn time.Time `gorm:"column:LastUpdatedOn;not null;default:GETDATE()"`
}

func (Test) TableName() string {
	return Table("tbl_TestMaster")
}
