package repository

import (
	"database/sql"
	"time"

	"b2b-diagnostic-aggregator/apis/internal/domain"
	persistencemodels "b2b-diagnostic-aggregator/apis/internal/persistence/models"

	"gorm.io/gorm"
)

type LeadRepository interface {
	FindAll() ([]domain.Lead, error)
	List(filter LeadListFilter) ([]domain.Lead, int64, error)
	FindByID(id int64) (*domain.Lead, error)
	ExistsByID(id int64) (bool, error)
	Create(l *domain.Lead) error
	Update(l *domain.Lead) error
	Delete(id int64) error
	// UpdateStatusForIDs sets status and audit columns; if labID is non-nil, LabID is set on all matching rows.
	UpdateStatusForIDs(leadIDs []int64, statusID int8, lastUpdatedBy int64, labID *int64) (int64, error)
	FindByClientID(clientID int64) ([]domain.Lead, error)
	FindByStatus(statusID int8) ([]domain.Lead, error)
	FindByPackage(packageID int) ([]domain.Lead, error)
	FindByPatientID(patientID string) (*domain.Lead, error)
	FindByContactNumber(contactNumber string) ([]domain.Lead, error)
	FindByEmail(email string) ([]domain.Lead, error)
	// FindActiveLeadStatusIDByName resolves LeadStatusID from MediAdmin.tbl_LeadStatusMaster (IsActive = 1).
	FindActiveLeadStatusIDByName(name string) (int8, error)
	UpdateLeadReportURLAndStatus(leadID int64, reportURL string, statusID int8, userID int64) error
	// FindLeadsPendingFitCertification returns FIT leads pending worker processing. Does not filter on IsFitCertificateTobeGenerated (all values eligible by query).
	FindLeadsPendingFitCertification(limit int, pendingLeadStatusID int8) ([]domain.Lead, error)
	// MarkFitCertificationGenerated sets certification flags and status if the lead still matches pending criteria; logs history. Returns whether a row was updated.
	MarkFitCertificationGenerated(leadID int64, userID int64, fromLeadStatusID, toLeadStatusID int8) (updated bool, err error)
	// MarkReportReadyToDownload advances a lead to the downloadable state without generating a certificate; logs history. Returns whether a row was updated.
	MarkReportReadyToDownload(leadID int64, userID int64, fromLeadStatusID, toLeadStatusID int8) (updated bool, err error)
	// UpdateLeadReportApproval sets IsFit (0=hold, 1=fit, 2=unfit), download flag, IsFitCertificateTobeGenerated, remarks, FitUpdatedOn, and last-updated audit. Only rows with LeadStatusID = 8 (uploaded) match; when IsFit = 1, LeadStatusID is set to 9 (approved). Otherwise LeadStatusID is not changed. remarks nil or empty clears ApprovalRemarks.
	UpdateLeadReportApproval(leadID int64, lastUpdatedBy int64, isFit int8, allowDownload bool, isFitCertificateTobeGenerated bool, remarks *string) (rowsAffected int64, err error)
}

type leadRepository struct {
	db *gorm.DB
}

// leadListScan is tbl_Leads with an optional lab name from LEFT JOIN tbl_LabMaster.
type leadListScan struct {
	persistencemodels.Lead
	JoinedLabName sql.NullString `gorm:"column:joined_lab_name"`
}

func NewLeadRepository(db *gorm.DB) LeadRepository {
	return &leadRepository{db: db}
}

func (r *leadRepository) FindAll() ([]domain.Lead, error) {
	var leads []persistencemodels.Lead
	err := r.db.Find(&leads).Error
	return mapLeadsToDomain(leads), err
}

func (r *leadRepository) List(filter LeadListFilter) ([]domain.Lead, int64, error) {
	q := r.leadListJoinedQuery(filter)

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	sortColumn := "l." + mapLeadSortColumn(filter.SortBy)
	order := normalizeSortOrder(filter.SortOrder)
	offset := (filter.Page - 1) * filter.PageSize

	var rows []leadListScan
	err := r.leadListJoinedQuery(filter).
		Select("l.*, lm.LabName AS joined_lab_name").
		Order(sortColumn + " " + order).
		Limit(filter.PageSize).
		Offset(offset).
		Find(&rows).Error
	if err != nil {
		return nil, 0, err
	}

	out := make([]domain.Lead, len(rows))
	for i := range rows {
		out[i] = mapLeadToDomainWithOptionalLabName(rows[i].Lead, rows[i].JoinedLabName)
	}
	return out, total, nil
}

