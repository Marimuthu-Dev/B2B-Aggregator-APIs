package repository

import (
	"time"

	"b2b-diagnostic-aggregator/apis/internal/domain"
	"b2b-diagnostic-aggregator/apis/internal/timeutil"
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
	err := r.db.Where("UserId = ? AND UserType = ? AND ExpiryTimestamp > ? AND IsPasswordChanged = ?",
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
	err := r.db.Where("UserId = ? AND UserType = ? AND ForgetPasswordKey = ? AND ExpiryTimestamp > ? AND IsPasswordChanged = ?",
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
		Where("UID = ?", record.UID).
		Updates(map[string]interface{}{
			"IsPasswordChanged": true,
			"PasswordUpdatedOn": now,
		}).Error
}

func mapForgotPasswordToPersistence(d domain.ForgotPassword) persistencemodels.ForgotPassword {
	return persistencemodels.ForgotPassword{
		UID:                 d.UID,
		UserID:              d.UserID,
		UserType:            d.UserType,
		ForgetPasswordKey:   d.ForgetPasswordKey,
		CreatedOn:           d.CreatedOn.ToTime(),
		ExpiryTimestamp:     d.ExpiryTimestamp.ToTime(),
		IsPasswordChanged:   d.IsPasswordChanged,
		PasswordUpdatedOn:   d.PasswordUpdatedOn,
	}
}

func mapForgotPasswordToDomain(p persistencemodels.ForgotPassword) domain.ForgotPassword {
	return domain.ForgotPassword{
		UID:                 p.UID,
		UserID:              p.UserID,
		UserType:            p.UserType,
		ForgetPasswordKey:   p.ForgetPasswordKey,
		CreatedOn:           timeutil.FromTime(p.CreatedOn),
		ExpiryTimestamp:     timeutil.FromTime(p.ExpiryTimestamp),
		IsPasswordChanged:   p.IsPasswordChanged,
		PasswordUpdatedOn:   p.PasswordUpdatedOn,
	}
}
