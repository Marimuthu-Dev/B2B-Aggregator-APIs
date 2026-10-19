package service

import (
	"context"
	"fmt"
	"math"
	"time"

	"b2b-diagnostic-aggregator/apis/internal/domain"
	"b2b-diagnostic-aggregator/apis/internal/dto"
	"b2b-diagnostic-aggregator/apis/internal/repository"
	"b2b-diagnostic-aggregator/apis/internal/timeutil"
)

// Default thresholds for the Ops monitor.
const (
	DefaultNotificationWarnMinutes     = 5
	DefaultNotificationCriticalMinutes = 10
	DefaultNotificationStaleHours      = 24
)

// NotificationMonitorService defines business logic for monitoring outbound notification queues.
type NotificationMonitorService interface {
	GetSummary(ctx context.Context) (*dto.NotificationSummaryResponse, error)
	GetBacklog(ctx context.Context, channel domain.NotificationChannel) (*dto.NotificationChannelSummaryResponse, error)
	GetDailyHistory(ctx context.Context, channel domain.NotificationChannel, days int) ([]dto.NotificationDailyPointResponse, error)
	GetHourlyHistory(ctx context.Context, channel domain.NotificationChannel) ([]dto.NotificationHourlyPointResponse, error)
	GetMessages(ctx context.Context, channel domain.NotificationChannel, filter domain.NotificationListFilter) (*dto.NotificationPageResponse, error)
}

type notificationMonitorService struct {
	repo       *repository.NotificationMonitorRepository
	thresholds domain.NotificationThresholds
}

// NewNotificationMonitorService instantiates the notification monitor service.
func NewNotificationMonitorService(repo *repository.NotificationMonitorRepository) NotificationMonitorService {
	return &notificationMonitorService{
		repo: repo,
		thresholds: domain.NotificationThresholds{
			WarnMinutes:     DefaultNotificationWarnMinutes,
			CriticalMinutes: DefaultNotificationCriticalMinutes,
			StaleHours:      DefaultNotificationStaleHours,
		},
	}
}

func formatDurationSeconds(secs int64) string {
	if secs < 0 {
		return "0s"
	}
	if secs < 60 {
		return fmt.Sprintf("%ds", secs)
	}
	mins := secs / 60
	remSecs := secs % 60
	if mins < 60 {
		if remSecs > 0 {
			return fmt.Sprintf("%dm %ds", mins, remSecs)
		}
		return fmt.Sprintf("%dm", mins)
	}
	hours := mins / 60
	remMins := mins % 60
	if remMins > 0 {
		return fmt.Sprintf("%dh %dm", hours, remMins)
	}
	return fmt.Sprintf("%dh", hours)
}

func formatTimeIST(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.In(timeutil.ISTLocation()).Format("2006-01-02 15:04:05")
	return &s
}

func computeHealthStatus(backlog *domain.NotificationBacklog) string {
	if backlog == nil {
		return "Healthy"
	}
	if backlog.OverCritical > 0 || backlog.StuckProcessing > 0 {
		return "Critical"
	}
	if backlog.OverWarn > 0 {
		return "Warning"
	}
	return "Healthy"
}

func (s *notificationMonitorService) toChannelSummary(channel domain.NotificationChannel, backlog *domain.NotificationBacklog) dto.NotificationChannelSummaryResponse {
	status := computeHealthStatus(backlog)

	res := dto.NotificationChannelSummaryResponse{
		Channel:         string(channel),
		Label:           channel.Label(),
		Status:          status,
		Unsent:          backlog.Unsent,
		Processing:      backlog.Processing,
		StuckProcessing: backlog.StuckProcessing,
		Stale:           backlog.Stale,
		OverWarn:        backlog.OverWarn,
		OverCritical:    backlog.OverCritical,
		SentLastHour:    backlog.SentLastHour,
		FailedToday:     backlog.FailedToday,
	}

	if backlog.OldestWaitSeconds != nil {
		res.OldestWaitSeconds = backlog.OldestWaitSeconds
		res.OldestWaitFormatted = formatDurationSeconds(*backlog.OldestWaitSeconds)
	}
	if backlog.AvgWaitSeconds != nil {
		res.AvgWaitSeconds = backlog.AvgWaitSeconds
		res.AvgWaitFormatted = formatDurationSeconds(*backlog.AvgWaitSeconds)
	}
	if backlog.LastSentOn != nil {
		res.LastSentOn = formatTimeIST(backlog.LastSentOn)
	}
	if backlog.SecondsSinceLastSent != nil {
		res.SecondsSinceLastSent = backlog.SecondsSinceLastSent
		res.SinceLastSentDisplay = formatDurationSeconds(*backlog.SecondsSinceLastSent) + " ago"
	}
	if backlog.LastQueuedOn != nil {
		res.LastQueuedOn = formatTimeIST(backlog.LastQueuedOn)
	}

	return res
}

