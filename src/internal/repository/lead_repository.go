package repository

import (
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
		"LeadStatusID":  statusID,
		"LastUpdatedBy": lastUpdatedBy,
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
	err := r.db.Where("Emailid = ?", email).Find(&leads).Error
	return mapLeadsToDomain(leads), err
}
