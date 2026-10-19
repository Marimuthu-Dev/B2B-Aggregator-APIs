package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"b2b-diagnostic-aggregator/apis/internal/domain"
	"b2b-diagnostic-aggregator/apis/internal/timeutil"
)

// NotificationMonitorRepository runs read-only monitoring queries over the outbound
// message queues ({DB_SCHEMA}.tbl_WhatsApp and {DB_SCHEMA}.tbl_Emails).
//
// Time handling: workers write CreatedOn / SentOn with the DB server's GETDATE(), whose
// time zone depends on the server (UTC on Azure SQL). Every "how long has it waited"
// calculation is therefore done in SQL against GETDATE(), and every timestamp returned is
// shifted to IST in SQL using the server's live offset — so results are correct whether the
// DB clock is UTC or IST, and independent of the API host's clock.
//
// Reads use NOLOCK: this is a monitoring view and must never block the workers' UPDLOCK claims.
type NotificationMonitorRepository struct {
	db *sql.DB
}

// NewNotificationMonitorRepository returns nil when db is nil (API started without a DB).
func NewNotificationMonitorRepository(db *sql.DB) *NotificationMonitorRepository {
	if db == nil {
		return nil
	}
	return &NotificationMonitorRepository{db: db}
}

// notificationChannelSpec holds the per-table SQL fragments. Every fragment is a compile-time
// constant (no user input) and uses the table alias "t".
type notificationChannelSpec struct {
	table      string
	idCol      string
	notSent    string // not yet delivered (includes permanent failures)
	unsent     string // awaiting delivery (excludes permanent failures)
	pending    string // awaiting delivery and not claimed by a worker
	processing string // claimed by a worker
	failed     string // permanent failure, will not be retried
	statusExpr string
	listSelect string // channel-specific grid columns: ID, Recipient, Title, Category, ClientID
	listFrom   string // FROM ... (with joins)
	searchExpr string // OR-ed LIKE predicates on @search / @searchExact
}

const (
	// WhatsApp worker protocol (see WhatsAppRepository):
	//   IsSent = 1                      -> sent
	//   IsSent <> 1, LastUpdatedBy = -1 -> claimed by a worker batch (processing)
	//   IsSent <> 1, LastUpdatedBy = -2 -> permanent failure (never retried)
	//   otherwise                       -> pending
	waNotSent    = `(t.IsSent = 0 OR t.IsSent IS NULL)`
	waUnsent     = `((t.IsSent = 0 OR t.IsSent IS NULL) AND t.LastUpdatedBy <> -2)`
	waPending    = `((t.IsSent = 0 OR t.IsSent IS NULL) AND t.LastUpdatedBy NOT IN (-1, -2))`
	waProcessing = `((t.IsSent = 0 OR t.IsSent IS NULL) AND t.LastUpdatedBy = -1)`
	waFailed     = `((t.IsSent = 0 OR t.IsSent IS NULL) AND t.LastUpdatedBy = -2)`
	waStatusExpr = `CASE WHEN t.IsSent = 1 THEN 'Sent'
		WHEN t.LastUpdatedBy = -2 THEN 'Failed'
		WHEN t.LastUpdatedBy = -1 THEN 'Processing'
		ELSE 'Pending' END`

	// Email worker protocol (see EmailOutboxRepository): IsSent = 1 sent, otherwise pending
	// (failures are reset to IsSent = 0 and retried on the next cycle).
	emNotSent    = `(t.IsSent = 0 OR t.IsSent IS NULL)`
	emStatusExpr = `CASE WHEN t.IsSent = 1 THEN 'Sent' ELSE 'Pending' END`
	sqlFalse     = `(1 = 0)`
)

