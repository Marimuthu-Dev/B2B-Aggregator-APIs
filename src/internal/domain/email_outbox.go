package domain

// OutboxEmail is a pending or claimed row from {DB_SCHEMA}.tbl_Emails used for sending.
type OutboxEmail struct {
	EmailID     int64
	Subject     string
	FromAddress string
	ToAddress   string
	CC          string
	BCC         string
	BodyContent string
}

// QueuedEmail is a new row to insert into {DB_SCHEMA}.tbl_Emails (IsSent = 0).
type QueuedEmail struct {
	Subject     string
	FromAddress string
	ToAddress   string
	CC          string
	BCC         string
	BodyContent string
	EmailType   string
	CreatedBy   int64
}
