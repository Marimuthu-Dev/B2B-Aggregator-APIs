package repository

import (
	"b2b-diagnostic-aggregator/apis/internal/models"
	"gorm.io/gorm"
)

type LeadRepository interface {
	FindAll() ([]models.Lead, error)
	FindByID(id int64) (*models.Lead, error)
	Create(l *models.Lead) error
	Update(l *models.Lead) error
	Delete(id int64) error
	FindByClientID(clientID int64) ([]models.Lead, error)
	FindByStatus(statusID int8) ([]models.Lead, error)
	FindByPackage(packageID int) ([]models.Lead, error)
	FindByPatientID(patientID string) (*models.Lead, error)
	FindByContactNumber(contactNumber string) ([]models.Lead, error)
	FindByEmail(email string) ([]models.Lead, error)
}

type leadRepository struct {
	db *gorm.DB
}

func NewLeadRepository(db *gorm.DB) LeadRepository {
	return &leadRepository{db: db}
}

func (r *leadRepository) FindAll() ([]models.Lead, error) {
	var leads []models.Lead
	err := r.db.Find(&leads).Error
	return leads, err
}

func (r *leadRepository) FindByID(id int64) (*models.Lead, error) {
	var l models.Lead
	err := r.db.First(&l, id).Error
	if err != nil {
		return nil, err
	}
	return &l, nil
}

func (r *leadRepository) Create(l *models.Lead) error {
	return r.db.Create(l).Error
}

func (r *leadRepository) Update(l *models.Lead) error {
	return r.db.Save(l).Error
}

func (r *leadRepository) Delete(id int64) error {
	return r.db.Delete(&models.Lead{}, id).Error
}

func (r *leadRepository) FindByClientID(clientID int64) ([]models.Lead, error) {
	var leads []models.Lead
	err := r.db.Where("ClientID = ?", clientID).Find(&leads).Error
	return leads, err
}

func (r *leadRepository) FindByStatus(statusID int8) ([]models.Lead, error) {
	var leads []models.Lead
	err := r.db.Where("LeadStatusID = ?", statusID).Find(&leads).Error
	return leads, err
}

func (r *leadRepository) FindByPackage(packageID int) ([]models.Lead, error) {
	var leads []models.Lead
	err := r.db.Where("PackageID = ?", packageID).Find(&leads).Error
	return leads, err
}

func (r *leadRepository) FindByPatientID(patientID string) (*models.Lead, error) {
	var l models.Lead
	err := r.db.Where("PatientID = ?", patientID).First(&l).Error
	if err != nil {
		return nil, err
	}
	return &l, nil
}

func (r *leadRepository) FindByContactNumber(contactNumber string) ([]models.Lead, error) {
	var leads []models.Lead
	err := r.db.Where("ContactNumber = ?", contactNumber).Find(&leads).Error
	return leads, err
}

func (r *leadRepository) FindByEmail(email string) ([]models.Lead, error) {
	var leads []models.Lead
	err := r.db.Where("Emailid = ?", email).Find(&leads).Error
	return leads, err
}
