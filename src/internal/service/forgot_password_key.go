package service

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"b2b-diagnostic-aggregator/apis/internal/apperrors"
	"b2b-diagnostic-aggregator/apis/internal/domain"
	"b2b-diagnostic-aggregator/apis/internal/repository"
	"b2b-diagnostic-aggregator/apis/internal/timeutil"
	"b2b-diagnostic-aggregator/apis/pkg/utils"
)

const (
	forgotPasswordKeyTTL           = 5 * time.Minute
	clientCreatedForgotPasswordTTL = 5 * 24 * time.Hour
)

// insertForgotPasswordKey writes tbl_ForgotPassword the same way as POST /api/v1/login/forgot-password-key.
// userID is tbl_Login/tbl_ClientMaster id; userType is 1=employee, 2=client, 3=lab, 4=store.
func insertForgotPasswordKey(forgotRepo repository.ForgotPasswordRepository, userID int64, userType int, ttl time.Duration) (string, error) {
	if forgotRepo == nil {
		return "", fmt.Errorf("forgot password repository is not configured")
	}
	if userID == 0 {
		return "", apperrors.NewInternal("user id is required for forgot password key", nil)
	}
	if ttl <= 0 {
		ttl = forgotPasswordKeyTTL
	}

	now := time.Now().UTC()
	expiry := now.Add(ttl)

	payload := map[string]interface{}{
		"userId": userID, "userType": userType, "expiry": expiry.Format(time.RFC3339),
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", apperrors.NewInternal("Failed to generate reset key", err)
	}
	resetKey, err := utils.Encrypt(string(payloadBytes))
	if err != nil {
		return "", apperrors.NewInternal("Failed to generate reset key", err)
	}

	rec := &domain.ForgotPassword{
		UserID:            userID,
		UserType:          strconv.Itoa(userType),
		ForgetPasswordKey: resetKey,
		CreatedOn:         timeutil.FromTime(now),
		ExpiryTimestamp:   timeutil.FromTime(expiry),
		IsPasswordChanged: false,
	}
	if err := forgotRepo.Create(rec); err != nil {
		return "", err
	}
	return resetKey, nil
}
