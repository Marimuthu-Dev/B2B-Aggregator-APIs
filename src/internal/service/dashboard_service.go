package service

import (
	"b2b-diagnostic-aggregator/apis/internal/domain"
	"b2b-diagnostic-aggregator/apis/internal/dto"
	"b2b-diagnostic-aggregator/apis/internal/repository"
)

type DashboardService interface {
	GetLabTrackingSummary(filter domain.LabTrackingFilter) (*dto.LabTrackingSummaryResponse, error)
	GetLabTrackingFilterOptions() (*dto.LabTrackingFilterOptionsResponse, error)
}

type dashboardService struct {
	repo repository.DashboardRepository
}

func NewDashboardService(repo repository.DashboardRepository) DashboardService {
	return &dashboardService{repo: repo}
}

func toLabTrackingCountsResponse(c domain.LabTrackingCounts) dto.LabTrackingCountsResponse {
	return dto.LabTrackingCountsResponse{
		TotalAppointments:      c.TotalAppointments,
		PendingSchedule:        c.PendingSchedule,
		AppointmentsScheduled:  c.AppointmentsScheduled,
		SamplePending:          c.SamplePending,
		ReportsPendingUpload:   c.ReportsPendingUpload,
		ReportsUploaded:        c.ReportsUploaded,
		ReportsPendingApproval: c.ReportsPendingApproval,
		ReportsApproved:        c.ReportsApproved,
	}
}

func toDashboardOptionResponses(opts []domain.DashboardOption) []dto.DashboardOptionResponse {
	out := make([]dto.DashboardOptionResponse, 0, len(opts))
	for _, o := range opts {
		out = append(out, dto.DashboardOptionResponse{ID: o.ID, Name: o.Name, ClientTypeID: o.ClientTypeID})
	}
	return out
}

func (s *dashboardService) GetLabTrackingSummary(filter domain.LabTrackingFilter) (*dto.LabTrackingSummaryResponse, error) {
	summary, err := s.repo.LabTrackingSummary(filter)
	if err != nil {
		return nil, err
	}
	res := &dto.LabTrackingSummaryResponse{
		Counts: toLabTrackingCountsResponse(summary.Counts),
		Labs:   make([]dto.LabTrackingLabResponse, 0, len(summary.Labs)),
	}
	for _, lab := range summary.Labs {
		name := lab.LabName
		if lab.LabID == nil {
			name = "Lab Not Assigned"
		}
		res.Labs = append(res.Labs, dto.LabTrackingLabResponse{
			LabID:                     lab.LabID,
			LabName:                   name,
			LabTrackingCountsResponse: toLabTrackingCountsResponse(lab.LabTrackingCounts),
		})
	}
	return res, nil
}

func (s *dashboardService) GetLabTrackingFilterOptions() (*dto.LabTrackingFilterOptionsResponse, error) {
	opts, err := s.repo.LabTrackingFilterOptions()
	if err != nil {
		return nil, err
	}
	return &dto.LabTrackingFilterOptionsResponse{
		Cities:  toDashboardOptionResponses(opts.Cities),
		Labs:    toDashboardOptionResponses(opts.Labs),
		Clients: toDashboardOptionResponses(opts.Clients),
	}, nil
}
