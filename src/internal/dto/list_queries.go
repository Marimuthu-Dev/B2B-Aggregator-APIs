package dto

import (
	"fmt"
	"strings"
	"time"
)

// MouListQuery holds optional MOU filters shared by client and lab list endpoints.
type MouListQuery struct {
	MouStatus     string `form:"mouStatus" binding:"omitempty"`
	MouExpiryFrom string `form:"mouExpiryFrom" binding:"omitempty"`
	MouExpiryTo   string `form:"mouExpiryTo" binding:"omitempty"`
}

type ClientListQuery struct {
	PaginationQuery
	CityID   *int8 `form:"cityId" binding:"omitempty,min=1"`
	StateID  *int8 `form:"stateId" binding:"omitempty,min=1"`
	IsActive *bool `form:"isActive" binding:"omitempty"`
	MouListQuery
	Search string `form:"search" binding:"omitempty"`
}

// ParseMouListFilters parses mouStatus (pipe-separated) and validates coupled mouExpiryFrom/mouExpiryTo.
// Expiry bounds are plain calendar dates (YYYY-MM-DD): parsed with time.Parse, which uses UTC midnight
// for date-only strings — no location-specific interpretation.
// Returns expiry range only when both date params are present; otherwise nil range and no error.
func ParseMouListFilters(q MouListQuery) (statuses []string, expiryRange *MouExpiryRangeParsed, err error) {
	statuses, err = parseMouStatusPipeList(q.MouStatus)
	if err != nil {
		return nil, nil, err
	}
	fromStr := strings.TrimSpace(q.MouExpiryFrom)
	toStr := strings.TrimSpace(q.MouExpiryTo)
	if fromStr == "" && toStr == "" {
		return statuses, nil, nil
	}
	if fromStr == "" || toStr == "" {
		return nil, nil, fmt.Errorf("mouExpiryFrom and mouExpiryTo must both be set when filtering by MOU expiry date")
	}
	from, err := time.Parse("2006-01-02", fromStr)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid mouExpiryFrom: use YYYY-MM-DD")
	}
	to, err := time.Parse("2006-01-02", toStr)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid mouExpiryTo: use YYYY-MM-DD")
	}
	if from.After(to) {
		return nil, nil, fmt.Errorf("mouExpiryFrom must be on or before mouExpiryTo")
	}
	return statuses, &MouExpiryRangeParsed{From: from, To: to}, nil
}

// MouExpiryRangeParsed holds inclusive MOU expiry bounds as calendar dates (UTC midnight per bound).
type MouExpiryRangeParsed struct {
	From time.Time
	To   time.Time
}

func parseMouStatusPipeList(s string) ([]string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	parts := strings.Split(s, "|")
	seen := make(map[string]struct{})
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		canonical, ok := normalizeMouStatusToken(p)
		if !ok {
			return nil, fmt.Errorf("invalid mouStatus value %q: use active, expired, or expiringSoon", p)
		}
		if _, dup := seen[canonical]; dup {
			continue
		}
		seen[canonical] = struct{}{}
		out = append(out, canonical)
	}
	return out, nil
}

func normalizeMouStatusToken(p string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(p)) {
	case "active":
		return "active", true
	case "expired":
		return "expired", true
	case "expiringsoon":
		return "expiringSoon", true
	default:
		return "", false
	}
}

type LabListQuery struct {
	PaginationQuery
	CityID   *uint8 `form:"cityId" binding:"omitempty,min=1"`
	StateID  *uint8 `form:"stateId" binding:"omitempty,min=1"`
	IsActive *bool `form:"isActive" binding:"omitempty"`
	MouListQuery
	Search string `form:"search" binding:"omitempty"`
}

type LeadListQuery struct {
	PaginationQuery
	LeadID           *int64  `form:"leadId" binding:"omitempty,min=1"`
	ClientID         *int64  `form:"clientId" binding:"omitempty,min=1"`
	LabID            *int64  `form:"labId" binding:"omitempty,min=1"`
	StatusID         *int8   `form:"statusId" binding:"omitempty,min=1"`
	PackageID        *int    `form:"packageId" binding:"omitempty,min=1"`
	CollectionType   *string `form:"collectionType" binding:"omitempty"`
	// Search matches PatientName, ContactNumber, or EmailID (substring, LIKE).
	Search string `form:"search" binding:"omitempty"`
	// FitnessStatus filters by tbl_Leads.IsFit (Empty | Not Assessed | On Hold | Fit | UnFit); see domain.ParseLeadListFitnessFilter.
	FitnessStatus string `form:"fitnessStatus" binding:"omitempty"`
	// AppointmentAtFrom / AppointmentAtTo filter by l.AppointmentAt (IST calendar day, YYYY-MM-DD); either or both.
	AppointmentAtFrom string `form:"appointmentAtFrom" binding:"omitempty"`
	AppointmentAtTo   string `form:"appointmentAtTo" binding:"omitempty"`
}

type PackageListQuery struct {
	PaginationQuery
	IsActive *bool  `form:"isActive" binding:"omitempty"`
	Search   string `form:"search" binding:"omitempty"`
}
