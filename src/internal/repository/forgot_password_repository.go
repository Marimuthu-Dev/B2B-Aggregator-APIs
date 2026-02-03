package repository

import (
	"time"

	"b2b-diagnostic-aggregator/apis/internal/domain"
	persistencemodels "b2b-diagnostic-aggregator/apis/internal/persistence/models"

	"gorm.io/gorm"
)

type ForgotPasswordRepository interface {
	Create(data *domain.ForgotPassword) error
	FindLatestValidKey(userID int64, userType string) (*domain.ForgotPassword, error)
	FindByKey(forgetPasswordKey string, userID int64, userType string) (*domain.ForgotPassword, error)
	MarkAsUsed(record *domain.ForgotPassword) error
}

type forgotPasswordRepository struct {
	db *gorm.DB
}

func NewForgotPasswordRepository(db *gorm.DB) ForgotPasswordRepository {
	return &forgotPasswordRepository{db: db}
}

func (r *forgotPasswordRepository) Create(data *domain.ForgotPassword) error {
	p := mapForgotPasswordToPersistence(*data)
	if err := r.db.Create(&p).Error; err != nil {
		return err
	}
	*data = mapForgotPasswordToDomain(p)
	return nil
}

func (r *forgotPasswordRepository) FindLatestValidKey(userID int64, userType string) (*domain.ForgotPassword, error) {
	var p persistencemodels.ForgotPassword
	now := time.Now().UTC()
	err := r.db.Where("UserID = ? AND UserType = ? AND ExpiryTimestamp > ? AND IsPasswordChanged = ?",
		userID, userType, now, false).
		Order("CreatedOn DESC").
		First(&p).Error
	if err != nil {
		return nil, err
	}
	d := mapForgotPasswordToDomain(p)
	return &d, nil
}

func (r *forgotPasswordRepository) FindByKey(forgetPasswordKey string, userID int64, userType string) (*domain.ForgotPassword, error) {
	var p persistencemodels.ForgotPassword
	now := time.Now().UTC()
	err := r.db.Where("UserID = ? AND UserType = ? AND ForgetPasswordKey = ? AND ExpiryTimestamp > ? AND IsPasswordChanged = ?",
		userID, userType, forgetPasswordKey, now, false).
		First(&p).Error
	if err != nil {
		return nil, err
	}
	d := mapForgotPasswordToDomain(p)
	return &d, nil
}

func (r *forgotPasswordRepository) MarkAsUsed(record *domain.ForgotPassword) error {
	now := time.Now().UTC()
	return r.db.Model(&persistencemodels.ForgotPassword{}).
		Where("Uid = ?", record.Uid).
		Updates(map[string]interface{}{
			"IsPasswordChanged":   true,
			"IsPasswordUpdatedOn": now,
		}).Error
}

func mapForgotPasswordToPersistence(d domain.ForgotPassword) persistencemodels.ForgotPassword {
	return persistencemodels.ForgotPassword{
		Uid:                 d.Uid,
		UserID:              d.UserID,
		UserType:            d.UserType,
		ForgetPasswordKey:   d.ForgetPasswordKey,
		CreatedOn:           d.CreatedOn,
		ExpiryTimestamp:     d.ExpiryTimestamp,
		IsPasswordChanged:   d.IsPasswordChanged,
		IsPasswordUpdatedOn: d.IsPasswordUpdatedOn,
	}
}

func mapForgotPasswordToDomain(p persistencemodels.ForgotPassword) domain.ForgotPassword {
	return domain.ForgotPassword{
		Uid:                 p.Uid,
		UserID:              p.UserID,
		UserType:            p.UserType,
		ForgetPasswordKey:   p.ForgetPasswordKey,
		CreatedOn:           p.CreatedOn,
		ExpiryTimestamp:     p.ExpiryTimestamp,
		IsPasswordChanged:   p.IsPasswordChanged,
		IsPasswordUpdatedOn: p.IsPasswordUpdatedOn,
	}
}
