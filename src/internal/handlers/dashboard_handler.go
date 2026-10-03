package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"b2b-diagnostic-aggregator/apis/internal/apperrors"
	"b2b-diagnostic-aggregator/apis/internal/domain"
	"b2b-diagnostic-aggregator/apis/internal/middleware"
	"b2b-diagnostic-aggregator/apis/internal/service"
	"b2b-diagnostic-aggregator/apis/internal/timeutil"
	"b2b-diagnostic-aggregator/apis/pkg/utils"

	"github.com/gin-gonic/gin"
)

type DashboardHandler struct {
	svc service.DashboardService
}

func NewDashboardHandler(svc service.DashboardService) *DashboardHandler {
	return &DashboardHandler{svc: svc}
}

// requireOpsUser restricts cross-client/cross-lab dashboards to Ops (UserType 1).
func requireOpsUser(c *gin.Context) bool {
	userType, ok := middleware.GetUserType(c)
	if !ok || userType != utils.UserTypeEmployee {
		respondError(c, apperrors.NewForbidden("This dashboard is available only for Ops users", nil))
		return false
	}
	return true
}

// GetLabTracking godoc: GET /api/v1/dashboard/lab-tracking
// Query (all optional): fromDate, toDate (YYYY-MM-DD, IST, applied to lead CreatedOn), cityId, labId, clientId, clientTypeId.
func (h *DashboardHandler) GetLabTracking(c *gin.Context) {
	if !requireOpsUser(c) {
		return
	}
	filter, err := parseLabTrackingFilter(c)
	if err != nil {
		respondError(c, err)
		return
	}
	data, err := h.svc.GetLabTrackingSummary(filter)
	if err != nil {
		respondError(c, err)
		return
	}
	respondData(c, http.StatusOK, data, "Lab tracking dashboard fetched successfully", nil)
}

// GetLabTrackingFilters godoc: GET /api/v1/dashboard/lab-tracking/filters
func (h *DashboardHandler) GetLabTrackingFilters(c *gin.Context) {
	if !requireOpsUser(c) {
		return
	}
	data, err := h.svc.GetLabTrackingFilterOptions()
	if err != nil {
		respondError(c, err)
		return
	}
	respondData(c, http.StatusOK, data, "Lab tracking filters fetched successfully", nil)
}

func parseLabTrackingFilter(c *gin.Context) (domain.LabTrackingFilter, error) {
	var f domain.LabTrackingFilter
	var err error

	loc := timeutil.ISTLocation()
	const layout = "2006-01-02"
	if s := strings.TrimSpace(c.Query("fromDate")); s != "" {
		d, e := time.ParseInLocation(layout, s, loc)
		if e != nil {
			return f, apperrors.NewBadRequest("Invalid fromDate: use YYYY-MM-DD", e)
		}
		start := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, loc)
		f.From = &start
	}
	if s := strings.TrimSpace(c.Query("toDate")); s != "" {
		d, e := time.ParseInLocation(layout, s, loc)
		if e != nil {
			return f, apperrors.NewBadRequest("Invalid toDate: use YYYY-MM-DD", e)
		}
		end := time.Date(d.Year(), d.Month(), d.Day(), 23, 59, 59, 999999999, loc)
		f.To = &end
	}
	if f.From != nil && f.To != nil && f.From.After(*f.To) {
		return f, apperrors.NewBadRequest("fromDate must be on or before toDate", nil)
	}

	if f.CityID, err = optionalPositiveInt64Query(c, "cityId"); err != nil {
		return f, err
	}
	if f.LabID, err = optionalPositiveInt64Query(c, "labId"); err != nil {
		return f, err
	}
	if f.ClientID, err = optionalPositiveInt64Query(c, "clientId"); err != nil {
		return f, err
	}
	if f.ClientTypeID, err = optionalPositiveInt64Query(c, "clientTypeId"); err != nil {
		return f, err
	}
	return f, nil
}

func optionalPositiveInt64Query(c *gin.Context, key string) (*int64, error) {
	s := strings.TrimSpace(c.Query(key))
	if s == "" {
		return nil, nil
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil || v <= 0 {
		return nil, apperrors.NewBadRequest("Invalid "+key+": must be a positive integer", err)
	}
	return &v, nil
}