func (r *leadRepository) leadListJoinedQuery(filter LeadListFilter) *gorm.DB {
	leadTable := persistencemodels.Lead{}.TableName()
	labTable := persistencemodels.Lab{}.TableName()
	q := r.db.Table(leadTable + " AS l").
		Joins("LEFT JOIN " + labTable + " AS lm ON l.LabID = lm.LabID")
	if filter.LeadID != nil {
		q = q.Where("l.LeadID = ?", *filter.LeadID)
	}
	if filter.ClientID != nil {
		q = q.Where("l.ClientID = ?", *filter.ClientID)
	}
	if filter.LabID != nil {
		q = q.Where("l.LabID = ?", *filter.LabID)
	}
	if filter.StatusID != nil {
		q = q.Where("l.LeadStatusID = ?", *filter.StatusID)
	}
	if filter.PackageID != nil {
		q = q.Where("l.PackageID = ?", *filter.PackageID)
	}
	if filter.CollectionType != nil && *filter.CollectionType != "" {
		q = q.Where("l.CollectionType = ?", *filter.CollectionType)
	}
	return q
}

func mapLeadSortColumn(sortBy string) string {
	switch sortBy {
	case "patientName":
		return "PatientName"
	case "clientId":
		return "ClientID"
	case "labId":
		return "LabID"
	case "statusId":
		return "LeadStatusID"
	case "createdOn":
		return "CreatedOn"
	case "collectionType":
		return "CollectionType"
	case "leadId":
		return "LeadID"
	default:
		return "LeadID"
	}
}

func (r *leadRepository) FindByID(id int64) (*domain.Lead, error) {
	var l persistencemodels.Lead
	err := r.db.First(&l, id).Error
	if err != nil {
		return nil, err
	}
	domainLead := mapLeadToDomain(l)
	return &domainLead, nil
}

func (r *leadRepository) ExistsByID(id int64) (bool, error) {
	var count int64
	if err := r.db.Model(&persistencemodels.Lead{}).Where("LeadID = ?", id).Limit(1).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *leadRepository) Create(l *domain.Lead) error {
	persist := mapLeadToPersistence(*l)
	if err := r.db.Create(&persist).Error; err != nil {
		return err
	}
	*l = mapLeadToDomain(persist)
	return nil
}

func (r *leadRepository) Update(l *domain.Lead) error {
	persist := mapLeadToPersistence(*l)
	if err := r.db.Save(&persist).Error; err != nil {
		return err
	}
	*l = mapLeadToDomain(persist)
	return nil
}

func (r *leadRepository) Delete(id int64) error {
	return r.db.Delete(&persistencemodels.Lead{}, id).Error
}

func (r *leadRepository) UpdateStatusForIDs(leadIDs []int64, statusID int8, lastUpdatedBy int64, labID *int64) (int64, error) {
	updates := map[string]interface{}{
		"LeadStatusID":  statusID,
		"LastUpdatedBy": lastUpdatedBy,
		"LastUpdatedOn": time.Now(),
	}
	if labID != nil {
		updates["LabID"] = *labID
	}
	result := r.db.Model(&persistencemodels.Lead{}).Where("LeadID IN ?", leadIDs).Updates(updates)
	return result.RowsAffected, result.Error
}

func (r *leadRepository) FindByClientID(clientID int64) ([]domain.Lead, error) {
	var leads []persistencemodels.Lead
	err := r.db.Where("ClientID = ?", clientID).Find(&leads).Error
	return mapLeadsToDomain(leads), err
}

func (r *leadRepository) FindByStatus(statusID int8) ([]domain.Lead, error) {
	var leads []persistencemodels.Lead
	err := r.db.Where("LeadStatusID = ?", statusID).Find(&leads).Error
	return mapLeadsToDomain(leads), err
}

func (r *leadRepository) FindByPackage(packageID int) ([]domain.Lead, error) {
	var leads []persistencemodels.Lead
	err := r.db.Where("PackageID = ?", packageID).Find(&leads).Error
	return mapLeadsToDomain(leads), err
}

func (r *leadRepository) FindByPatientID(patientID string) (*domain.Lead, error) {
	var l persistencemodels.Lead
	err := r.db.Where("PatientID = ?", patientID).First(&l).Error
	if err != nil {
		return nil, err
	}
	domainLead := mapLeadToDomain(l)
	return &domainLead, nil
}

func (r *leadRepository) FindByContactNumber(contactNumber string) ([]domain.Lead, error) {
	var leads []persistencemodels.Lead
	err := r.db.Where("ContactNumber = ?", contactNumber).Find(&leads).Error
	return mapLeadsToDomain(leads), err
}

func (r *leadRepository) FindByEmail(email string) ([]domain.Lead, error) {
	var leads []persistencemodels.Lead
	err := r.db.Where("EmailID = ?", email).Find(&leads).Error
	return mapLeadsToDomain(leads), err
}