func notificationSpec(ch domain.NotificationChannel) (notificationChannelSpec, error) {
	switch ch {
	case domain.NotificationChannelWhatsApp:
		table := whatsappTable()
		return notificationChannelSpec{
			table:      table,
			idCol:      "WhatsAppID",
			notSent:    waNotSent,
			unsent:     waUnsent,
			pending:    waPending,
			processing: waProcessing,
			failed:     waFailed,
			statusExpr: waStatusExpr,
			listSelect: `t.WhatsAppID AS ID,
  t.ToMobile AS Recipient,
  ISNULL(tp.TemplateName, '') AS Title,
  ISNULL(tp.TemplateType, '') AS Category,
  t.ClientID AS ClientID`,
			listFrom: table + ` t WITH (NOLOCK)
LEFT JOIN ` + whatsappTemplatesTable() + ` tp WITH (NOLOCK) ON tp.TemplateID = t.TemplateID`,
			searchExpr: `(t.ToMobile LIKE @search ESCAPE '\' OR t.FromMobile LIKE @search ESCAPE '\'
  OR tp.TemplateName LIKE @search ESCAPE '\' OR CAST(t.WhatsAppID AS varchar(20)) = @searchExact)`,
		}, nil
	case domain.NotificationChannelEmail:
		table := emailsTable()
		return notificationChannelSpec{
			table:      table,
			idCol:      "EmailID",
			notSent:    emNotSent,
			unsent:     emNotSent,
			pending:    emNotSent,
			processing: sqlFalse,
			failed:     sqlFalse,
			statusExpr: emStatusExpr,
			listSelect: `t.EmailID AS ID,
  t.ToAddress AS Recipient,
  ISNULL(t.Subject, '') AS Title,
  ISNULL(t.EmailType, '') AS Category,
  CAST(NULL AS bigint) AS ClientID`,
			listFrom: table + ` t WITH (NOLOCK)`,
			searchExpr: `(t.ToAddress LIKE @search ESCAPE '\' OR t.Subject LIKE @search ESCAPE '\'
  OR t.EmailType LIKE @search ESCAPE '\' OR CAST(t.EmailID AS varchar(20)) = @searchExact)`,
		}, nil
	default:
		return notificationChannelSpec{}, fmt.Errorf("unknown notification channel %q", ch)
	}
}

// monitorClockPreamble declares the shared clock variables:
//
//	@now        DB server "now" (same clock the workers use for CreatedOn/SentOn)
//	@shift      minutes to add to a DB timestamp to get IST (330 on a UTC server, 0 on an IST server);
//	            the server offset is rounded to 15 minutes to absorb the two clock reads
//	@todayStart start of the current IST day, expressed on the DB clock (sargable on CreatedOn/SentOn)
const monitorClockPreamble = `
SET NOCOUNT ON;
DECLARE @now datetime = GETDATE();
DECLARE @shift int = 330 - CAST(ROUND(DATEDIFF(SECOND, GETUTCDATE(), GETDATE()) / 900.0, 0) AS int) * 15;
DECLARE @todayStart datetime = DATEADD(MINUTE, -@shift, CAST(CAST(DATEADD(MINUTE, @shift, @now) AS date) AS datetime));
`

// sqlDateTimeAsIST re-labels a DB wall-clock value (already shifted to IST in SQL) as Asia/Kolkata.
// go-mssqldb returns datetime values with a UTC location but the stored wall-clock fields.
func sqlDateTimeAsIST(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), timeutil.ISTLocation())
}

func nullTimeAsIST(nt sql.NullTime) *time.Time {
	if !nt.Valid {
		return nil
	}
	v := sqlDateTimeAsIST(nt.Time)
	return &v
}

func nullInt64Ptr(n sql.NullInt64) *int64 {
	if !n.Valid {
		return nil
	}
	v := n.Int64
	return &v
}

func thresholdParams(th domain.NotificationThresholds) []any {
	return []any{
		sql.Named("warn", th.WarnMinutes),
		sql.Named("crit", th.CriticalMinutes),
		sql.Named("staleHours", th.StaleHours),
	}
}

