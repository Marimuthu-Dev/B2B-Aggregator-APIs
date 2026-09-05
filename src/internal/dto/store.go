package dto

import "b2b-diagnostic-aggregator/apis/internal/domain"

type StoreRequest struct {
	ClientID      int64  `json:"ClientID" binding:"required"`
	StoreName     string `json:"StoreName" binding:"required"`
	Address       string `json:"Address" binding:"required"`
	CityID        int16  `json:"CityID" binding:"required"`
	StateID       int16  `json:"StateID" binding:"required"`
	Pincode       string `json:"Pincode" binding:"required"`
	ContactNumber string `json:"ContactNumber" binding:"required"`
	EmailID       string `json:"EmailID" binding:"required"`
	IsActive      *bool  `json:"IsActive" binding:"omitempty"`
	Password      string `json:"Password" binding:"omitempty"`
}

type StoreUpdateRequest struct {
	StoreName     *string `json:"StoreName"`
	Address       *string `json:"Address"`
	CityID        *int16  `json:"CityID"`
	StateID       *int16  `json:"StateID"`
	Pincode       *string `json:"Pincode"`
	ContactNumber *string `json:"ContactNumber"`
	EmailID       *string `json:"EmailID"`
	IsActive      *bool   `json:"IsActive"`
}

func (r StoreUpdateRequest) HasAtLeastOneField() bool {
	return r.StoreName != nil || r.Address != nil || r.CityID != nil || r.StateID != nil || r.Pincode != nil ||
		r.ContactNumber != nil || r.EmailID != nil || r.IsActive != nil
}

func (r StoreRequest) ToDomain() domain.Store {
	isActive := true
	if r.IsActive != nil {
		isActive = *r.IsActive
	}
	return domain.Store{
		ClientID:      r.ClientID,
		StoreName:     r.StoreName,
		Address:       r.Address,
		CityID:        r.CityID,
		StateID:       r.StateID,
		Pincode:       r.Pincode,
		ContactNumber: r.ContactNumber,
		EmailID:       r.EmailID,
		IsActive:      isActive,
	}
}

type StoreListQuery struct {
	PaginationQuery
	ClientID *int64 `form:"clientId" binding:"omitempty,min=1"`
	IsActive *bool  `form:"isActive" binding:"omitempty"`
	Search   string `form:"search" binding:"omitempty"`
}
