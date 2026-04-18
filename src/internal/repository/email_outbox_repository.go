package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"b2b-diagnostic-aggregator/apis/internal/domain"

	_ "github.com/microsoft/go-mssqldb"
)

// EmailOutboxRepository handles SQL Server access for MediAdmin.tbl_Emails.
type EmailOutboxRepository struct {
	db *sql.DB
}

// NewEmailOutboxRepository uses a shared *sql.DB (same pool as config.ConnectDatabase / fitness-worker).
// The caller owns the pool and must close it; Close on this repository is a no-op.
func NewEmailOutboxRepository(ctx context.Context, db *sql.DB) (*EmailOutboxRepository, error) {
	if db == nil {
		return nil, fmt.Errorf("db is nil")
	}
	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("sql ping: %w", err)
	}
	return &EmailOutboxRepository{db: db}, nil
}

// Close is a no-op; the underlying *sql.DB is owned by the caller (e.g. GORM from ConnectDatabase).
func (r *EmailOutboxRepository) Close() error {
	return nil
}

// SelectPendingBatch reads up to batchSize rows where IsSent is 0 or NULL, ordered by CreatedOn.
// It does not modify rows; use MarkSent / MarkAfterFailure after send attempts.
func (r *EmailOutboxRepository) SelectPendingBatch(ctx context.Context, batchSize int) ([]domain.OutboxEmail, error) {
	if batchSize < 1 {
		batchSize = 1
	}

	var b strings.Builder
	b.WriteString(`
SELECT TOP (`)
	b.WriteString(fmt.Sprintf("%d", batchSize))
	b.WriteString(`)
  EmailID,
  Subject,
  FromAddress,
  ToAddress,
  CCAddress,
  BCCAddress,
  BodyContent
FROM MediAdmin.tbl_Emails WITH (ROWLOCK, READPAST)
WHERE IsSent = 0 OR IsSent IS NULL
ORDER BY CreatedOn ASC, EmailID ASC`)

	rows, err := r.db.QueryContext(ctx, b.String())
	if err != nil {
		return nil, fmt.Errorf("select pending batch: %w", err)
	}
	defer rows.Close()

	var out []domain.OutboxEmail
	for rows.Next() {
		var ccAddr, bccAddr sql.NullString
		var e domain.OutboxEmail
		err := rows.Scan(
			&e.EmailID,
			&e.Subject,
			&e.FromAddress,
			&e.ToAddress,
			&ccAddr,
			&bccAddr,
			&e.BodyContent,
		)
		if err != nil {
			return nil, fmt.Errorf("scan pending row: %w", err)
		}
		if ccAddr.Valid {
			e.CC = strings.TrimSpace(ccAddr.String)
		}
		if bccAddr.Valid {
			e.BCC = strings.TrimSpace(bccAddr.String)
		}
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("pending batch rows: %w", err)
	}
	return out, nil
}

// MarkSent sets success state for a row that is still pending (IsSent 0 or NULL).
func (r *EmailOutboxRepository) MarkSent(ctx context.Context, emailID int64) error {
	const q = `
UPDATE MediAdmin.tbl_Emails
SET IsSent = 1,
    SentOn = GETDATE(),
    LastUpdatedOn = GETDATE()
WHERE EmailID = @p1 AND (IsSent = 0 OR IsSent IS NULL)`
	res, err := r.db.ExecContext(ctx, q, sql.Named("p1", emailID))
	if err != nil {
		return fmt.Errorf("mark sent: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("mark sent: no row updated for EmailID=%d", emailID)
	}
	return nil
}

// MarkAfterFailure sets IsSent = 0 so the row is picked again on the next cycle (same as new failures).
func (r *EmailOutboxRepository) MarkAfterFailure(ctx context.Context, emailID int64) error {
	const q = `
UPDATE MediAdmin.tbl_Emails
SET IsSent = 0,
    LastUpdatedOn = GETDATE()
WHERE EmailID = @id`

	_, err := r.db.ExecContext(ctx, q, sql.Named("id", emailID))
	if err != nil {
		return fmt.Errorf("mark after failure: %w", err)
	}
	return nil
}
