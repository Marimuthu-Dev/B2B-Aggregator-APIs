package domain

import "time"

type Lead struct {
	LeadID        int64
	ClientID      int64
	PatientID     string
	PatientName   string
	Age           int8
	Gender        string
	PackageID     int
	ContactNumber string
	Emailid       string
	Address       string
	CityID        int8
	StateID       int8
	Pincode       string
	LeadStatusID  int8
	CreatedBy     int64
	CreatedOn     time.Time
	LastUpdatedBy int64
	LastUpdatedOn time.Time
}

type LeadHistory struct {
	UID       int64
	LeadID    int64
	Action    string
	CreatedBy int64
	CreatedOn time.Time
}

const (
	LeadActionCreate       = "CREATE"
	LeadActionUpdate       = "UPDATE"
	LeadActionDelete       = "DELETE"
	LeadActionStatusUpdate = "STATUS_UPDATE"
)
