package repository

import (
	"database/sql"

	"b2b-diagnostic-aggregator/apis/internal/domain"
	persistencemodels "b2b-diagnostic-aggregator/apis/internal/persistence/models"

	"gorm.io/gorm"
)

// DashboardRepository runs read-only aggregate queries for dashboards.
// All table names are resolved through persistencemodels.Table(), so queries follow DB_SCHEMA.
type DashboardRepository interface {
	LabTrackingSummary(filter domain.LabTrackingFilter) (*domain.LabTrackingSummary, error)
	LabTrackingFilterOptions() (*domain.LabTrackingFilterOptions, error)
}

type dashboardRepository struct {
	db *gorm.DB
}

func NewDashboardRepository(db *gorm.DB) DashboardRepository {
	return &dashboardRepository{db: db}
}

// labTrackingCountsSelect buckets leads by LeadStatusID (see domain.LabTrackingCounts for the mapping).
const labTrackingCountsSelect = `COUNT(1) AS TotalAppointments,
	COALESCE(SUM(CASE WHEN l.LeadStatusID BETWEEN 1 AND 3 THEN 1 ELSE 0 END), 0) AS PendingSchedule,
	COALESCE(SUM(CASE WHEN l.LeadStatusID >= 4 THEN 1 ELSE 0 END), 0) AS AppointmentsScheduled,
	COALESCE(SUM(CASE WHEN l.LeadStatusID BETWEEN 4 AND 5 THEN 1 ELSE 0 END), 0) AS SamplePending,
	COALESCE(SUM(CASE WHEN l.LeadStatusID BETWEEN 6 AND 7 THEN 1 ELSE 0 END), 0) AS ReportsPendingUpload,
	COALESCE(SUM(CASE WHEN l.LeadStatusID >= 8 THEN 1 ELSE 0 END), 0) AS ReportsUploaded,
	COALESCE(SUM(CASE WHEN l.LeadStatusID = 8 THEN 1 ELSE 0 END), 0) AS ReportsPendingApproval,
	COALESCE(SUM(CASE WHEN l.LeadStatusID >= 9 THEN 1 ELSE 0 END), 0) AS ReportsApproved`

type LabTrackingCountsRow struct {
	TotalAppointments      int64 `gorm:"column:TotalAppointments"`
	PendingSchedule        int64 `gorm:"column:PendingSchedule"`
	AppointmentsScheduled  int64 `gorm:"column:AppointmentsScheduled"`
	SamplePending          int64 `gorm:"column:SamplePending"`
	ReportsPendingUpload   int64 `gorm:"column:ReportsPendingUpload"`
	ReportsUploaded        int64 `gorm:"column:ReportsUploaded"`
	ReportsPendingApproval int64 `gorm:"column:ReportsPendingApproval"`
	ReportsApproved        int64 `gorm:"column:ReportsApproved"`
}

func (s LabTrackingCountsRow) toDomain() domain.LabTrackingCounts {
	return domain.LabTrackingCounts{
		TotalAppointments:      s.TotalAppointments,
		PendingSchedule:        s.PendingSchedule,
		AppointmentsScheduled:  s.AppointmentsScheduled,
		SamplePending:          s.SamplePending,
		ReportsPendingUpload:   s.ReportsPendingUpload,
		ReportsUploaded:        s.ReportsUploaded,
		ReportsPendingApproval: s.ReportsPendingApproval,
		ReportsApproved:        s.ReportsApproved,
	}
}

type labTrackingLabScan struct {
	LabID   sql.NullInt64  `gorm:"column:LabID"`
	LabName sql.NullString `gorm:"column:LabName"`
	LabTrackingCountsRow
}

