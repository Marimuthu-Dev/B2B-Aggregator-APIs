package domain

import "b2b-diagnostic-aggregator/apis/internal/timeutil"

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
	LeadStatusID            int8
	IsFit                   bool
	IsFitCertifiedGenerated bool
	ReportURL               string `json:"reportUrl,omitempty"`
	CreatedBy     int64
	CreatedOn     timeutil.ISTTime
	LastUpdatedBy int64
	LastUpdatedOn timeutil.ISTTime
}

// LeadDetail is lead with resolved ClientName and PackageName for API response.
type LeadDetail struct {
	Lead
	ClientName  string `json:"clientName,omitempty"`
	PackageName string `json:"packageName,omitempty"`
}

type LeadHistory struct {
	UID       int64
	LeadID    int64
	Action    string
	CreatedBy int64
	CreatedOn timeutil.ISTTime
}

const (
	LeadActionCreate       = "CREATE"
	LeadActionUpdate       = "UPDATE"
	LeadActionDelete       = "DELETE"
	LeadActionStatusUpdate = "STATUS_UPDATE"
	LeadActionCsvImport    = "CSV_IMPORT"
	// LeadActionReportUploaded matches tbl_LeadsHistory.Action and tbl_LeadStatusMaster.LeadStatusName lookup value.
	LeadActionReportUploaded = "Report Uploaded"
	// LeadActionReportReadyToDownload is logged when a lead can directly move to downloadable state without certificate generation.
	LeadActionReportReadyToDownload = "Report Ready to Download"
	// LeadActionFitCertificationGenerated is logged when the fitness worker merges the certificate PDF with the report.
	LeadActionFitCertificationGenerated = "Fit Certification Generated"
)
