package domain

// OutboxEmail is a pending or claimed row from MediAdmin.tbl_Emails used for sending.
type OutboxEmail struct {
	EmailID     int64
	Subject     string
	FromAddress string
	ToAddress   string
	CC          string
	BCC         string
	BodyContent string
}
