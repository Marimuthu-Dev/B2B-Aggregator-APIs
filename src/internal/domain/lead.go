package domain

import "b2b-diagnostic-aggregator/apis/internal/timeutil"

// Lead fitness / report approval tri-state (MediAdmin.tbl_Leads.IsFit, TINYINT).
// Values must fit SQL Server tinyint (0–255): 0 = on hold, 1 = fit, 2 = unfit.
const (
	LeadFitHold  int8 = 0
	LeadFitFit   int8 = 1
	LeadFitUnfit int8 = 2
)

// LeadStatusIDDefault is the initial status for new leads when the client omits LeadStatusID (JSON zero / empty CSV).
const LeadStatusIDDefault int8 = 1

// Lead workflow: 8 = report uploaded (eligible for POST /leads/{id}/reports/approve); 9 = report approved (set when IsFit = 1 on approve).
const (
	LeadStatusIDReportUploaded int8 = 8
	LeadStatusIDReportApproved int8 = 9
	// LeadStatusIDClientDownloadNoFitGate is the minimum LeadStatusID at which clients may obtain a report
	// download URL without IsFit / IsReportDownloadable checks (e.g. post certificate merge, typically 10).
	LeadStatusIDClientDownloadNoFitGate int8 = 10
)

// LeadStatusIDReportApproval is the LeadStatusID required on the lead row before approve may run (report uploaded).
const LeadStatusIDReportApproval = LeadStatusIDReportUploaded

type Lead struct {
	LeadID                  int64
	ClientID                int64
	PatientID               string
	PatientName             string
	Age                     int8
	Gender                  string
	PackageID               int
	ContactNumber           string
	Emailid                 string
	Address                 string
	CityID                  int8
	StateID                 int8
	Pincode                 string
	LeadStatusID            int8
	AppointmentAt           *timeutil.ISTTime `json:"appointmentAt,omitempty"`
	LabID                   *int64            `json:"labId,omitempty"`
	IsFit                   int8              `json:"isFit"`
	IsReportDownloadable    bool              `json:"isReportDownloadable"`
	ApprovalRemarks         string            `json:"approvalRemarks,omitempty"`
	FitUpdatedOn            *timeutil.ISTTime `json:"fitUpdatedOn,omitempty"`
	IsFitCertifiedGenerated bool
	ReportURL               string `json:"reportUrl,omitempty"`
	CreatedBy               int64
	CreatedOn               timeutil.ISTTime
	LastUpdatedBy           int64
	LastUpdatedOn           timeutil.ISTTime
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
	// LeadActionReportApproval is logged when a user sets fit/hold/unfit and download/remarks via POST /leads/{id}/reports/approve.
	LeadActionReportApproval = "Report Approval"
)
