package domain

// OutboxWhatsApp is a pending or claimed row from {DB_SCHEMA}.tbl_WhatsApp used for sending.
type OutboxWhatsApp struct {
	WhatsAppID  int64
	ClientID    int64
	FromMobile  string
	ToMobile    string
	WhatsAppText string
	CreatedBy   int64
}

// QueuedWhatsApp is a new row to insert into {DB_SCHEMA}.tbl_WhatsApp (IsSent = 0).
type QueuedWhatsApp struct {
	ClientID     int64
	FromMobile   string
	ToMobile     string
	WhatsAppText string
	CreatedBy    int64
}
