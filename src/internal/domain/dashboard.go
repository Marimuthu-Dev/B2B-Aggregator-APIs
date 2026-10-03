package domain

import "time"

// LabTrackingFilter narrows the Ops "Appointment & Report Tracking" dashboard.
// All fields are optional; nil means "no filter". From/To apply to tbl_Leads.CreatedOn (appointment received).
type LabTrackingFilter struct {
	From         *time.Time
	To           *time.Time
	CityID       *int64
	LabID        *int64
	ClientID     *int64
	ClientTypeID *int64 // tbl_ClientMaster.ClientTypeID (1=Food, 2=Corporate, 3=PPMC)
}

// LabTrackingCounts are lead counts bucketed by LeadStatusID.
//
//	1-3  New / Accepted / Lab Assigned          -> PendingSchedule
//	>=4  Appointment Scheduled onwards          -> AppointmentsScheduled
//	4-5  Scheduled / Confirmed (no sample yet)  -> SamplePending
//	6-7  Sample Collected / In-Progress         -> ReportsPendingUpload
//	>=8  Report Uploaded onwards                -> ReportsUploaded
//	8    Report Uploaded (awaiting approval)    -> ReportsPendingApproval
//	>=9  Report Approved onwards                -> ReportsApproved
type LabTrackingCounts struct {
	TotalAppointments      int64
	PendingSchedule        int64
	AppointmentsScheduled  int64
	SamplePending          int64
	ReportsPendingUpload   int64
	ReportsUploaded        int64
	ReportsPendingApproval int64
	ReportsApproved        int64
}

// LabTrackingLabRow is the per-lab breakdown (LabID nil = lab not yet assigned).
type LabTrackingLabRow struct {
	LabID   *int64
	LabName string
	LabTrackingCounts
}

// LabTrackingSummary is the full dashboard payload.
type LabTrackingSummary struct {
	Counts LabTrackingCounts
	Labs   []LabTrackingLabRow
}

// DashboardOption is a simple id/name pair for dashboard filter dropdowns.
type DashboardOption struct {
	ID           int64
	Name         string
	ClientTypeID *int64
}

// LabTrackingFilterOptions holds dropdown data for the dashboard filter bar.
type LabTrackingFilterOptions struct {
	Cities  []DashboardOption
	Labs    []DashboardOption
	Clients []DashboardOption
}
