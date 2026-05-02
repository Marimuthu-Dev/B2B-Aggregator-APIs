package models

// CityMaster maps [MediAdmin].[tbl_CityMaster] for joins (read paths).
type CityMaster struct {
	CityID   int32  `gorm:"primaryKey;column:CityID"`
	CityName string `gorm:"column:CityName;type:varchar(100);not null"`
}

func (CityMaster) TableName() string {
	return "MediAdmin.tbl_CityMaster"
}
