package repository

import (
	"database/sql"
	"strings"

	"b2b-diagnostic-aggregator/apis/internal/domain"
	persistencemodels "b2b-diagnostic-aggregator/apis/internal/persistence/models"
	"b2b-diagnostic-aggregator/apis/internal/timeutil"
)

func mapLeadToDomainWithOptionalLabName(p persistencemodels.Lead, labName sql.NullString) domain.Lead {
	d := mapLeadToDomain(p)
	if labName.Valid {
		d.LabName = strings.TrimSpace(labName.String)
	}
	return d
}

func mapLeadToDomain(p persistencemodels.Lead) domain.Lead {
	return domain.Lead{
		LeadID:                        p.LeadID,
		ClientID:                      p.ClientID,
		PatientID:                     p.PatientID,
		PatientName:                   p.PatientName,
		Age:                           p.Age,
		Gender:                        p.Gender,
		PackageID:                     p.PackageID,
		ContactNumber:                 p.ContactNumber,
		Emailid:                       p.Emailid,
		Address:                       p.Address,
		CityID:                        p.CityID,
		StateID:                       p.StateID,
		Pincode:                       p.Pincode,
		CollectionType:                p.CollectionType,
		LeadStatusID:                  p.LeadStatusID,
		AppointmentAt:                 timeutil.StoredFromTimePtr(p.AppointmentAt),
		LabID:                         p.LabID,
		IsFit:                         isFitFromPtr(p.IsFit),
		IsReportDownloadable:          p.IsReportDownloadable,
		ApprovalRemarks:               derefString(p.ApprovalRemarks),
		FitUpdatedOn:                  timeutil.FromTimePtr(p.FitUpdatedOn),
		IsFitCertificateTobeGenerated: p.IsFitCertificateTobeGenerated,
		IsFitCertifiedGenerated:       p.IsFitCertifiedGenerated,
		ReportURL:                     derefReportURL(p.ReportURL),
		CreatedBy:                     p.CreatedBy,
		CreatedOn:                     timeutil.FromTime(p.CreatedOn),
		LastUpdatedBy:                 p.LastUpdatedBy,
		LastUpdatedOn:                 timeutil.FromTime(p.LastUpdatedOn),
	}
}

func mapLeadToPersistence(d domain.Lead) persistencemodels.Lead {
	return persistencemodels.Lead{
		LeadID:                        d.LeadID,
		ClientID:                      d.ClientID,
		PatientID:                     d.PatientID,
		PatientName:                   d.PatientName,
		Age:                           d.Age,
		Gender:                        d.Gender,
		PackageID:                     d.PackageID,
		ContactNumber:                 d.ContactNumber,
		Emailid:                       d.Emailid,
		Address:                       d.Address,
		CityID:                        d.CityID,
		StateID:                       d.StateID,
		Pincode:                       d.Pincode,
		CollectionType:                d.CollectionType,
		LeadStatusID:                  d.LeadStatusID,
		AppointmentAt:                 timeutil.StoredToTimePtr(d.AppointmentAt),
		LabID:                         d.LabID,
		IsFit:                         isFitToPtr(d.IsFit),
		IsReportDownloadable:          d.IsReportDownloadable,
		ApprovalRemarks:               stringPtrOrNil(d.ApprovalRemarks),
		FitUpdatedOn:                  timeutil.ToTimePtr(d.FitUpdatedOn),
		IsFitCertificateTobeGenerated: d.IsFitCertificateTobeGenerated,
		IsFitCertifiedGenerated:       d.IsFitCertifiedGenerated,
		ReportURL:                     stringPtrOrNil(d.ReportURL),
		CreatedBy:                     d.CreatedBy,
		CreatedOn:                     d.CreatedOn.ToTime(),
		LastUpdatedBy:                 d.LastUpdatedBy,
		LastUpdatedOn:                 d.LastUpdatedOn.ToTime(),
	}
}

func derefReportURL(p *string) string {
	if p == nil {
		return ""
	}
	return strings.TrimSpace(*p)
}

func stringPtrOrNil(s string) *string {
	t := strings.TrimSpace(s)
	if t == "" {
		return nil
	}
	return &t
}

func derefString(p *string) string {
	if p == nil {
		return ""
	}
	return strings.TrimSpace(*p)
}

func isFitFromPtr(p *int8) int8 {
	if p == nil {
		return domain.LeadFitHold
	}
	// Legacy rows may have used -1 for unfit before tinyint-safe encoding.
	if *p == -1 {
		return domain.LeadFitUnfit
	}
	return *p
}

func isFitToPtr(v int8) *int8 {
	x := v
	return &x
}

func mapLeadsToDomain(leads []persistencemodels.Lead) []domain.Lead {
	if len(leads) == 0 {
		return nil
	}
	mapped := make([]domain.Lead, len(leads))
	for i, lead := range leads {
		mapped[i] = mapLeadToDomain(lead)
	}
	return mapped
}

func mapLeadHistoryToDomain(p persistencemodels.LeadHistory) domain.LeadHistory {
	return domain.LeadHistory{
		UID:       p.UID,
		LeadID:    p.LeadID,
		Action:    p.Action,
		CreatedBy: p.CreatedBy,
		CreatedOn: timeutil.FromTime(p.CreatedOn),
	}
}

func mapLeadHistoryToPersistence(d domain.LeadHistory) persistencemodels.LeadHistory {
	return persistencemodels.LeadHistory{
		UID:       d.UID,
		LeadID:    d.LeadID,
		Action:    d.Action,
		CreatedBy: d.CreatedBy,
		CreatedOn: d.CreatedOn.ToTime(),
	}
}

func mapLeadHistoriesToDomain(histories []persistencemodels.LeadHistory) []domain.LeadHistory {
	if len(histories) == 0 {
		return nil
	}
	mapped := make([]domain.LeadHistory, len(histories))
	for i, history := range histories {
		mapped[i] = mapLeadHistoryToDomain(history)
	}
	return mapped
}

func mapLeadHistoriesToPersistence(histories []domain.LeadHistory) []persistencemodels.LeadHistory {
	if len(histories) == 0 {
		return nil
	}
	mapped := make([]persistencemodels.LeadHistory, len(histories))
	for i, history := range histories {
		mapped[i] = mapLeadHistoryToPersistence(history)
	}
	return mapped
}
