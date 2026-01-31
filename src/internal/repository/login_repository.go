package repository

import (
	"b2b-diagnostic-aggregator/apis/internal/models"
	"gorm.io/gorm"
)

type LoginRepository interface {
	FindByUserID(userID int64) (*models.Login, error)
	UpdatePassword(userID int64, newPassword string) error
}

type loginRepository struct {
	db *gorm.DB
}

func NewLoginRepository(db *gorm.DB) LoginRepository {
	return &loginRepository{db: db}
}

func (r *loginRepository) FindByUserID(userID int64) (*models.Login, error) {
	var login models.Login
	err := r.db.Where("UserID = ?", userID).First(&login).Error
	if err != nil {
		return nil, err
	}
	return &login, nil
}

func (r *loginRepository) UpdatePassword(userID int64, newPassword string) error {
	return r.db.Model(&models.Login{}).Where("UserID = ?", userID).Update("Pwd", newPassword).Error
}