func (s *notificationMonitorService) GetSummary(ctx context.Context) (*dto.NotificationSummaryResponse, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("notification monitor repository is unavailable")
	}

	channelsSummary := make([]dto.NotificationChannelSummaryResponse, 0, len(domain.AllNotificationChannels))
	overallStatus := "Healthy"

	nowIST := time.Now().In(timeutil.ISTLocation()).Format("2006-01-02 15:04:05 IST")

	for _, ch := range domain.AllNotificationChannels {
		backlog, err := s.repo.Backlog(ctx, ch, s.thresholds)
		if err != nil {
			return nil, err
		}
		chSummary := s.toChannelSummary(ch, backlog)
		channelsSummary = append(channelsSummary, chSummary)

		if chSummary.Status == "Critical" {
			overallStatus = "Critical"
		} else if chSummary.Status == "Warning" && overallStatus != "Critical" {
			overallStatus = "Warning"
		}
	}

	return &dto.NotificationSummaryResponse{
		OverallStatus: overallStatus,
		Thresholds: dto.NotificationThresholdsDto{
			WarnMinutes:     s.thresholds.WarnMinutes,
			CriticalMinutes: s.thresholds.CriticalMinutes,
			StaleHours:      s.thresholds.StaleHours,
		},
		Channels:      channelsSummary,
		ServerTimeIST: nowIST,
	}, nil
}

func (s *notificationMonitorService) GetBacklog(ctx context.Context, channel domain.NotificationChannel) (*dto.NotificationChannelSummaryResponse, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("notification monitor repository is unavailable")
	}

	backlog, err := s.repo.Backlog(ctx, channel, s.thresholds)
	if err != nil {
		return nil, err
	}
	res := s.toChannelSummary(channel, backlog)
	return &res, nil
}

func (s *notificationMonitorService) GetDailyHistory(ctx context.Context, channel domain.NotificationChannel, days int) ([]dto.NotificationDailyPointResponse, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("notification monitor repository is unavailable")
	}
	if days <= 0 {
		days = 7
	}
	if days > 30 {
		days = 30
	}

	points, err := s.repo.Daily(ctx, channel, s.thresholds, days)
	if err != nil {
		return nil, err
	}

	// Index existing points by YYYY-MM-DD
	pointMap := make(map[string]domain.NotificationDailyPoint, len(points))
	for _, p := range points {
		pointMap[p.Date.Format("2006-01-02")] = p
	}

	// Generate all days in sequence up to today (IST)
	out := make([]dto.NotificationDailyPointResponse, 0, days)
	now := time.Now().In(timeutil.ISTLocation())
	startDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, timeutil.ISTLocation()).AddDate(0, 0, 1-days)

	for i := 0; i < days; i++ {
		cur := startDate.AddDate(0, 0, i)
		dateStr := cur.Format("2006-01-02")
		if p, ok := pointMap[dateStr]; ok {
			var avgSec *int64
			if p.Sent > 0 && p.TotalDeliverySecs > 0 {
				avg := p.TotalDeliverySecs / p.Sent
				avgSec = &avg
			}
			out = append(out, dto.NotificationDailyPointResponse{
				Date:               dateStr,
				Queued:             p.Queued,
				Unsent:             p.Unsent,
				Failed:             p.Failed,
				Sent:               p.Sent,
				SentLate:           p.SentLate,
				AvgDeliverySeconds: avgSec,
				MaxDeliverySeconds: p.MaxDeliverySeconds,
			})
		} else {
			out = append(out, dto.NotificationDailyPointResponse{
				Date:   dateStr,
				Queued: 0,
				Unsent: 0,
				Failed: 0,
				Sent:   0,
			})
		}
	}

	return out, nil
}

