package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"b2b-diagnostic-aggregator/apis/internal/apperrors"
	"b2b-diagnostic-aggregator/apis/internal/domain"
	"b2b-diagnostic-aggregator/apis/internal/service"
	"b2b-diagnostic-aggregator/apis/internal/timeutil"

	"github.com/gin-gonic/gin"
)

// NotificationMonitorHandler handles Ops monitoring requests for WhatsApp & Email queues.
type NotificationMonitorHandler struct {
	svc service.NotificationMonitorService
}

// NewNotificationMonitorHandler returns a new NotificationMonitorHandler.
func NewNotificationMonitorHandler(svc service.NotificationMonitorService) *NotificationMonitorHandler {
	return &NotificationMonitorHandler{svc: svc}
}

func parseChannel(c *gin.Context) (domain.NotificationChannel, error) {
	raw := strings.ToLower(strings.TrimSpace(c.Param("channel")))
	ch, ok := domain.ParseNotificationChannel(raw)
	if !ok {
		return "", apperrors.NewBadRequest("Invalid channel: must be 'whatsapp' or 'email'", nil)
	}
	return ch, nil
}

// GetSummary godoc: GET /api/v1/dashboard/notifications/summary
func (h *NotificationMonitorHandler) GetSummary(c *gin.Context) {
	if !requireOpsUser(c) {
		return
	}
	data, err := h.svc.GetSummary(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}
	respondData(c, http.StatusOK, data, "Notification summary fetched successfully", nil)
}

// GetBacklog godoc: GET /api/v1/dashboard/notifications/:channel/backlog
func (h *NotificationMonitorHandler) GetBacklog(c *gin.Context) {
	if !requireOpsUser(c) {
		return
	}
	ch, err := parseChannel(c)
	if err != nil {
		respondError(c, err)
		return
	}
	data, err := h.svc.GetBacklog(c.Request.Context(), ch)
	if err != nil {
		respondError(c, err)
		return
	}
	respondData(c, http.StatusOK, data, ch.Label()+" backlog fetched successfully", nil)
}

// GetDailyHistory godoc: GET /api/v1/dashboard/notifications/:channel/history/daily?days=7
func (h *NotificationMonitorHandler) GetDailyHistory(c *gin.Context) {
	if !requireOpsUser(c) {
		return
	}
	ch, err := parseChannel(c)
	if err != nil {
		respondError(c, err)
		return
	}
	days := 7
	if dStr := strings.TrimSpace(c.Query("days")); dStr != "" {
		if d, err := strconv.Atoi(dStr); err == nil && d > 0 && d <= 30 {
			days = d
		}
	}
	data, err := h.svc.GetDailyHistory(c.Request.Context(), ch, days)
	if err != nil {
		respondError(c, err)
		return
	}
	respondData(c, http.StatusOK, data, ch.Label()+" daily history fetched successfully", nil)
}

// GetHourlyHistory godoc: GET /api/v1/dashboard/notifications/:channel/history/hourly
func (h *NotificationMonitorHandler) GetHourlyHistory(c *gin.Context) {
	if !requireOpsUser(c) {
		return
	}
	ch, err := parseChannel(c)
	if err != nil {
		respondError(c, err)
		return
	}
	data, err := h.svc.GetHourlyHistory(c.Request.Context(), ch)
	if err != nil {
		respondError(c, err)
		return
	}
	respondData(c, http.StatusOK, data, ch.Label()+" hourly history fetched successfully", nil)
}

// GetMessages godoc: GET /api/v1/dashboard/notifications/:channel/messages
func (h *NotificationMonitorHandler) GetMessages(c *gin.Context) {
	if !requireOpsUser(c) {
		return
	}
	ch, err := parseChannel(c)
	if err != nil {
		respondError(c, err)
		return
	}

	filter := domain.NotificationListFilter{
		Status:   strings.ToLower(strings.TrimSpace(c.Query("status"))),
		Search:   strings.TrimSpace(c.Query("search")),
		Page:     1,
		PageSize: 25,
	}

	if pStr := strings.TrimSpace(c.Query("page")); pStr != "" {
		if p, err := strconv.Atoi(pStr); err == nil && p > 0 {
			filter.Page = p
		}
	}
	if psStr := strings.TrimSpace(c.Query("pageSize")); psStr != "" {
		if ps, err := strconv.Atoi(psStr); err == nil && ps > 0 && ps <= 100 {
			filter.PageSize = ps
		}
	}

	const layout = "2006-01-02"
	loc := timeutil.ISTLocation()

	if fromStr := strings.TrimSpace(c.Query("fromDate")); fromStr != "" {
		if _, err := time.ParseInLocation(layout, fromStr, loc); err != nil {
			respondError(c, apperrors.NewBadRequest("Invalid fromDate: format must be YYYY-MM-DD", err))
			return
		}
		filter.FromDate = fromStr
	}

	if toStr := strings.TrimSpace(c.Query("toDate")); toStr != "" {
		if _, err := time.ParseInLocation(layout, toStr, loc); err != nil {
			respondError(c, apperrors.NewBadRequest("Invalid toDate: format must be YYYY-MM-DD", err))
			return
		}
		filter.ToDate = toStr
	}

	data, err := h.svc.GetMessages(c.Request.Context(), ch, filter)
	if err != nil {
		respondError(c, err)
		return
	}
	respondData(c, http.StatusOK, data, ch.Label()+" messages fetched successfully", nil)
}
