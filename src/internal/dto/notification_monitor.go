package dto

// NotificationThresholdsDto exposes the dashboard thresholds so the UI and API agree.
type NotificationThresholdsDto struct {
	WarnMinutes     int `json:"WarnMinutes"`
	CriticalMinutes int `json:"CriticalMinutes"`
	StaleHours      int `json:"StaleHours"`
}

// NotificationChannelSummaryResponse is the KPI / status block of one queue.
type NotificationChannelSummaryResponse struct {
	Channel              string  `json:"Channel"`
	Label                string  `json:"Label"`
	Status               string  `json:"Status"` // "Healthy" | "Warning" | "Critical"
	Unsent               int64   `json:"Unsent"`
	Processing           int64   `json:"Processing"`
	StuckProcessing      int64   `json:"StuckProcessing"`
	Stale                int64   `json:"Stale"`
	OverWarn             int64   `json:"OverWarn"`
	OverCritical         int64   `json:"OverCritical"`
	OldestWaitSeconds    *int64  `json:"OldestWaitSeconds"`
	OldestWaitFormatted  string  `json:"OldestWaitFormatted,omitempty"`
	AvgWaitSeconds       *int64  `json:"AvgWaitSeconds"`
	AvgWaitFormatted     string  `json:"AvgWaitFormatted,omitempty"`
	LastSentOn           *string `json:"LastSentOn"`
	SecondsSinceLastSent *int64  `json:"SecondsSinceLastSent"`
	SinceLastSentDisplay string  `json:"SinceLastSentDisplay,omitempty"`
	LastQueuedOn         *string `json:"LastQueuedOn"`
	SentLastHour         int64   `json:"SentLastHour"`
	FailedToday          int64   `json:"FailedToday"`
}

// NotificationSummaryResponse aggregates both channels for the top-level Ops view.
type NotificationSummaryResponse struct {
	OverallStatus string                                `json:"OverallStatus"` // "Healthy" | "Warning" | "Critical"
	Thresholds    NotificationThresholdsDto             `json:"Thresholds"`
	Channels      []NotificationChannelSummaryResponse  `json:"Channels"`
	ServerTimeIST string                                `json:"ServerTimeIST"`
}

// NotificationRecordResponse is one row of the WhatsApp / Email monitoring grid.
type NotificationRecordResponse struct {
	ID                int64   `json:"ID"`
	Recipient         string  `json:"Recipient"`
	Title             string  `json:"Title"`
	Category          string  `json:"Category"`
	ClientID          *int64  `json:"ClientID"`
	Status            string  `json:"Status"` // "Sent" | "Pending" | "Processing" | "Failed"
	CreatedOn         *string `json:"CreatedOn"`
	SentOn            *string `json:"SentOn"`
	LastUpdatedOn     *string `json:"LastUpdatedOn"`
	WaitSeconds       int64   `json:"WaitSeconds"`
	WaitTimeFormatted string  `json:"WaitTimeFormatted"`
	WaitSeverity      string  `json:"WaitSeverity"` // "normal" | "warning" | "critical"
}

// NotificationPageResponse wraps paginated grid records.
type NotificationPageResponse struct {
	Items      []NotificationRecordResponse `json:"Items"`
	Total      int64                        `json:"Total"`
	Page       int                          `json:"Page"`
	PageSize   int                          `json:"PageSize"`
	TotalPages int                          `json:"TotalPages"`
}

// NotificationDailyPointResponse represents activity on one calendar day.
type NotificationDailyPointResponse struct {
	Date               string `json:"Date"` // YYYY-MM-DD
	Queued             int64  `json:"Queued"`
	Unsent             int64  `json:"Unsent"`
	Failed             int64  `json:"Failed"`
	Sent               int64  `json:"Sent"`
	SentLate           int64  `json:"SentLate"`
	AvgDeliverySeconds *int64 `json:"AvgDeliverySeconds"`
	MaxDeliverySeconds *int64 `json:"MaxDeliverySeconds"`
}

// NotificationHourlyPointResponse represents activity in one rolling hour.
type NotificationHourlyPointResponse struct {
	Hour   string `json:"Hour"` // HH:00
	Queued int64  `json:"Queued"`
	Sent   int64  `json:"Sent"`
}
