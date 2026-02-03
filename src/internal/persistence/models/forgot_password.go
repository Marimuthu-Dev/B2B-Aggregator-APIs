package models

import "time"

// ForgotPassword maps to MediAdmin.ForgotPassword
type ForgotPassword struct {
	Uid                 int64      `gorm:"primaryKey;column:Uid;autoIncrement"`
	UserID              int64      `gorm:"column:UserID;not null"`
	UserType            string     `gorm:"column:UserType;type:varchar(15)"`
	ForgetPasswordKey   string     `gorm:"column:ForgetPasswordKey;type:varchar(255);not null"`
	CreatedOn           time.Time  `gorm:"column:CreatedOn;not null;default:GETDATE()"`
	ExpiryTimestamp     time.Time  `gorm:"column:ExpiryTimestamp;not null"`
	IsPasswordChanged   bool       `gorm:"column:IsPasswordChanged;not null;default:false"`
	IsPasswordUpdatedOn *time.Time `gorm:"column:IsPasswordUpdatedOn"`
}

func (ForgotPassword) TableName() string {
	return "MediAdmin.ForgotPassword"
}
