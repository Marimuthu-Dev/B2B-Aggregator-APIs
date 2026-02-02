package domain

import "time"

type Test struct {
	TestID        int
	TestName      string
	Category      string
	IsActive      bool
	CreatedBy     int64
	CreatedOn     time.Time
	LastUpdatedBy int64
	LastUpdatedOn time.Time
}