// Backlog returns the live queue state for one channel.
func (r *NotificationMonitorRepository) Backlog(ctx context.Context, ch domain.NotificationChannel, th domain.NotificationThresholds) (*domain.NotificationBacklog, error) {
	spec, err := notificationSpec(ch)
	if err != nil {
		return nil, err
	}

	// "active" = unsent and newer than the stale cut-off; only active rows drive health.
	q := monitorClockPreamble + `
DECLARE @staleCut datetime = DATEADD(HOUR, -@staleHours, @now);
DECLARE @warnCut datetime = DATEADD(MINUTE, -@warn, @now);
DECLARE @critCut datetime = DATEADD(MINUTE, -@crit, @now);

SELECT
  b.Unsent, b.Processing, b.StuckProcessing, b.Stale, b.OverWarn, b.OverCritical,
  DATEADD(MINUTE, @shift, b.OldestActiveOn) AS OldestActiveOn,
  DATEDIFF(SECOND, b.OldestActiveOn, @now) AS OldestWaitSec,
  b.AvgWaitSec,
  DATEADD(MINUTE, @shift, a.LastSentOn) AS LastSentOn,
  DATEDIFF(SECOND, a.LastSentOn, @now) AS SinceLastSentSec,
  DATEADD(MINUTE, @shift, a.LastQueuedOn) AS LastQueuedOn,
  a.SentLastHour,
  a.FailedToday,
  DATEADD(MINUTE, @shift, @now) AS NowLocal
FROM (
  SELECT
    COUNT_BIG(CASE WHEN ` + spec.unsent + ` THEN 1 END) AS Unsent,
    COUNT_BIG(CASE WHEN ` + spec.processing + ` THEN 1 END) AS Processing,
    COUNT_BIG(CASE WHEN ` + spec.processing + ` AND t.LastUpdatedOn <= @critCut THEN 1 END) AS StuckProcessing,
    COUNT_BIG(CASE WHEN ` + spec.unsent + ` AND t.CreatedOn < @staleCut THEN 1 END) AS Stale,
    COUNT_BIG(CASE WHEN ` + spec.unsent + ` AND t.CreatedOn >= @staleCut AND t.CreatedOn <= @warnCut THEN 1 END) AS OverWarn,
    COUNT_BIG(CASE WHEN ` + spec.unsent + ` AND t.CreatedOn >= @staleCut AND t.CreatedOn <= @critCut THEN 1 END) AS OverCritical,
    MIN(CASE WHEN ` + spec.unsent + ` AND t.CreatedOn >= @staleCut THEN t.CreatedOn END) AS OldestActiveOn,
    AVG(CASE WHEN ` + spec.unsent + ` AND t.CreatedOn >= @staleCut THEN CAST(DATEDIFF(SECOND, t.CreatedOn, @now) AS bigint) END) AS AvgWaitSec
  FROM ` + spec.table + ` t WITH (NOLOCK)
  WHERE ` + spec.notSent + `
) b
CROSS JOIN (
  SELECT
    (SELECT MAX(t.SentOn) FROM ` + spec.table + ` t WITH (NOLOCK) WHERE t.IsSent = 1) AS LastSentOn,
    (SELECT TOP (1) t.CreatedOn FROM ` + spec.table + ` t WITH (NOLOCK) ORDER BY t.` + spec.idCol + ` DESC) AS LastQueuedOn,
    (SELECT COUNT_BIG(1) FROM ` + spec.table + ` t WITH (NOLOCK) WHERE t.IsSent = 1 AND t.SentOn >= DATEADD(HOUR, -1, @now)) AS SentLastHour,
    (SELECT COUNT_BIG(1) FROM ` + spec.table + ` t WITH (NOLOCK) WHERE ` + spec.failed + ` AND t.CreatedOn >= @todayStart) AS FailedToday
) a;`

	var (
		out                                   domain.NotificationBacklog
		oldestOn, lastSentOn, lastQueuedOn    sql.NullTime
		oldestWait, avgWait, sinceLastSent    sql.NullInt64
		nowLocal                              time.Time
	)
	err = r.db.QueryRowContext(ctx, q, thresholdParams(th)...).Scan(
		&out.Unsent, &out.Processing, &out.StuckProcessing, &out.Stale, &out.OverWarn, &out.OverCritical,
		&oldestOn, &oldestWait, &avgWait,
		&lastSentOn, &sinceLastSent, &lastQueuedOn,
		&out.SentLastHour, &out.FailedToday,
		&nowLocal,
	)
	if err != nil {
		return nil, fmt.Errorf("notification backlog (%s): %w", ch, err)
	}
	out.OldestUnsentOn = nullTimeAsIST(oldestOn)
	out.OldestWaitSeconds = nullInt64Ptr(oldestWait)
	out.AvgWaitSeconds = nullInt64Ptr(avgWait)
	out.LastSentOn = nullTimeAsIST(lastSentOn)
	out.SecondsSinceLastSent = nullInt64Ptr(sinceLastSent)
	out.LastQueuedOn = nullTimeAsIST(lastQueuedOn)
	out.Now = sqlDateTimeAsIST(nowLocal)
	return &out, nil
}

