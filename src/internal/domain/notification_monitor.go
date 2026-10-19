package domain

import "time"

// NotificationChannel identifies an outbound message queue monitored by Ops.
type NotificationChannel string

const (
	// NotificationChannelWhatsApp is {DB_SCHEMA}.tbl_WhatsApp (whatsapp-worker).
	NotificationChannelWhatsApp NotificationChannel = "whatsapp"
	// NotificationChannelEmail is {DB_SCHEMA}.tbl_Emails (email-worker).
	NotificationChannelEmail NotificationChannel = "email"
)

// AllNotificationChannels is the display order on the Ops notification monitor.
var AllNotificationChannels = []NotificationChannel{NotificationChannelWhatsApp, NotificationChannelEmail}

// ParseNotificationChannel validates a channel path/query value against the allow-list.
func ParseNotificationChannel(s string) (NotificationChannel, bool) {
	switch NotificationChannel(s) {
	case NotificationChannelWhatsApp:
		return NotificationChannelWhatsApp, true
	case NotificationChannelEmail:
		return NotificationChannelEmail, true
	default:
		return "", false
	}
}

// Label is the human-readable channel name.
func (c NotificationChannel) Label() string {
	switch c {
	case NotificationChannelWhatsApp:
		return "WhatsApp"
	case NotificationChannelEmail:
		return "Email"
	default:
		return string(c)
	}
}

// NotificationThresholds drive the "waiting too long" highlighting.
//
//	WarnMinutes      unsent longer than this -> Warning (amber)
//	CriticalMinutes  unsent longer than this -> Critical (red)
//	StaleHours       unsent rows older than this are reported separately as Stale and do not
//	                 drive health (e.g. an email with a bad address that the worker retries forever),
//	                 so one poisoned row cannot keep the dashboard permanently red.
type NotificationThresholds struct {
	WarnMinutes     int
	CriticalMinutes int
	StaleHours      int
}

// Message status values derived from IsSent / LastUpdatedBy (see repository for the exact rules).
const (
	NotificationStatusSent       = "Sent"
	NotificationStatusPending    = "Pending"
	NotificationStatusProcessing = "Processing" // WhatsApp only: claimed by a worker (LastUpdatedBy = -1)
	NotificationStatusFailed     = "Failed"     // WhatsApp only: permanent failure (LastUpdatedBy = -2)
)

// NotificationBacklog is the live queue state of one channel.
// All time values are IST wall-clock (converted in SQL from the DB server clock).
type NotificationBacklog struct {
	Unsent          int64 // awaiting delivery (pending + processing), excludes permanent failures
	Processing      int64 // claimed by a worker but not yet marked sent/failed
	StuckProcessing int64 // processing rows not touched for longer than CriticalMinutes
	Stale           int64 // unsent rows older than StaleHours (excluded from health)
	OverWarn        int64 // active unsent rows waiting >= WarnMinutes
	OverCritical    int64 // active unsent rows waiting >= CriticalMinutes

	OldestUnsentOn    *time.Time
	OldestWaitSeconds *int64
	AvgWaitSeconds    *int64

	LastSentOn           *time.Time
	SecondsSinceLastSent *int64
	LastQueuedOn         *time.Time
	SentLastHour         int64
	FailedToday          int64

	Now time.Time // DB "now" in IST
}

// NotificationDailyPoint is one IST calendar day. Queued/Unsent/Failed are bucketed by CreatedOn;
// Sent/delivery metrics are bucketed by SentOn (i.e. "actually delivered that day").
type NotificationDailyPoint struct {
	Date               time.Time
	Queued             int64
	Unsent             int64
	Failed             int64
	Sent               int64
	SentLate           int64 // delivered, but took >= WarnMinutes
	TotalDeliverySecs  int64
	MaxDeliverySeconds *int64
}

// NotificationHourlyPoint is one IST clock hour.
type NotificationHourlyPoint struct {
	Hour   time.Time
	Queued int64
	Sent   int64
}

// NotificationListFilter narrows the message grid. Status must be one of the NotificationListStatus* values.
type NotificationListFilter struct {
	Status   string
	FromDate string // YYYY-MM-DD (IST, on CreatedOn), already validated
	ToDate   string // YYYY-MM-DD (IST, on CreatedOn), already validated
	Search   string
	Page     int
	PageSize int
}

// Allowed grid status filters.
const (
	NotificationListStatusAll        = "all"
	NotificationListStatusUnsent     = "unsent"
	NotificationListStatusPending    = "pending"
	NotificationListStatusProcessing = "processing"
	NotificationListStatusSent       = "sent"
	NotificationListStatusFailed     = "failed"
	NotificationListStatusDelayed    = "delayed"  // unsent and waiting >= WarnMinutes
	NotificationListStatusSentLate   = "sentlate" // sent, but delivery took >= WarnMinutes
)

// NotificationRecord is one row of the WhatsApp / Email grid (no message body is ever loaded).
type NotificationRecord struct {
	ID            int64
	Recipient     string
	Title         string // WhatsApp: template name, Email: subject
	Category      string // WhatsApp: template type, Email: email type
	ClientID      *int64
	Status        string
	CreatedOn     *time.Time
	SentOn        *time.Time
	LastUpdatedOn *time.Time
	WaitSeconds   int64 // sent: delivery time; failed: time until failure; otherwise: time waiting so far
}

// NotificationPage is a page of grid rows.
type NotificationPage struct {
	Items []NotificationRecord
	Total int64
	Now   *time.Time
}
