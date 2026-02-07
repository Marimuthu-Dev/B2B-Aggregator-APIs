package models

import "time"

type Employee struct {
	UID            int64     `gorm:"primaryKey;column:UID;autoIncrement"`
	FullName       string    `gorm:"column:FullName;type:varchar(100);not null"`
	Address        string    `gorm:"column:Address;type:varchar(200);not null"`
	CityID         int8      `gorm:"column:CityID;not null"`
	StateID        int8      `gorm:"column:StateID;not null"`
	Pincode        string    `gorm:"column:Pincode;type:varchar(6);not null"`
	MobileNumber   string    `gorm:"column:MobileNumber;type:varchar(10);not null"`
	CompanyEmailID string    `gorm:"column:CompanyEmailID;type:varchar(75);not null"`
	Designation    string    `gorm:"column:Designation;type:varchar(20);not null"`
	Department     string    `gorm:"column:Department;type:varchar(15);not null"`
	CreatedBy      int64     `gorm:"column:CreatedBy;not null"`
	CreatedOn      time.Time `gorm:"column:CreatedOn;not null;default:GETDATE()"`
	LastUpdatedBy  int64     `gorm:"column:LastUpdatedBy;not null"`
	LastUpdatedOn  time.Time `gorm:"column:LastUpdatedOn;not null;default:GETDATE()"`
}

func (Employee) TableName() string {
	return "MediAdmin.tbl_EmployeeMaster"
}
