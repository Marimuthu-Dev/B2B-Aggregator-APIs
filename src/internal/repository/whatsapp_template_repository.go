package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"b2b-diagnostic-aggregator/apis/internal/domain"

	_ "github.com/microsoft/go-mssqldb"
)

// WhatsAppTemplateRepository handles SQL Server access for {DB_SCHEMA}.tbl_WhatsAppTemplates.
type WhatsAppTemplateRepository struct {
	db *sql.DB
}

// NewWhatsAppTemplateRepository uses a shared *sql.DB (same pool as config.ConnectDatabase / other workers).
func NewWhatsAppTemplateRepository(ctx context.Context, db *sql.DB) (*WhatsAppTemplateRepository, error) {
	if db == nil {
		return nil, fmt.Errorf("db is nil")
	}
	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("sql ping: %w", err)
	}
	return &WhatsAppTemplateRepository{db: db}, nil
}

// Close is a no-op; the underlying *sql.DB is owned by the caller.
func (r *WhatsAppTemplateRepository) Close() error {
	return nil
}

// NewWhatsAppTemplateRepositoryFromSQL uses an existing pool (API process). Does not ping.
func NewWhatsAppTemplateRepositoryFromSQL(db *sql.DB) *WhatsAppTemplateRepository {
	if db == nil {
		return nil
	}
	return &WhatsAppTemplateRepository{db: db}
}

const (
	templateNameMax    = 100
	templateTypeMax    = 20
	templateDescMax    = 500
)

// FindByName finds a template by its name.
func (r *WhatsAppTemplateRepository) FindByName(ctx context.Context, templateName string) (*domain.WhatsAppTemplate, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("whatsapp template repository is not configured")
	}
	
	templateName = clipRunes(strings.TrimSpace(templateName), templateNameMax)
	if templateName == "" {
		return nil, fmt.Errorf("TemplateName is required")
	}

	q := fmt.Sprintf(`
SELECT 
  TemplateID,
  TemplateName,
  TemplateType,
  VariableCount,
  Description,
  IsActive,
  ApprovedByCPaaS,
  CreatedBy,
  CreatedOn,
  LastUpdatedBy,
  LastUpdatedOn
FROM %s
WHERE TemplateName = @templateName AND IsActive = 1`, whatsappTemplatesTable())

	var t domain.WhatsAppTemplate
	var approvedByCPaaS sql.NullBool
	
	var variableCount sql.NullInt64
	err := r.db.QueryRowContext(ctx, q, sql.Named("templateName", templateName)).Scan(
		&t.TemplateID,
		&t.TemplateName,
		&t.TemplateType,
		&variableCount,
		&t.Description,
		&t.IsActive,
		&approvedByCPaaS,
		&t.CreatedBy,
		&t.CreatedOn,
		&t.LastUpdatedBy,
		&t.LastUpdatedOn,
	)
	if variableCount.Valid {
		t.VariableCount = variableCount.Int64
	}
	
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("template not found: %s", templateName)
	}
	if err != nil {
		return nil, fmt.Errorf("find template by name: %w", err)
	}
	
	if approvedByCPaaS.Valid {
		t.ApprovedByCPaaS = approvedByCPaaS.Bool
	}
	
	return &t, nil
}

// FindByID finds a template by its ID.
func (r *WhatsAppTemplateRepository) FindByID(ctx context.Context, templateID int64) (*domain.WhatsAppTemplate, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("whatsapp template repository is not configured")
	}

	q := fmt.Sprintf(`
SELECT 
  TemplateID,
  TemplateName,
  TemplateType,
  VariableCount,
  Description,
  IsActive,
  ApprovedByCPaaS,
  CreatedBy,
  CreatedOn,
  LastUpdatedBy,
  LastUpdatedOn
FROM %s
WHERE TemplateID = @templateID`, whatsappTemplatesTable())

	var t domain.WhatsAppTemplate
	var approvedByCPaaS sql.NullBool
	
	err := r.db.QueryRowContext(ctx, q, sql.Named("templateID", templateID)).Scan(
		&t.TemplateID,
		&t.TemplateName,
		&t.TemplateType,
		&t.VariableCount,
		&t.Description,
		&t.IsActive,
		&approvedByCPaaS,
		&t.CreatedBy,
		&t.CreatedOn,
		&t.LastUpdatedBy,
		&t.LastUpdatedOn,
	)
	
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("template not found with ID: %d", templateID)
	}
	if err != nil {
		return nil, fmt.Errorf("find template by ID: %w", err)
	}
	
	if approvedByCPaaS.Valid {
		t.ApprovedByCPaaS = approvedByCPaaS.Bool
	}
	
	return &t, nil
}