// Daily returns per-IST-day totals for the last `days` days (only days with activity are returned;
// the service fills gaps).
func (r *NotificationMonitorRepository) Daily(ctx context.Context, ch domain.NotificationChannel, th domain.NotificationThresholds, days int) ([]domain.NotificationDailyPoint, error) {
	spec, err := notificationSpec(ch)
	if err != nil {
		return nil, err
	}
	q := monitorClockPreamble + `
DECLARE @rangeStart datetime = DATEADD(DAY, 1 - @days, @todayStart);

SELECT
  x.Day,
  SUM(x.Queued) AS Queued,
  SUM(x.Unsent) AS Unsent,
  SUM(x.Failed) AS Failed,
  SUM(x.Sent) AS Sent,
  SUM(x.SentLate) AS SentLate,
  ISNULL(SUM(x.DeliverySec), 0) AS TotalDeliverySec,
  MAX(x.DeliverySec) AS MaxDeliverySec
FROM (
  SELECT
    CAST(DATEADD(MINUTE, @shift, t.CreatedOn) AS date) AS Day,
    CAST(1 AS bigint) AS Queued,
    CAST(CASE WHEN ` + spec.unsent + ` THEN 1 ELSE 0 END AS bigint) AS Unsent,
    CAST(CASE WHEN ` + spec.failed + ` THEN 1 ELSE 0 END AS bigint) AS Failed,
    CAST(0 AS bigint) AS Sent,
    CAST(0 AS bigint) AS SentLate,
    CAST(NULL AS bigint) AS DeliverySec
  FROM ` + spec.table + ` t WITH (NOLOCK)
  WHERE t.CreatedOn >= @rangeStart
  UNION ALL
  SELECT
    CAST(DATEADD(MINUTE, @shift, t.SentOn) AS date),
    0, 0, 0, 1,
    CAST(CASE WHEN DATEDIFF(SECOND, t.CreatedOn, t.SentOn) >= @warn * 60 THEN 1 ELSE 0 END AS bigint),
    CAST(DATEDIFF(SECOND, t.CreatedOn, t.SentOn) AS bigint)
  FROM ` + spec.table + ` t WITH (NOLOCK)
  WHERE t.IsSent = 1 AND t.SentOn >= @rangeStart
) x
GROUP BY x.Day
ORDER BY x.Day;`

	args := append(thresholdParams(th), sql.Named("days", days))
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("notification daily (%s): %w", ch, err)
	}
	defer rows.Close()

	var out []domain.NotificationDailyPoint
	for rows.Next() {
		var p domain.NotificationDailyPoint
		var maxDelivery sql.NullInt64
		if err := rows.Scan(&p.Date, &p.Queued, &p.Unsent, &p.Failed, &p.Sent, &p.SentLate, &p.TotalDeliverySecs, &maxDelivery); err != nil {
			return nil, fmt.Errorf("notification daily scan (%s): %w", ch, err)
		}
		p.MaxDeliverySeconds = nullInt64Ptr(maxDelivery)
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("notification daily rows (%s): %w", ch, err)
	}
	return out, nil
}

