package domain

import (
	"fmt"
	"strconv"
	"strings"

	"b2b-diagnostic-aggregator/apis/internal/timeutil"
)

// Lead collection site: MediAdmin.tbl_Leads.CollectionType (varchar).
const (
	LeadCollectionHome   = "Home"
	LeadCollectionCenter = "Center"
	LeadCollectionCamp   = "Camp"
)

// ParseLeadCollectionType normalizes input to Home, Center, or Camp.
func ParseLeadCollectionType(s string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "home":
		return LeadCollectionHome, nil
	case "center":
		return LeadCollectionCenter, nil
	case "camp":
		return LeadCollectionCamp, nil
	default:
		return "", fmt.Errorf("CollectionType must be Home, Center, or Camp")
	}
}

// Lead fitness / report approval tri-state (MediAdmin.tbl_Leads.IsFit, TINYINT).
// Values must fit SQL Server tinyint (0–255): 0 = on hold, 1 = fit, 2 = unfit.
const (
	LeadFitHold  int8 = 0
	LeadFitFit   int8 = 1
	LeadFitUnfit int8 = 2
)

// FitnessStatusFromIsFitPtr maps MediAdmin.tbl_Leads.IsFit for read APIs (NULL → "Empty").
// Legacy -1 is treated as unfit (same as 2).
func FitnessStatusFromIsFitPtr(p *int8) string {
	if p == nil {
		return "Empty"
	}
	switch *p {
	case LeadFitHold:
		return "On Hold"
	case LeadFitFit:
		return "Fit"
	case LeadFitUnfit:
		return "UnFit"
	default:
		if *p == -1 {
			return "UnFit"
		}
		return ""
	}
}

// LeadListFitnessFilter selects rows by MediAdmin.tbl_Leads.IsFit for GET /leads list queries.
type LeadListFitnessFilter int8

const (
	LeadListFitnessFilterNone LeadListFitnessFilter = 0
	LeadListFitnessFilterEmpty LeadListFitnessFilter = 1
	LeadListFitnessFilterOnHold LeadListFitnessFilter = 2
	LeadListFitnessFilterFit LeadListFitnessFilter = 3
	LeadListFitnessFilterUnfit LeadListFitnessFilter = 4
)

// ParseLeadListFitnessFilter parses optional query param fitnessStatus / FitnessStatus (case-insensitive).
// Accepted values align with JSON FitnessStatus: Empty, Not Assessed (same as Empty / IsFit NULL), On Hold, Fit, UnFit (also onhold, hold, unfit).
func ParseLeadListFitnessFilter(s string) (LeadListFitnessFilter, error) {
	n := strings.TrimSpace(s)
	if n == "" {
		return LeadListFitnessFilterNone, nil
	}
	lower := strings.ToLower(n)
	compact := strings.ReplaceAll(lower, " ", "")
	compactNoDash := strings.ReplaceAll(compact, "-", "")
	switch {
	case lower == "empty" || lower == "not assessed" || compactNoDash == "notassessed":
		return LeadListFitnessFilterEmpty, nil
	case compact == "onhold" || lower == "on hold" || lower == "hold":
		return LeadListFitnessFilterOnHold, nil
	case compact == "fit":
		return LeadListFitnessFilterFit, nil
	case compact == "unfit" || lower == "un fit":
		return LeadListFitnessFilterUnfit, nil
	default:
		return 0, fmt.Errorf(`fitnessStatus must be one of: Empty, Not Assessed, On Hold, Fit, UnFit`)
	}
}

// LeadEmpIDMaxLen matches MediAdmin.tbl_Leads.EmpID (varchar(10)).
const LeadEmpIDMaxLen = 10

// ValidateLeadEmpID returns an error when s exceeds LeadEmpIDMaxLen (empty is allowed).
func ValidateLeadEmpID(s string) error {
	if len(strings.TrimSpace(s)) > LeadEmpIDMaxLen {
		return fmt.Errorf("EmpID must be at most %d characters", LeadEmpIDMaxLen)
	}
	return nil
}

// LeadStoreIDMaxLen matches MediAdmin.tbl_Leads.StoreID (varchar(15)).
const LeadStoreIDMaxLen = 15

// ValidateLeadStoreID returns an error when s exceeds LeadStoreIDMaxLen (empty is allowed).
func ValidateLeadStoreID(s string) error {
	if len(strings.TrimSpace(s)) > LeadStoreIDMaxLen {
		return fmt.Errorf("StoreID must be at most %d characters", LeadStoreIDMaxLen)
	}
	return nil
}

