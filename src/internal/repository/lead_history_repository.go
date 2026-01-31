package repository

import (
	"b2b-diagnostic-aggregator/apis/internal/models"
	"gorm.io/gorm"
)

type LeadHistoryRepository interface {
	LogAction(history *models.LeadHistory) error
	BulkLogActions(histories []models.LeadHistory) error
	FindByLeadID(leadID int64) ([]models.LeadHistory, error)
}

type leadHistoryRepository struct {
	db *gorm.DB
}

func NewLeadHistoryRepository(db *gorm.DB) LeadHistoryRepository {
	return &leadHistoryRepository{db: db}
}

func (r *leadHistoryRepository) LogAction(history *models.LeadHistory) error {
	return r.db.Create(history).Error
}

func (r *leadHistoryRepository) BulkLogActions(histories []models.LeadHistory) error {
	return r.db.Create(&histories).Error
}

func (r *leadHistoryRepository) FindByLeadID(leadID int64) ([]models.LeadHistory, error) {
	var histories []models.LeadHistory
	err := r.db.Where("LeadID = ?", leadID).Order("CreatedOn DESC").Find(&histories).Error
	return histories, err
}