// FindAllActive finds all active templates.
func (r *WhatsAppTemplateRepository) FindAllActive(ctx context.Context) ([]domain.WhatsAppTemplate, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("whatsapp template repository is not configured")
	}

	q := fmt.Sprintf(`
SELECT 
  TemplateID,
  TemplateName,
  TemplateType,
  VariableCount,
  Description,
  IsActive,
  ApprovedByCPaaS,
  CreatedBy,
  CreatedOn,
  LastUpdatedBy,
  LastUpdatedOn
FROM %s
WHERE IsActive = 1
ORDER BY TemplateName ASC`, whatsappTemplatesTable())

	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("find all active templates: %w", err)
	}
	defer rows.Close()

	var templates []domain.WhatsAppTemplate
	for rows.Next() {
		var t domain.WhatsAppTemplate
		var approvedByCPaaS sql.NullBool
		
		err := rows.Scan(
			&t.TemplateID,
			&t.TemplateName,
			&t.TemplateType,
			&t.VariableCount,
			&t.Description,
			&t.IsActive,
			&approvedByCPaaS,
			&t.CreatedBy,
			&t.CreatedOn,
			&t.LastUpdatedBy,
			&t.LastUpdatedOn,
		)
		if err != nil {
			return nil, fmt.Errorf("scan template row: %w", err)
		}
		
		if approvedByCPaaS.Valid {
			t.ApprovedByCPaaS = approvedByCPaaS.Bool
		}
		
		templates = append(templates, t)
	}
	
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("template rows: %w", err)
	}
	
	return templates, nil
}

// Create inserts a new template.
func (r *WhatsAppTemplateRepository) Create(ctx context.Context, t domain.WhatsAppTemplate) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("whatsapp template repository is not configured")
	}
	
	templateName := clipRunes(strings.TrimSpace(t.TemplateName), templateNameMax)
	templateType := clipRunes(strings.TrimSpace(t.TemplateType), templateTypeMax)
	description := clipRunes(strings.TrimSpace(t.Description), templateDescMax)
	
	if templateName == "" {
		return fmt.Errorf("TemplateName is required")
	}
	if templateType == "" {
		templateType = "regular" // default
	}

	q := fmt.Sprintf(`
INSERT INTO %s (
  TemplateName,
  TemplateType,
  VariableCount,
  Description,
  IsActive,
  ApprovedByCPaaS,
  CreatedBy,
  CreatedOn,
  LastUpdatedBy,
  LastUpdatedOn
) VALUES (
  @templateName,
  @templateType,
  @variableCount,
  @description,
  @isActive,
  @approvedByCPaaS,
  @createdBy,
  GETDATE(),
  @lastUpdatedBy,
  GETDATE()
)`, whatsappTemplatesTable())

	_, err := r.db.ExecContext(ctx, q,
		sql.Named("templateName", templateName),
		sql.Named("templateType", templateType),
		sql.Named("variableCount", nullableInt64(t.VariableCount)),
		sql.Named("description", nullableString(description)),
		sql.Named("isActive", t.IsActive),
		sql.Named("approvedByCPaaS", nullableBool(t.ApprovedByCPaaS)),
		sql.Named("createdBy", t.CreatedBy),
		sql.Named("lastUpdatedBy", t.LastUpdatedBy),
	)
	if err != nil {
		return fmt.Errorf("create template: %w", err)
	}
	return nil
}

func nullableString(s string) any {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return s
}

func nullableBool(b bool) any {
	return b
}
