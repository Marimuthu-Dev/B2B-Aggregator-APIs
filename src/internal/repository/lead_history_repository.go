package repository

import (
	"b2b-diagnostic-aggregator/apis/internal/domain"
	persistencemodels "b2b-diagnostic-aggregator/apis/internal/persistence/models"

	"gorm.io/gorm"
)

type LeadHistoryRepository interface {
	LogAction(history *domain.LeadHistory) error
	BulkLogActions(histories []domain.LeadHistory) error
	FindByLeadID(leadID int64) ([]domain.LeadHistory, error)
}

type leadHistoryRepository struct {
	db *gorm.DB
}

func NewLeadHistoryRepository(db *gorm.DB) LeadHistoryRepository {
	return &leadHistoryRepository{db: db}
}

func (r *leadHistoryRepository) LogAction(history *domain.LeadHistory) error {
	persist := mapLeadHistoryToPersistence(*history)
	if err := r.db.Create(&persist).Error; err != nil {
		return err
	}
	*history = mapLeadHistoryToDomain(persist)
	return nil
}

func (r *leadHistoryRepository) BulkLogActions(histories []domain.LeadHistory) error {
	persist := mapLeadHistoriesToPersistence(histories)
	return r.db.Create(&persist).Error
}

func (r *leadHistoryRepository) FindByLeadID(leadID int64) ([]domain.LeadHistory, error) {
	var histories []persistencemodels.LeadHistory
	err := r.db.Where("LeadID = ?", leadID).Order("CreatedOn DESC").Find(&histories).Error
	return mapLeadHistoriesToDomain(histories), err
}
