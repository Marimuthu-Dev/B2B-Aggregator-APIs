package models

import "time"

type EmploymentTypeMaster struct {
	EmploymentTypeID   uint8     `gorm:"primaryKey;column:EmploymentTypeID;autoIncrement"`
	EmploymentTypeName string    `gorm:"column:EmploymentTypeName;type:varchar(50);not null"`
	IsActive           bool      `gorm:"column:IsActive;not null;default:true"`
	CreatedOn          time.Time `gorm:"column:CreatedOn;not null;default:GETDATE()"`
}

func (EmploymentTypeMaster) TableName() string {
	return Table("tbl_EmploymentTypeMaster")
}
