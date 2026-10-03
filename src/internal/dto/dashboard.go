package dto

// LabTrackingCountsResponse is the KPI block of the Ops lab tracking dashboard.
type LabTrackingCountsResponse struct {
	TotalAppointments      int64 `json:"TotalAppointments"`
	PendingSchedule        int64 `json:"PendingSchedule"`
	AppointmentsScheduled  int64 `json:"AppointmentsScheduled"`
	SamplePending          int64 `json:"SamplePending"`
	ReportsPendingUpload   int64 `json:"ReportsPendingUpload"`
	ReportsUploaded        int64 `json:"ReportsUploaded"`
	ReportsPendingApproval int64 `json:"ReportsPendingApproval"`
	ReportsApproved        int64 `json:"ReportsApproved"`
}

// LabTrackingLabResponse is one row of the per-lab breakdown. LabID is null for leads without a lab.
type LabTrackingLabResponse struct {
	LabID   *int64 `json:"LabID"`
	LabName string `json:"LabName"`
	LabTrackingCountsResponse
}

// LabTrackingSummaryResponse is returned by GET /api/v1/dashboard/lab-tracking.
type LabTrackingSummaryResponse struct {
	Counts LabTrackingCountsResponse `json:"Counts"`
	Labs   []LabTrackingLabResponse  `json:"Labs"`
}

// DashboardOptionResponse is an id/name pair for dashboard filter dropdowns.
type DashboardOptionResponse struct {
	ID           int64  `json:"ID"`
	Name         string `json:"Name"`
	ClientTypeID *int64 `json:"ClientTypeID,omitempty"`
}

// LabTrackingFilterOptionsResponse is returned by GET /api/v1/dashboard/lab-tracking/filters.
type LabTrackingFilterOptionsResponse struct {
	Cities  []DashboardOptionResponse `json:"Cities"`
	Labs    []DashboardOptionResponse `json:"Labs"`
	Clients []DashboardOptionResponse `json:"Clients"`
}