func (r *leadRepository) FindActiveLeadStatusIDByName(name string) (int8, error) {
	var statusID int8
	err := r.db.Table("MediAdmin.tbl_LeadStatusMaster").
		Select("LeadStatusID").
		Where("LeadStatusName = ? AND IsActive = ?", name, true).
		Take(&statusID).Error
	if err != nil {
		return 0, err
	}
	return statusID, nil
}

func (r *leadRepository) UpdateLeadReportURLAndStatus(leadID int64, reportURL string, statusID int8, userID int64) error {
	result := r.db.Model(&persistencemodels.Lead{}).Where("LeadID = ?", leadID).Updates(map[string]interface{}{
		"ReportURL":     reportURL,
		"LeadStatusID":  statusID,
		"LastUpdatedBy": userID,
		"LastUpdatedOn": time.Now().UTC(),
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *leadRepository) FindLeadsPendingFitCertification(limit int, pendingLeadStatusID int8) ([]domain.Lead, error) {
	if limit <= 0 {
		limit = 10
	}
	var leads []persistencemodels.Lead
	err := r.db.Where("LeadStatusID = ? AND IsFit = ? AND IsFitCertifiedGenerated = ?",
		pendingLeadStatusID, domain.LeadFitFit, false).
		Where("ReportURL IS NOT NULL AND LTRIM(RTRIM(ReportURL)) <> ''").
		Order("LeadID ASC").
		Limit(limit).
		Find(&leads).Error
	return mapLeadsToDomain(leads), err
}

func (r *leadRepository) MarkFitCertificationGenerated(leadID int64, userID int64, fromLeadStatusID, toLeadStatusID int8) (bool, error) {
	var updated bool
	err := r.db.Transaction(func(tx *gorm.DB) error {
		now := time.Now().UTC()
		res := tx.Model(&persistencemodels.Lead{}).
			Where("LeadID = ? AND LeadStatusID = ? AND IsFit = ? AND IsFitCertifiedGenerated = ?",
				leadID, fromLeadStatusID, domain.LeadFitFit, false).
			Updates(map[string]interface{}{
				"IsFitCertifiedGenerated": true,
				"FitCertifiedGeneratedOn": now,
				"LeadStatusID":            toLeadStatusID,
				"LastUpdatedBy":           userID,
				"LastUpdatedOn":           now,
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return nil
		}
		updated = true
		h := persistencemodels.LeadHistory{
			LeadID:    leadID,
			Action:    domain.LeadActionFitCertificationGenerated,
			CreatedBy: userID,
			CreatedOn: now,
		}
		return tx.Create(&h).Error
	})
	return updated, err
}

func (r *leadRepository) MarkReportReadyToDownload(leadID int64, userID int64, fromLeadStatusID, toLeadStatusID int8) (bool, error) {
	var updated bool
	err := r.db.Transaction(func(tx *gorm.DB) error {
		now := time.Now().UTC()
		res := tx.Model(&persistencemodels.Lead{}).
			Where("LeadID = ? AND LeadStatusID = ? AND IsFit = ? AND IsFitCertifiedGenerated = ?",
				leadID, fromLeadStatusID, domain.LeadFitFit, false).
			Updates(map[string]interface{}{
				"LeadStatusID":  toLeadStatusID,
				"LastUpdatedBy": userID,
				"LastUpdatedOn": now,
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return nil
		}
		updated = true
		h := persistencemodels.LeadHistory{
			LeadID:    leadID,
			Action:    domain.LeadActionReportReadyToDownload,
			CreatedBy: userID,
			CreatedOn: now,
		}
		return tx.Create(&h).Error
	})
	return updated, err
}

func (r *leadRepository) UpdateLeadReportApproval(leadID int64, lastUpdatedBy int64, isFit int8, allowDownload bool, isFitCertificateTobeGenerated bool, remarks *string) (int64, error) {
	now := time.Now().UTC()
	var ar sql.NullString
	if remarks != nil {
		s := *remarks
		if s != "" {
			ar = sql.NullString{String: s, Valid: true}
		}
	}
	updates := map[string]interface{}{
		"IsFit":                         isFit,
		"IsReportDownloadable":          allowDownload,
		"IsFitCertificateTobeGenerated": isFitCertificateTobeGenerated,
		"ApprovalRemarks":               ar,
		"FitUpdatedOn":                  now,
		"LastUpdatedBy":                 lastUpdatedBy,
		"LastUpdatedOn":                 now,
		"LeadStatusID": gorm.Expr("CASE WHEN ? = ? THEN ? ELSE LeadStatusID END",
			isFit, domain.LeadFitFit, domain.LeadStatusIDReportApproved),
	}
	res := r.db.Model(&persistencemodels.Lead{}).Where("LeadID = ? AND LeadStatusID = ?", leadID, domain.LeadStatusIDReportApproval).Updates(updates)
	if res.Error != nil {
		return 0, res.Error
	}
	return res.RowsAffected, nil
}