func (s *notificationMonitorService) GetHourlyHistory(ctx context.Context, channel domain.NotificationChannel) ([]dto.NotificationHourlyPointResponse, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("notification monitor repository is unavailable")
	}

	points, err := s.repo.Hourly(ctx, channel)
	if err != nil {
		return nil, err
	}

	hourMap := make(map[string]domain.NotificationHourlyPoint, len(points))
	for _, p := range points {
		hourMap[p.Hour.Format("15:00")] = p
	}

	// Rolling 24 hours up to current hour (IST)
	out := make([]dto.NotificationHourlyPointResponse, 0, 24)
	now := time.Now().In(timeutil.ISTLocation())
	startHour := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, timeutil.ISTLocation()).Add(-23 * time.Hour)

	for i := 0; i < 24; i++ {
		h := startHour.Add(time.Duration(i) * time.Hour)
		hourStr := h.Format("15:00")
		if p, ok := hourMap[hourStr]; ok {
			out = append(out, dto.NotificationHourlyPointResponse{
				Hour:   hourStr,
				Queued: p.Queued,
				Sent:   p.Sent,
			})
		} else {
			out = append(out, dto.NotificationHourlyPointResponse{
				Hour:   hourStr,
				Queued: 0,
				Sent:   0,
			})
		}
	}

	return out, nil
}

func (s *notificationMonitorService) GetMessages(ctx context.Context, channel domain.NotificationChannel, filter domain.NotificationListFilter) (*dto.NotificationPageResponse, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("notification monitor repository is unavailable")
	}

	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 25
	}
	if filter.PageSize > 100 {
		filter.PageSize = 100
	}

	page, err := s.repo.List(ctx, channel, s.thresholds, filter)
	if err != nil {
		return nil, err
	}

	items := make([]dto.NotificationRecordResponse, 0, len(page.Items))
	warnSec := int64(s.thresholds.WarnMinutes * 60)
	critSec := int64(s.thresholds.CriticalMinutes * 60)

	for _, rec := range page.Items {
		severity := "normal"
		if rec.Status == domain.NotificationStatusFailed {
			severity = "critical"
		} else if rec.Status != domain.NotificationStatusSent {
			// Unsent message
			if rec.WaitSeconds >= critSec {
				severity = "critical"
			} else if rec.WaitSeconds >= warnSec {
				severity = "warning"
			}
		} else if rec.WaitSeconds >= warnSec {
			// Sent, but took > 5 mins
			severity = "warning"
		}

		items = append(items, dto.NotificationRecordResponse{
			ID:                rec.ID,
			Recipient:         rec.Recipient,
			Title:             rec.Title,
			Category:          rec.Category,
			ClientID:          rec.ClientID,
			Status:            rec.Status,
			CreatedOn:         formatTimeIST(rec.CreatedOn),
			SentOn:            formatTimeIST(rec.SentOn),
			LastUpdatedOn:     formatTimeIST(rec.LastUpdatedOn),
			WaitSeconds:       rec.WaitSeconds,
			WaitTimeFormatted: formatDurationSeconds(rec.WaitSeconds),
			WaitSeverity:      severity,
		})
	}

	totalPages := 0
	if page.Total > 0 {
		totalPages = int(math.Ceil(float64(page.Total) / float64(filter.PageSize)))
	}

	return &dto.NotificationPageResponse{
		Items:      items,
		Total:      page.Total,
		Page:       filter.Page,
		PageSize:   filter.PageSize,
		TotalPages: totalPages,
	}, nil
}
