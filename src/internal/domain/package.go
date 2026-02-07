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

// PackageWithTestsDetail is one item for GET /packages/with-tests-details response.
type PackageWithTestsDetail struct {
	PackageDetails Package `json:"packageDetails"`
	TestIDs       []int          `json:"testIds"`
	TestCount     int            `json:"testCount"`
	TestDetails   []TestInPackage `json:"testDetails"`
}

type TestInPackage struct {
	TestID   int    `json:"TestID"`
	TestName string `json:"TestName"`
	Category string `json:"Category"`
	IsActive bool   `json:"IsActive"`
}

// PackageClientMappingView is package-client mapping with names for list response.
type PackageClientMappingView struct {
	PackageClientID int       `json:"PackageClientID"`
	PackageID       int       `json:"PackageID"`
	ClientID        int64     `json:"ClientID"`
	Price           float64   `json:"Price"`
	IsActive        bool      `json:"IsActive"`
	CreatedBy       int64     `json:"CreatedBy"`
	CreatedOn       time.Time `json:"CreatedOn"`
	LastUpdatedBy   int64     `json:"LastUpdatedBy"`
	LastUpdatedOn   time.Time `json:"LastUpdatedOn"`
	PackageName     string    `json:"PackageName,omitempty"`
	ClientName      string    `json:"ClientName,omitempty"`
}

// PackageLabMappingView is package-lab mapping with names for list response.
type PackageLabMappingView struct {
	PackageLabID  int       `json:"PackageLabID"`
	PackageID     int       `json:"PackageID"`
	LabID         int64     `json:"LabID"`
	Price         float64   `json:"Price"`
	IsActive      bool      `json:"IsActive"`
	CreatedBy     int64     `json:"CreatedBy"`
	CreatedOn     time.Time `json:"CreatedOn"`
	LastUpdatedBy int64     `json:"LastUpdatedBy"`
	LastUpdatedOn time.Time `json:"LastUpdatedOn"`
	PackageName   string    `json:"PackageName,omitempty"`
	LabName       string    `json:"LabName,omitempty"`
}
