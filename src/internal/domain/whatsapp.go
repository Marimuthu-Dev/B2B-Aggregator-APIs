package domain

const (
	WhatsAppTemplateTypeRegular = "regular"
	WhatsAppTemplateTypeMMLite  = "mm_lite"
)

// OutboxWhatsApp is a pending or claimed row from {DB_SCHEMA}.tbl_WhatsApp used for sending.
type OutboxWhatsApp struct {
	WhatsAppID   int64
	ClientID     int64
	FromMobile   string
	ToMobile     string
	WhatsAppText string
	TemplateID   int64
	TemplateName string
	TemplateType string
	CreatedBy    int64
}

// QueuedWhatsApp is a new row to insert into {DB_SCHEMA}.tbl_WhatsApp (IsSent = 0).
type QueuedWhatsApp struct {
	ClientID     int64
	FromMobile   string
	ToMobile     string
	WhatsAppText string
	TemplateID   int64
	TemplateName string
	CreatedBy    int64
}

// WhatsAppTemplate represents a WhatsApp template from {DB_SCHEMA}.tbl_WhatsAppTemplates.
type WhatsAppTemplate struct {
	TemplateID      int64
	TemplateName    string
	TemplateType    string
	VariableCount   int64
	Description     string
	IsActive        bool
	ApprovedByCPaaS bool
	CreatedBy       int64
	CreatedOn       string
	LastUpdatedBy   int64
	LastUpdatedOn   string
}

// IsMMLite returns true when this template is configured as the MM Lite marketing variant.
func (t WhatsAppTemplate) IsMMLite() bool {
	return t.TemplateType == WhatsAppTemplateTypeMMLite
}

// ResolveTemplateType returns the effective template type for a message,
// preferring the row-level value and falling back to the template entity.
func ResolveTemplateType(msg OutboxWhatsApp, tpl *WhatsAppTemplate) string {
	if msg.TemplateType != "" {
		return msg.TemplateType
	}
	if tpl != nil && tpl.TemplateType != "" {
		return tpl.TemplateType
	}
	return WhatsAppTemplateTypeRegular
}