// Hourly returns per-IST-hour queued/sent counts for the rolling last 24 hours
// (only hours with activity are returned; the service fills gaps).
func (r *NotificationMonitorRepository) Hourly(ctx context.Context, ch domain.NotificationChannel) ([]domain.NotificationHourlyPoint, error) {
	spec, err := notificationSpec(ch)
	if err != nil {
		return nil, err
	}
	// Bucket on the IST-shifted value so hours align to IST (+05:30) rather than the DB clock.
	q := monitorClockPreamble + `
DECLARE @hourStartLocal datetime = DATEADD(HOUR, DATEDIFF(HOUR, 0, DATEADD(MINUTE, @shift, @now)) - 23, 0);
DECLARE @hourStart datetime = DATEADD(MINUTE, -@shift, @hourStartLocal);

SELECT x.Bucket, SUM(x.Queued) AS Queued, SUM(x.Sent) AS Sent
FROM (
  SELECT DATEADD(HOUR, DATEDIFF(HOUR, 0, DATEADD(MINUTE, @shift, t.CreatedOn)), 0) AS Bucket,
         CAST(1 AS bigint) AS Queued, CAST(0 AS bigint) AS Sent
  FROM ` + spec.table + ` t WITH (NOLOCK)
  WHERE t.CreatedOn >= @hourStart
  UNION ALL
  SELECT DATEADD(HOUR, DATEDIFF(HOUR, 0, DATEADD(MINUTE, @shift, t.SentOn)), 0), 0, 1
  FROM ` + spec.table + ` t WITH (NOLOCK)
  WHERE t.IsSent = 1 AND t.SentOn >= @hourStart
) x
GROUP BY x.Bucket
ORDER BY x.Bucket;`

	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("notification hourly (%s): %w", ch, err)
	}
	defer rows.Close()

	var out []domain.NotificationHourlyPoint
	for rows.Next() {
		var p domain.NotificationHourlyPoint
		var bucket time.Time
		if err := rows.Scan(&bucket, &p.Queued, &p.Sent); err != nil {
			return nil, fmt.Errorf("notification hourly scan (%s): %w", ch, err)
		}
		p.Hour = sqlDateTimeAsIST(bucket)
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("notification hourly rows (%s): %w", ch, err)
	}
	return out, nil
}

