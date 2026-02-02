package repository

import "gorm.io/gorm"

type LeadUnitOfWork interface {
	WithinTransaction(func(LeadRepository, LeadHistoryRepository) error) error
}

type leadUnitOfWork struct {
	db *gorm.DB
}

func NewLeadUnitOfWork(db *gorm.DB) LeadUnitOfWork {
	return &leadUnitOfWork{db: db}
}

func (u *leadUnitOfWork) WithinTransaction(fn func(LeadRepository, LeadHistoryRepository) error) error {
	return u.db.Transaction(func(tx *gorm.DB) error {
		leadRepo := NewLeadRepository(tx)
		historyRepo := NewLeadHistoryRepository(tx)
		return fn(leadRepo, historyRepo)
	})
}
