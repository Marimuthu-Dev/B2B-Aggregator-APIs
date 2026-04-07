package dto

import (
	"b2b-diagnostic-aggregator/apis/internal/domain"
)

type PackageRequest struct {
	PackageID   int64   `binding:"omitempty"`
	PackageName string  `binding:"required"`
	Description string  `binding:"omitempty"`
	IsActive    *bool   `binding:"omitempty"`
}

type CreatePackageWithTestsRequest struct {
	PackageRequest
	TestIDs []int64 `json:"testIds" binding:"required"`
}

type PackageStatusUpdateRequest struct {
	IsActive bool `json:"IsActive" binding:"required"`
}

type PackageClientMappingRequest struct {
	PackageID int64 `json:"PackageID" binding:"required"`
	ClientID  int64 `json:"ClientID" binding:"required"`
	Price     int   `json:"Price" binding:"required,min=0,max=32767"`
	IsActive  *bool   `json:"IsActive"`
}

type PackageLabMappingRequest struct {
	PackageID int64 `json:"PackageID" binding:"required"`
	LabID     int64 `json:"LabID" binding:"required"`
	Price     int   `json:"Price" binding:"required,min=0,max=32767"`
	IsActive  *bool   `json:"IsActive"`
}

// PackageLabMappingListQuery is optional filters for GET /packages/lab-mapping.
// IsActive: omit to default to active-only (true); send true/false to override.
type PackageLabMappingListQuery struct {
	PackageID *int64 `form:"PackageID" binding:"omitempty,min=1"`
	LabID     *int64 `form:"LabID" binding:"omitempty,min=1"`
	IsActive  *bool  `form:"IsActive" binding:"omitempty"`
}

type PackageMappingStatusUpdateRequest struct {
	IsActive *bool `json:"IsActive" binding:"required"` // pointer so required allows false (validator treats value-type required as "non-zero")
}

func (r PackageRequest) ToDomain() domain.Package {
	isActive := true
	if r.IsActive != nil {
		isActive = *r.IsActive
	}
	return domain.Package{
		PackageID:   r.PackageID,
		PackageName: r.PackageName,
		Description: r.Description,
		IsActive:    isActive,
	}
}
