package repository

import (
	"database/sql"
	"strings"

	"b2b-diagnostic-aggregator/apis/internal/domain"
	persistencemodels "b2b-diagnostic-aggregator/apis/internal/persistence/models"
	"b2b-diagnostic-aggregator/apis/internal/timeutil"
)

func mapLeadToDomainWithOptionalJoinedNames(p persistencemodels.Lead, labName, clientName, cityName, stateName sql.NullString) domain.Lead {
	d := mapLeadToDomain(p)
	if labName.Valid {
		d.LabName = strings.TrimSpace(labName.String)
	}
	if clientName.Valid {
		d.ClientName = strings.TrimSpace(clientName.String)
	}
	if cityName.Valid {
		d.CityName = strings.TrimSpace(cityName.String)
	}
	if stateName.Valid {
		d.StateName = strings.TrimSpace(stateName.String)
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
		EmpID:                         derefString(p.EmpID),
		StoreID:                       derefString(p.StoreID),
		CollectionType:                p.CollectionType,
		LeadStatusID:                  p.LeadStatusID,
		AppointmentAt:                 timeutil.StoredFromTimePtr(p.AppointmentAt),
		LabID:                         p.LabID,
		BrandID:                       p.BrandID,
		IsFit:                         isFitDomainFromDB(p.IsFit),
		FitnessStatus:                 domain.FitnessStatusFromIsFitPtr(p.IsFit),
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
		EmpID:                         stringPtrOrNil(d.EmpID),
		StoreID:                       stringPtrOrNil(d.StoreID),
		CollectionType:                d.CollectionType,
		LeadStatusID:                  d.LeadStatusID,
		AppointmentAt:                 timeutil.StoredToTimePtr(d.AppointmentAt),
		LabID:                         d.LabID,
		BrandID:                       d.BrandID,
		IsFit:                         isFitDBFromDomain(d.IsFit),
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

// isFitDomainFromDB maps DB IsFit to domain: NULL stays nil; legacy -1 → unfit pointer.
func isFitDomainFromDB(p *int8) *int8 {
	if p == nil {
		return nil
	}
	if *p == -1 {
		v := domain.LeadFitUnfit
		return &v
	}
	v := *p
	return &v
}

func isFitDBFromDomain(v *int8) *int8 {
	if v == nil {
		return nil
	}
	x := *v
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