// escapeLike escapes LIKE wildcards so user search text is matched literally (ESCAPE '\').
func escapeLike(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`, `[`, `\[`)
	return r.Replace(s)
}

// List returns one page of messages, newest first (ORDER BY identity DESC uses the clustered PK).
// filter must already be validated by the caller (status allow-list, dates, paging bounds).
func (r *NotificationMonitorRepository) List(ctx context.Context, ch domain.NotificationChannel, th domain.NotificationThresholds, filter domain.NotificationListFilter) (*domain.NotificationPage, error) {
	spec, err := notificationSpec(ch)
	if err != nil {
		return nil, err
	}

	var where strings.Builder
	where.WriteString("WHERE 1 = 1")
	switch filter.Status {
	case "", domain.NotificationListStatusAll:
	case domain.NotificationListStatusUnsent:
		where.WriteString("\n  AND " + spec.unsent)
	case domain.NotificationListStatusPending:
		where.WriteString("\n  AND " + spec.pending)
	case domain.NotificationListStatusProcessing:
		where.WriteString("\n  AND " + spec.processing)
	case domain.NotificationListStatusSent:
		where.WriteString("\n  AND t.IsSent = 1")
	case domain.NotificationListStatusFailed:
		where.WriteString("\n  AND " + spec.failed)
	case domain.NotificationListStatusDelayed:
		where.WriteString("\n  AND " + spec.unsent + " AND t.CreatedOn <= DATEADD(MINUTE, -@warn, @now)")
	case domain.NotificationListStatusSentLate:
		where.WriteString("\n  AND t.IsSent = 1 AND DATEDIFF(SECOND, t.CreatedOn, t.SentOn) >= @warn * 60")
	default:
		return nil, fmt.Errorf("unsupported status filter %q", filter.Status)
	}
	// Dates are IST calendar days; CAST(... AS date) parses YYYY-MM-DD independent of SET DATEFORMAT.
	if filter.FromDate != "" {
		where.WriteString("\n  AND t.CreatedOn >= DATEADD(MINUTE, -@shift, CAST(CAST(@fromDate AS date) AS datetime))")
	}
	if filter.ToDate != "" {
		where.WriteString("\n  AND t.CreatedOn < DATEADD(MINUTE, -@shift, DATEADD(DAY, 1, CAST(CAST(@toDate AS date) AS datetime)))")
	}
	search := strings.TrimSpace(filter.Search)
	if search != "" {
		where.WriteString("\n  AND " + spec.searchExpr)
	}

	q := monitorClockPreamble + `
SELECT
  ` + spec.listSelect + `,
  ` + spec.statusExpr + ` AS Status,
  DATEADD(MINUTE, @shift, t.CreatedOn) AS CreatedOn,
  DATEADD(MINUTE, @shift, t.SentOn) AS SentOn,
  DATEADD(MINUTE, @shift, t.LastUpdatedOn) AS LastUpdatedOn,
  CAST(CASE WHEN t.IsSent = 1 THEN DATEDIFF(SECOND, t.CreatedOn, t.SentOn)
       WHEN ` + spec.failed + ` THEN DATEDIFF(SECOND, t.CreatedOn, t.LastUpdatedOn)
       ELSE DATEDIFF(SECOND, t.CreatedOn, @now) END AS bigint) AS WaitSec,
  COUNT_BIG(1) OVER () AS TotalCount,
  DATEADD(MINUTE, @shift, @now) AS NowLocal
FROM ` + spec.listFrom + `
` + where.String() + `
ORDER BY t.` + spec.idCol + ` DESC
OFFSET @offset ROWS FETCH NEXT @limit ROWS ONLY;`

	args := append(thresholdParams(th),
		sql.Named("offset", (filter.Page-1)*filter.PageSize),
		sql.Named("limit", filter.PageSize),
		sql.Named("fromDate", filter.FromDate),
		sql.Named("toDate", filter.ToDate),
		sql.Named("search", "%"+escapeLike(search)+"%"),
		sql.Named("searchExact", search),
	)
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("notification list (%s): %w", ch, err)
	}
	defer rows.Close()

	page := &domain.NotificationPage{Items: make([]domain.NotificationRecord, 0, filter.PageSize)}
	for rows.Next() {
		var (
			rec                        domain.NotificationRecord
			recipient, title, category sql.NullString
			clientID, waitSec          sql.NullInt64
			createdOn, sentOn, updOn   sql.NullTime
			total                      int64
			nowLocal                   time.Time
		)
		if err := rows.Scan(&rec.ID, &recipient, &title, &category, &clientID, &rec.Status,
			&createdOn, &sentOn, &updOn, &waitSec, &total, &nowLocal); err != nil {
			return nil, fmt.Errorf("notification list scan (%s): %w", ch, err)
		}
		rec.Recipient = strings.TrimSpace(recipient.String)
		rec.Title = strings.TrimSpace(title.String)
		rec.Category = strings.TrimSpace(category.String)
		rec.ClientID = nullInt64Ptr(clientID)
		rec.CreatedOn = nullTimeAsIST(createdOn)
		rec.SentOn = nullTimeAsIST(sentOn)
		rec.LastUpdatedOn = nullTimeAsIST(updOn)
		if waitSec.Valid && waitSec.Int64 > 0 {
			rec.WaitSeconds = waitSec.Int64
		}
		page.Total = total
		if page.Now == nil {
			n := sqlDateTimeAsIST(nowLocal)
			page.Now = &n
		}
		page.Items = append(page.Items, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("notification list rows (%s): %w", ch, err)
	}
	return page, nil
}

// itoa64 is a small helper for building cache keys / labels without fmt.
func itoa64(n int64) string { return strconv.FormatInt(n, 10) }
