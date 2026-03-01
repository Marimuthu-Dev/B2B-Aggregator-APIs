package dto

import (
	"b2b-diagnostic-aggregator/apis/internal/domain"
)

type PackageRequest struct {
	PackageID   int     `binding:"omitempty"`
	PackageName string  `binding:"required"`
	Description string  `binding:"omitempty"`
	IsActive    *bool   `binding:"omitempty"`
}

type CreatePackageWithTestsRequest struct {
	PackageRequest
	TestIDs []int `json:"testIds" binding:"required"`
}

type PackageStatusUpdateRequest struct {
	IsActive bool `json:"IsActive" binding:"required"`
}

type PackageClientMappingRequest struct {
	PackageID int     `json:"PackageID" binding:"required"`
	ClientID  int64   `json:"ClientID" binding:"required"`
	Price     float64 `json:"Price" binding:"required,min=0"`
	IsActive  *bool   `json:"IsActive"`
}

type PackageLabMappingRequest struct {
	PackageID int     `json:"PackageID" binding:"required"`
	LabID     int64   `json:"LabID" binding:"required"`
	Price     float64 `json:"Price" binding:"required,min=0"`
	IsActive  *bool   `json:"IsActive"`
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
