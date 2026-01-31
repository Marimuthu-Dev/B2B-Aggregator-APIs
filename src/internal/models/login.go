package models

import (
	"time"
)

type Login struct {
	RecordID      int64     `gorm:"primaryKey;column:RecordID;autoIncrement"`
	UserID        int64     `gorm:"column:UserID;not null;unique"`
	Pwd           string    `gorm:"column:Pwd;type:varchar(100);not null"`
	UserType      string    `gorm:"column:UserType;type:varchar(10);not null"`
	CreatedOn     time.Time `gorm:"column:CreatedOn;not null;default:GETDATE()"`
	LastUpdatedOn time.Time `gorm:"column:LastUpdatedOn;not null;default:GETDATE()"`
}

func (Login) TableName() string {
	return "MediAdmin.tbl_Login"
}

type LoginRequest struct {
	UserID   int64  `json:"userId" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	User         interface{} `json:"user"`
	Token        string      `json:"token"`
	RefreshToken string      `json:"refreshToken"`
}
