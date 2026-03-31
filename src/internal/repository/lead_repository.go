package repository

import (
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
	UpdateStatusForIDs(leadIDs []int64, statusID int8, lastUpdatedBy int64) (int64, error)
	FindByClientID(clientID int64) ([]domain.Lead, error)
	FindByStatus(statusID int8) ([]domain.Lead, error)
	FindByPackage(packageID int) ([]domain.Lead, error)
	FindByPatientID(patientID string) (*domain.Lead, error)
	FindByContactNumber(contactNumber string) ([]domain.Lead, error)
	FindByEmail(email string) ([]domain.Lead, error)
	// FindActiveLeadStatusIDByName resolves LeadStatusID from MediAdmin.tbl_LeadStatusMaster (IsActive = 1).
	FindActiveLeadStatusIDByName(name string) (int8, error)
	UpdateLeadReportURLAndStatus(leadID int64, reportURL string, statusID int8, userID int64) error
	// FindLeadsPendingFitCertification returns leads ready for fitness certificate PDF generation (see worker).
	FindLeadsPendingFitCertification(limit int, pendingLeadStatusID int8) ([]domain.Lead, error)
	// MarkFitCertificationGenerated sets certification flags and status if the lead still matches pending criteria; logs history. Returns whether a row was updated.
	MarkFitCertificationGenerated(leadID int64, userID int64, fromLeadStatusID, toLeadStatusID int8) (updated bool, err error)
}

type leadRepository struct {
	db *gorm.DB
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
	query := r.db.Model(&persistencemodels.Lead{})
	if filter.ClientID != nil {
		query = query.Where("ClientID = ?", *filter.ClientID)
	}
	if filter.StatusID != nil {
		query = query.Where("LeadStatusID = ?", *filter.StatusID)
	}
	if filter.PackageID != nil {
		query = query.Where("PackageID = ?", *filter.PackageID)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	sortColumn := mapLeadSortColumn(filter.SortBy)
	order := normalizeSortOrder(filter.SortOrder)
	offset := (filter.Page - 1) * filter.PageSize

	var leads []persistencemodels.Lead
	err := query.Order(sortColumn + " " + order).Limit(filter.PageSize).Offset(offset).Find(&leads).Error
	return mapLeadsToDomain(leads), total, err
}

func mapLeadSortColumn(sortBy string) string {
	switch sortBy {
	case "patientName":
		return "PatientName"
	case "clientId":
		return "ClientID"
	case "statusId":
		return "LeadStatusID"
	case "createdOn":
		return "CreatedOn"
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

func (r *leadRepository) UpdateStatusForIDs(leadIDs []int64, statusID int8, lastUpdatedBy int64) (int64, error) {
	result := r.db.Model(&persistencemodels.Lead{}).Where("LeadID IN ?", leadIDs).Updates(map[string]interface{}{
		"LeadStatusID":   statusID,
		"LastUpdatedBy":  lastUpdatedBy,
		"LastUpdatedOn":  time.Now(),
	})
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
	err := r.db.Where("LeadStatusID = ? AND IsFit = ? AND IsFitCertifiedGenerated = ?", pendingLeadStatusID, true, false).
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
				leadID, fromLeadStatusID, true, false).
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
