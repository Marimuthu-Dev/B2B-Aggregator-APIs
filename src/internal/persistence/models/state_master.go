package models

// StateMaster maps [MediAdmin].[tbl_StateMaster] for joins (read paths).
type StateMaster struct {
	StateID   int32  `gorm:"primaryKey;column:StateID"`
	StateName string `gorm:"column:StateName;type:varchar(100);not null"`
}

func (StateMaster) TableName() string {
	return "MediAdmin.tbl_StateMaster"
}