// LeadBelongsToStore is true when StoreMasterID or varchar StoreID matches the store JWT userId.
func LeadBelongsToStore(storeID int64, storeMasterID *int64, storeIDText string) bool {
	if storeID <= 0 {
		return false
	}
	if storeMasterID != nil && *storeMasterID == storeID {
		return true
	}
	return strings.TrimSpace(storeIDText) == strconv.FormatInt(storeID, 10)
}

// LeadStatusIDDefault is the initial status for new leads when the client omits LeadStatusID (JSON zero / empty CSV).
const LeadStatusIDDefault int8 = 1

// Lead workflow: 8 = report uploaded (eligible for POST /leads/{id}/reports/approve); 9 = report approved (set when IsFit = 1 on approve).
const (
	LeadStatusIDReportUploaded int8 = 8
	LeadStatusIDReportApproved int8 = 9
	// LeadStatusIDClientDownloadNoFitGate is the minimum LeadStatusID at which clients may obtain a report
	// download URL without IsFit / IsReportDownloadable checks (e.g. post certificate merge, typically 10).
	LeadStatusIDClientDownloadNoFitGate int8 = 10
	LeadStatusIDReportDownloaded        int8 = 11
)

// LeadStatusIDForbiddenForPUTLead is true when id must not be applied via PUT /api/v1/leads/{id}
// (statuses >= 8 are workflow-only: report pipeline through download/sent).
func LeadStatusIDForbiddenForPUTLead(id int8) bool {
	return id >= LeadStatusIDReportUploaded
}

// LeadStatusIDDowngradeForbiddenForPUTLead is true when the lead is already in report workflow (>= 8)
// but the client tries to set LeadStatusID to a pre-workflow value (< 8) via PUT.
func LeadStatusIDDowngradeForbiddenForPUTLead(existingLeadStatusID, requestedLeadStatusID int8) bool {
	return existingLeadStatusID >= LeadStatusIDReportUploaded && requestedLeadStatusID < LeadStatusIDReportUploaded
}

// LeadStatusIDReportApproval is the LeadStatusID required on the lead row before approve may run (report uploaded).
const LeadStatusIDReportApproval = LeadStatusIDReportUploaded

type Lead struct {
	LeadID                        int64
	ClientID                      int64
	ClientName                    string `json:"ClientName,omitempty"`
	PatientID                     string
	PatientName                   string
	Age                           int8
	Gender                        string
	PackageID                     int
	ContactNumber                 string
	Emailid                       string
	Address                       string
	CityID                        int32
	CityName                      string `json:"CityName,omitempty"`
	StateID                       int32
	StateName                     string `json:"StateName,omitempty"`
	Pincode                       string
	EmpID                         string `json:"EmpID"`
	StoreID                       string `json:"StoreID"`
	StoreMasterID                 *int64 `json:"StoreMasterID,omitempty"`
	// StoreName and StoreCity come from tbl_StoreMaster (MedLyfe only). Omitted on other schemas.
	StoreName                     string `json:"StoreName,omitempty"`
	StoreCity                     string `json:"StoreCity,omitempty"`
	CollectionType                string `json:"CollectionType"`
	LeadStatusID                  int8
	AppointmentAt                 *timeutil.StoredTime `json:"AppointmentAt"`
	LabID                         *int64               `json:"LabID,omitempty"`
	BrandID                       *int64               `json:"BrandID,omitempty"`
	LabName                       string               `json:"LabName,omitempty"`
	IsFit                         *int8                `json:"IsFit,omitempty"`
	FitnessStatus                 string               `json:"FitnessStatus,omitempty"`
	IsReportDownloadable          bool                 `json:"IsReportDownloadable"`
	ApprovalRemarks               string               `json:"ApprovalRemarks,omitempty"`
	FitUpdatedOn                  *timeutil.ISTTime    `json:"FitUpdatedOn,omitempty"`
	IsFitCertificateTobeGenerated bool                 `json:"IsFitCertificateTobeGenerated"`
	IsFitCertifiedGenerated       bool                 `json:"IsFitCertifiedGenerated"`
	ReportURL                     string               `json:"ReportURL,omitempty"`
	CreatedBy                     int64
	CreatedOn                     timeutil.ISTTime
	LastUpdatedBy                 int64
	LastUpdatedOn                 timeutil.ISTTime
}

// LeadDetail is lead with resolved PackageName for API response (ClientName is on embedded Lead).
type LeadDetail struct {
	Lead
	PackageName string `json:"PackageName,omitempty"`
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
