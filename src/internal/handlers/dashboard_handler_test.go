package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"b2b-diagnostic-aggregator/apis/internal/timeutil"
	"github.com/gin-gonic/gin"
)

func TestParseLabTrackingFilter_DatesAndFilters(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("valid custom date range in IST", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/dashboard/lab-tracking?fromDate=2026-10-24&toDate=2026-10-25&cityId=2&labId=3&clientId=4&clientTypeId=1", nil)

		filter, err := parseLabTrackingFilter(c)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if filter.From == nil || filter.To == nil {
			t.Fatalf("expected From and To to be set, got From=%v, To=%v", filter.From, filter.To)
		}

		loc := timeutil.ISTLocation()
		expectedFrom := time.Date(2026, 10, 24, 0, 0, 0, 0, loc)
		expectedTo := time.Date(2026, 10, 25, 23, 59, 59, 999999999, loc)

		if !filter.From.Equal(expectedFrom) {
			t.Errorf("From = %v, want %v", filter.From, expectedFrom)
		}
		if !filter.To.Equal(expectedTo) {
			t.Errorf("To = %v, want %v", filter.To, expectedTo)
		}
		if filter.CityID == nil || *filter.CityID != 2 {
			t.Errorf("CityID = %v, want 2", filter.CityID)
		}
		if filter.LabID == nil || *filter.LabID != 3 {
			t.Errorf("LabID = %v, want 3", filter.LabID)
		}
		if filter.ClientID == nil || *filter.ClientID != 4 {
			t.Errorf("ClientID = %v, want 4", filter.ClientID)
		}
		if filter.ClientTypeID == nil || *filter.ClientTypeID != 1 {
			t.Errorf("ClientTypeID = %v, want 1", filter.ClientTypeID)
		}
	})

	t.Run("empty filters allow all", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/dashboard/lab-tracking", nil)

		filter, err := parseLabTrackingFilter(c)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if filter.From != nil || filter.To != nil {
			t.Errorf("expected From and To nil, got From=%v, To=%v", filter.From, filter.To)
		}
	})

	t.Run("invalid date format returns error", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/dashboard/lab-tracking?fromDate=24-10-2026", nil)

		_, err := parseLabTrackingFilter(c)
		if err == nil {
			t.Fatal("expected error for non YYYY-MM-DD date")
		}
	})

	t.Run("fromDate after toDate returns error", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/dashboard/lab-tracking?fromDate=2026-10-26&toDate=2026-10-25", nil)

		_, err := parseLabTrackingFilter(c)
		if err == nil {
			t.Fatal("expected error when fromDate > toDate")
		}
	})
}
