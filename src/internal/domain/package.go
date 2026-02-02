package domain

import "time"

type Package struct {
	PackageID     int
	PackageName   string
	Description   string
	IsActive      bool
	CreatedBy     int64
	CreatedOn     time.Time
	LastUpdatedBy int64
	LastUpdatedOn time.Time
}
