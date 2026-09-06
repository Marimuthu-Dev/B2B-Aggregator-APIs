package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"b2b-diagnostic-aggregator/apis/internal/domain"
	persistencemodels "b2b-diagnostic-aggregator/apis/internal/persistence/models"

	_ "github.com/microsoft/go-mssqldb"
)

// WhatsAppRepository handles SQL Server access for {DB_SCHEMA}.tbl_WhatsApp.
type WhatsAppRepository struct {
	db *sql.DB
}

// NewWhatsAppRepository uses a shared *sql.DB (same pool as config.ConnectDatabase / other workers).
// The caller owns the pool and must close it; Close on this repository is a no-op.
func NewWhatsAppRepository(ctx context.Context, db *sql.DB) (*WhatsAppRepository, error) {
	if db == nil {
		return nil, fmt.Errorf("db is nil")
	}
	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("sql ping: %w", err)
	}
	return &WhatsAppRepository{db: db}, nil
}

// Close is a no-op; the underlying *sql.DB is owned by the caller (e.g. GORM from ConnectDatabase).
func (r *WhatsAppRepository) Close() error {
	return nil
}

// NewWhatsAppRepositoryFromSQL uses an existing pool (API process). Does not ping.
func NewWhatsAppRepositoryFromSQL(db *sql.DB) *WhatsAppRepository {
	if db == nil {
		return nil
	}
	return &WhatsAppRepository{db: db}
}

const (
	whatsappMobileMax = 10
	whatsappTextMax   = 350
)

// Enqueue inserts a pending row (IsSent = 0) into {DB_SCHEMA}.tbl_WhatsApp for the WhatsApp worker.
func (r *WhatsAppRepository) Enqueue(ctx context.Context, w domain.QueuedWhatsApp) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("whatsapp repository is not configured")
	}
	fromMobile := clipRunes(strings.TrimSpace(w.FromMobile), whatsappMobileMax)
	toMobile := clipRunes(strings.TrimSpace(w.ToMobile), whatsappMobileMax)
	whatsappText := clipRunes(strings.TrimSpace(w.WhatsAppText), whatsappTextMax)
	templateName := clipRunes(strings.TrimSpace(w.TemplateName), templateNameMax)

	if fromMobile == "" {
		return fmt.Errorf("FromMobile is required")
	}
	if toMobile == "" {
		return fmt.Errorf("ToMobile is required")
	}
	if whatsappText == "" {
		return fmt.Errorf("WhatsAppText is required")
	}

	q := fmt.Sprintf(`
INSERT INTO %s (
  ClientID,
  FromMobile,
  ToMobile,
  WhatsAppText,
  TemplateID,
  TemplateName,
  CreatedBy,
  CreatedOn,
  IsSent,
  SentOn,
  LastUpdatedBy,
  LastUpdatedOn
) VALUES (
  @clientID,
  @fromMobile,
  @toMobile,
  @whatsappText,
  @templateID,
  @templateName,
  @createdBy,
  GETDATE(),
  0,
  NULL,
  @createdBy,
  GETDATE()
)`, whatsappTable())

	_, err := r.db.ExecContext(ctx, q,
		sql.Named("clientID", nullableInt64(w.ClientID)),
		sql.Named("fromMobile", fromMobile),
		sql.Named("toMobile", toMobile),
		sql.Named("whatsappText", whatsappText),
		sql.Named("templateID", nullableInt64(w.TemplateID)),
		sql.Named("templateName", nullableString(templateName)),
		sql.Named("createdBy", w.CreatedBy),
	)
	if err != nil {
		return fmt.Errorf("enqueue whatsapp: %w", err)
	}
	return nil
}

// EnqueueWithTemplate resolves the given template name / id from {DB_SCHEMA}.tbl_WhatsAppTemplates,
// then enqueues a pending WhatsApp message with both TemplateID and TemplateName populated.
// If the template is not found or inactive, an error is returned.
func (r *WhatsAppRepository) EnqueueWithTemplate(
	ctx context.Context,
	w domain.QueuedWhatsApp,
	templateRepo *WhatsAppTemplateRepository,
) error {
	if templateRepo == nil {
		return fmt.Errorf("whatsapp template repository is required for EnqueueWithTemplate")
	}

	templateName := strings.TrimSpace(w.TemplateName)
	if w.TemplateID == 0 && templateName != "" {
		tpl, err := templateRepo.FindByName(ctx, templateName)
		if err != nil {
			return fmt.Errorf("resolve template by name %q: %w", templateName, err)
		}
		if !tpl.IsActive {
			return fmt.Errorf("template is inactive: %s", templateName)
		}
		w.TemplateID = tpl.TemplateID
		w.TemplateName = tpl.TemplateName
	} else if w.TemplateID != 0 && templateName == "" {
		tpl, err := templateRepo.FindByID(ctx, w.TemplateID)
		if err != nil {
			return fmt.Errorf("resolve template by id %d: %w", w.TemplateID, err)
		}
		if !tpl.IsActive {
			return fmt.Errorf("template is inactive: %s", tpl.TemplateName)
		}
		w.TemplateName = tpl.TemplateName
	}

	return r.Enqueue(ctx, w)
}

