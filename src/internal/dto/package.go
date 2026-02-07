package dto

import (
	"b2b-diagnostic-aggregator/apis/internal/domain"
	"time"
)

type PackageRequest struct {
	PackageID     int       `binding:"omitempty"`
	PackageName   string    `binding:"required"`
	Description   string    `binding:"omitempty"`
	IsActive      *bool     `binding:"omitempty"`
	CreatedBy     int64     `binding:"required"`
	CreatedOn     *time.Time `binding:"omitempty"`
	LastUpdatedBy int64     `binding:"required"`
	LastUpdatedOn *time.Time `binding:"omitempty"`
}

type CreatePackageWithTestsRequest struct {
	PackageRequest
	TestIDs []int `json:"testIds" binding:"required"`
}

type PackageStatusUpdateRequest struct {
	IsActive      bool  `json:"IsActive" binding:"required"`
	LastUpdatedBy int64 `json:"LastUpdatedBy" binding:"required"`
}

type PackageClientMappingRequest struct {
	PackageID       int     `json:"PackageID" binding:"required"`
	ClientID        int64   `json:"ClientID" binding:"required"`
	Price           float64 `json:"Price" binding:"required,min=0"`
	IsActive        *bool   `json:"IsActive"`
	CreatedBy       int64   `json:"CreatedBy" binding:"required"`
	LastUpdatedBy   int64   `json:"LastUpdatedBy" binding:"required"`
}

type PackageLabMappingRequest struct {
	PackageID       int     `json:"PackageID" binding:"required"`
	LabID           int64   `json:"LabID" binding:"required"`
	Price           float64 `json:"Price" binding:"required,min=0"`
	IsActive        *bool   `json:"IsActive"`
	CreatedBy       int64   `json:"CreatedBy" binding:"required"`
	LastUpdatedBy   int64   `json:"LastUpdatedBy" binding:"required"`
}

type PackageMappingStatusUpdateRequest struct {
	IsActive      bool  `json:"IsActive" binding:"required"`
	LastUpdatedBy int64 `json:"LastUpdatedBy" binding:"required"`
}

func (r PackageRequest) ToDomain() domain.Package {
	var createdOn time.Time
	if r.CreatedOn != nil {
		createdOn = *r.CreatedOn
	}
	var lastUpdatedOn time.Time
	if r.LastUpdatedOn != nil {
		lastUpdatedOn = *r.LastUpdatedOn
	}
	isActive := true
	if r.IsActive != nil {
		isActive = *r.IsActive
	}

	return domain.Package{
		PackageID:     r.PackageID,
		PackageName:   r.PackageName,
		Description:   r.Description,
		IsActive:      isActive,
		CreatedBy:     r.CreatedBy,
		CreatedOn:     createdOn,
		LastUpdatedBy: r.LastUpdatedBy,
		LastUpdatedOn: lastUpdatedOn,
	}
}
