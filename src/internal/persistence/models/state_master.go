package models

// StateMaster maps [MediAdmin].[tbl_StateMaster] / [MedLyfe].[tbl_StateMaster].
type StateMaster struct {
	StateID   int32  `gorm:"primaryKey;column:StateID"`
	StateName string `gorm:"column:StateName;type:varchar(100);not null"`
	IsActive  *bool  `gorm:"column:IsActive"`
}

func (StateMaster) TableName() string {
	return Table("tbl_StateMaster")
}
