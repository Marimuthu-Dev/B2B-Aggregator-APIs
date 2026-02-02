package repository

import (
	"b2b-diagnostic-aggregator/apis/internal/domain"
	persistencemodels "b2b-diagnostic-aggregator/apis/internal/persistence/models"

	"gorm.io/gorm"
)

type LoginRepository interface {
	FindByUserID(userID int64) (*domain.Login, error)
	UpdatePassword(userID int64, newPassword string) error
}

type loginRepository struct {
	db *gorm.DB
}

func NewLoginRepository(db *gorm.DB) LoginRepository {
	return &loginRepository{db: db}
}

func (r *loginRepository) FindByUserID(userID int64) (*domain.Login, error) {
	var login persistencemodels.Login
	err := r.db.Where("UserID = ?", userID).First(&login).Error
	if err != nil {
		return nil, err
	}
	domainLogin := mapLoginToDomain(login)
	return &domainLogin, nil
}

func (r *loginRepository) UpdatePassword(userID int64, newPassword string) error {
	return r.db.Model(&persistencemodels.Login{}).Where("UserID = ?", userID).Update("Pwd", newPassword).Error
}