// labTrackingBaseQuery builds FROM/JOIN/WHERE shared by the summary and per-lab queries.
// All filter values are bound as parameters.
func (r *dashboardRepository) labTrackingBaseQuery(filter domain.LabTrackingFilter) *gorm.DB {
	leadTable := persistencemodels.Lead{}.TableName()
	clientTable := persistencemodels.Client{}.TableName()
	labTable := persistencemodels.Lab{}.TableName()

	q := r.db.Table(leadTable + " AS l").
		Joins("LEFT JOIN " + clientTable + " AS cm ON l.ClientID = cm.ClientID").
		Joins("LEFT JOIN " + labTable + " AS lm ON l.LabID = lm.LabID")

	if filter.From != nil {
		q = q.Where("l.CreatedOn >= ?", *filter.From)
	}
	if filter.To != nil {
		q = q.Where("l.CreatedOn <= ?", *filter.To)
	}
	if filter.CityID != nil {
		q = q.Where("l.CityID = ?", *filter.CityID)
	}
	if filter.LabID != nil {
		q = q.Where("l.LabID = ?", *filter.LabID)
	}
	if filter.ClientID != nil {
		q = q.Where("l.ClientID = ?", *filter.ClientID)
	}
	if filter.ClientTypeID != nil {
		q = q.Where("cm.ClientTypeID = ?", *filter.ClientTypeID)
	}
	return q
}

func (r *dashboardRepository) LabTrackingSummary(filter domain.LabTrackingFilter) (*domain.LabTrackingSummary, error) {
	var totals LabTrackingCountsRow
	if err := r.labTrackingBaseQuery(filter).
		Select(labTrackingCountsSelect).
		Scan(&totals).Error; err != nil {
		return nil, err
	}

	var labRows []labTrackingLabScan
	if err := r.labTrackingBaseQuery(filter).
		Select("l.LabID AS LabID, MAX(lm.LabName) AS LabName, " + labTrackingCountsSelect).
		Group("l.LabID").
		Order("TotalAppointments DESC").
		Scan(&labRows).Error; err != nil {
		return nil, err
	}

	out := &domain.LabTrackingSummary{
		Counts: totals.toDomain(),
		Labs:   make([]domain.LabTrackingLabRow, 0, len(labRows)),
	}
	for _, row := range labRows {
		item := domain.LabTrackingLabRow{LabTrackingCounts: row.LabTrackingCountsRow.toDomain()}
		if row.LabID.Valid {
			id := row.LabID.Int64
			item.LabID = &id
		}
		if row.LabName.Valid {
			item.LabName = row.LabName.String
		}
		out.Labs = append(out.Labs, item)
	}
	return out, nil
}

type dashboardOptionScan struct {
	ID           int64         `gorm:"column:ID"`
	Name         string        `gorm:"column:Name"`
	ClientTypeID sql.NullInt64 `gorm:"column:ClientTypeID"`
}

func mapDashboardOptions(rows []dashboardOptionScan) []domain.DashboardOption {
	out := make([]domain.DashboardOption, 0, len(rows))
	for _, row := range rows {
		opt := domain.DashboardOption{ID: row.ID, Name: row.Name}
		if row.ClientTypeID.Valid {
			v := row.ClientTypeID.Int64
			opt.ClientTypeID = &v
		}
		out = append(out, opt)
	}
	return out
}

func (r *dashboardRepository) LabTrackingFilterOptions() (*domain.LabTrackingFilterOptions, error) {
	leadTable := persistencemodels.Lead{}.TableName()
	cityTable := persistencemodels.CityMaster{}.TableName()
	labTable := persistencemodels.Lab{}.TableName()
	clientTable := persistencemodels.Client{}.TableName()

	// Only cities that actually have leads, so the dropdown stays relevant.
	var cities []dashboardOptionScan
	if err := r.db.Table(cityTable + " AS ctm").
		Select("ctm.CityID AS ID, ctm.CityName AS Name").
		Where("EXISTS (SELECT 1 FROM " + leadTable + " AS l WHERE l.CityID = ctm.CityID)").
		Order("ctm.CityName ASC").
		Scan(&cities).Error; err != nil {
		return nil, err
	}

	var labs []dashboardOptionScan
	if err := r.db.Table(labTable).
		Select("LabID AS ID, LabName AS Name").
		Order("LabName ASC").
		Scan(&labs).Error; err != nil {
		return nil, err
	}

	var clients []dashboardOptionScan
	if err := r.db.Table(clientTable).
		Select("ClientID AS ID, ClientName AS Name, ClientTypeID AS ClientTypeID").
		Order("ClientName ASC").
		Scan(&clients).Error; err != nil {
		return nil, err
	}

	return &domain.LabTrackingFilterOptions{
		Cities:  mapDashboardOptions(cities),
		Labs:    mapDashboardOptions(labs),
		Clients: mapDashboardOptions(clients),
	}, nil
}
