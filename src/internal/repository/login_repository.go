package repository

import (
	"fmt"

	"b2b-diagnostic-aggregator/apis/internal/domain"
	persistencemodels "b2b-diagnostic-aggregator/apis/internal/persistence/models"

	"gorm.io/gorm"
)

type LoginRepository interface {
	FindByUserID(userID int64) (*domain.Login, error)
	Authenticate(userID int64, encryptedPassword, userType string) (bool, error)
	UpdatePassword(userID int64, newPassword string) error
	ChangePassword(userID int64, oldEncryptedPassword, newEncryptedPassword string) (int64, error)
}

type loginRepository struct {
	db *gorm.DB
}

func NewLoginRepository(db *gorm.DB) LoginRepository {
	return &loginRepository{db: db}
}

func (r *loginRepository) FindByUserID(userID int64) (*domain.Login, error) {
	fmt.Printf("[LOGIN] Repository.FindByUserID: entry userId=%d\n", userID)
	var login persistencemodels.Login
	err := r.db.Where("UserID = ?", userID).First(&login).Error
	if err != nil {
		fmt.Printf("[LOGIN] Repository.FindByUserID: query failed: %v\n", err)
		return nil, err
	}
	domainLogin := mapLoginToDomain(login)
	fmt.Printf("[LOGIN] Repository.FindByUserID: found UserID=%d UserType=%s\n", domainLogin.UserID, domainLogin.UserType)
	return &domainLogin, nil
}

func (r *loginRepository) Authenticate(userID int64, encryptedPassword, userType string) (bool, error) {
	fmt.Printf("[LOGIN] Repository.Authenticate: entry userId=%d userType=%s\n", userID, userType)
	var count int64
	err := r.db.Model(&persistencemodels.Login{}).
		Where("UserID = ? AND Pwd = ? AND UserType = ?", userID, encryptedPassword, userType).
		Count(&count).Error
	ok := count > 0
	fmt.Printf("[LOGIN] Repository.Authenticate: result ok=%v err=%v\n", ok, err)
	return ok, err
}

func (r *loginRepository) UpdatePassword(userID int64, newPassword string) error {
	return r.db.Model(&persistencemodels.Login{}).Where("UserID = ?", userID).Update("Pwd", newPassword).Error
}

func (r *loginRepository) ChangePassword(userID int64, oldEncryptedPassword, newEncryptedPassword string) (int64, error) {
	res := r.db.Model(&persistencemodels.Login{}).
		Where("UserID = ? AND Pwd = ?", userID, oldEncryptedPassword).
		Update("Pwd", newEncryptedPassword)
	return res.RowsAffected, res.Error
}
