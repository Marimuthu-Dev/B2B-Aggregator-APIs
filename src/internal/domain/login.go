package domain

import "time"

type Login struct {
	RecordID      int64
	UserID        int64
	Pwd           string
	UserType      string
	CreatedOn     time.Time
	LastUpdatedOn time.Time
}