func nullableInt64(n int64) any {
	if n == 0 {
		return nil
	}
	return n
}

// SelectPendingBatch reads up to batchSize rows where IsSent is 0 or NULL, ordered by CreatedOn.
// It does not modify rows; use MarkSent / MarkAfterFailure after send attempts.
func (r *WhatsAppRepository) SelectPendingBatch(ctx context.Context, batchSize int) ([]domain.OutboxWhatsApp, error) {
	if batchSize < 1 {
		batchSize = 1
	}

	var b strings.Builder
	b.WriteString(`
SELECT TOP (`)
	b.WriteString(fmt.Sprintf("%d", batchSize))
	b.WriteString(`)
  w.WhatsAppID,
  w.ClientID,
  w.FromMobile,
  w.ToMobile,
  w.WhatsAppText,
  w.TemplateID,
  COALESCE(NULLIF(t.TemplateName, ''), NULLIF(w.TemplateName, '')) AS TemplateName,
  t.TemplateType,
  w.CreatedBy
FROM `)
	b.WriteString(whatsappTable())
	b.WriteString(` w WITH (ROWLOCK, READPAST)
LEFT JOIN `)
	b.WriteString(whatsappTemplatesTable())
	b.WriteString(` t ON w.TemplateID = t.TemplateID
WHERE w.IsSent = 0 OR w.IsSent IS NULL
ORDER BY w.CreatedOn ASC, w.WhatsAppID ASC`)

	rows, err := r.db.QueryContext(ctx, b.String())
	if err != nil {
		return nil, fmt.Errorf("select pending batch: %w", err)
	}
	defer rows.Close()

	var out []domain.OutboxWhatsApp
	for rows.Next() {
		var clientID sql.NullInt64
		var templateID sql.NullInt64
		var templateName sql.NullString
		var templateType sql.NullString
		var w domain.OutboxWhatsApp
		err := rows.Scan(
			&w.WhatsAppID,
			&clientID,
			&w.FromMobile,
			&w.ToMobile,
			&w.WhatsAppText,
			&templateID,
			&templateName,
			&templateType,
			&w.CreatedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("scan pending row: %w", err)
		}
		if clientID.Valid {
			w.ClientID = clientID.Int64
		}
		if templateID.Valid {
			w.TemplateID = templateID.Int64
		}
		if templateName.Valid {
			w.TemplateName = templateName.String
		}
		if templateType.Valid {
			w.TemplateType = templateType.String
		}
		out = append(out, w)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("pending batch rows: %w", err)
	}
	return out, nil
}

// MarkSent sets success state for a row that is still pending (IsSent 0 or NULL).
func (r *WhatsAppRepository) MarkSent(ctx context.Context, whatsappID int64) error {
	q := fmt.Sprintf(`
UPDATE %s
SET IsSent = 1,
    SentOn = GETDATE(),
    LastUpdatedOn = GETDATE()
WHERE WhatsAppID = @p1 AND (IsSent = 0 OR IsSent IS NULL)`, whatsappTable())
	res, err := r.db.ExecContext(ctx, q, sql.Named("p1", whatsappID))
	if err != nil {
		return fmt.Errorf("mark sent: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("mark sent: no row updated for WhatsAppID=%d", whatsappID)
	}
	return nil
}

// MarkAfterFailure sets IsSent = 0 so the row is picked again on the next cycle (same as new failures).
func (r *WhatsAppRepository) MarkAfterFailure(ctx context.Context, whatsappID int64) error {
	q := fmt.Sprintf(`
UPDATE %s
SET IsSent = 0,
    LastUpdatedOn = GETDATE()
WHERE WhatsAppID = @id`, whatsappTable())

	_, err := r.db.ExecContext(ctx, q, sql.Named("id", whatsappID))
	if err != nil {
		return fmt.Errorf("mark after failure: %w", err)
	}
	return nil
}

func whatsappTable() string {
	return persistencemodels.Table("tbl_WhatsApp")
}

func whatsappTemplatesTable() string {
	return persistencemodels.Table("tbl_WhatsAppTemplates")
}
