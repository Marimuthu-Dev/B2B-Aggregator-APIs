package domain

import "b2b-diagnostic-aggregator/apis/internal/timeutil"

type Test struct {
	TestID        int64
	TestName      string
	Category      string
	IsActive      bool
	CreatedBy     int64
	CreatedOn     timeutil.ISTTime
	LastUpdatedBy int64
	LastUpdatedOn timeutil.ISTTime
}
