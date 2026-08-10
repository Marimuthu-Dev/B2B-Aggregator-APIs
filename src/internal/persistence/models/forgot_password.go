package models

import "time"

// ForgotPassword maps to MediAdmin.tbl_ForgotPassword.
type ForgotPassword struct {
	UID               int64      `gorm:"primaryKey;column:UID;autoIncrement"`
	UserID            int64      `gorm:"column:UserId;not null"`
	UserType          string     `gorm:"column:UserType;type:varchar(10);not null"`
	ForgetPasswordKey string     `gorm:"column:ForgetPasswordKey;type:nvarchar(255);not null"`
	CreatedOn         time.Time  `gorm:"column:CreatedOn;not null"`
	ExpiryTimestamp   time.Time  `gorm:"column:ExpiryTimestamp;not null"`
	IsPasswordChanged bool       `gorm:"column:IsPasswordChanged;not null"`
	PasswordUpdatedOn *time.Time `gorm:"column:PasswordUpdatedOn"`
}

func (ForgotPassword) TableName() string {
	return Table("tbl_ForgotPassword")
}
