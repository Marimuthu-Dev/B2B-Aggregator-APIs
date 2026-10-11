package models

// CityMaster maps [MediAdmin].[tbl_CityMaster] / [MedLyfe].[tbl_CityMaster].
type CityMaster struct {
	CityID   int32  `gorm:"primaryKey;column:CityID"`
	CityName string `gorm:"column:CityName;type:varchar(100);not null"`
	StateID  int32  `gorm:"column:StateID;not null"`
	IsActive *bool  `gorm:"column:IsActive"`
}

func (CityMaster) TableName() string {
	return Table("tbl_CityMaster")
}
