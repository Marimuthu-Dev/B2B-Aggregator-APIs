package service

import (
	"errors"
	"time"

	"b2b-diagnostic-aggregator/apis/internal/apperrors"
	"b2b-diagnostic-aggregator/apis/internal/config"
	"b2b-diagnostic-aggregator/apis/internal/dto"
	"b2b-diagnostic-aggregator/apis/internal/repository"
	"b2b-diagnostic-aggregator/apis/pkg/utils"
)

type LoginService interface {
	Login(userID int64, password string) (*dto.LoginResponse, error)
}

type loginService struct {
	repo       repository.LoginRepository
	jwtSecret  string
	accessTTL  time.Duration
	refreshTTL time.Duration
}

func NewLoginService(repo repository.LoginRepository, jwtCfg config.JWTConfig) LoginService {
	accessTTL, err := time.ParseDuration(jwtCfg.ExpiresIn)
	if err != nil {
		accessTTL = 24 * time.Hour
	}

	refreshTTL, err := time.ParseDuration(jwtCfg.RefreshExpiresIn)
	if err != nil {
		refreshTTL = 7 * 24 * time.Hour
	}

	return &loginService{
		repo:       repo,
		jwtSecret:  jwtCfg.Secret,
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
	}
}

func (s *loginService) Login(userID int64, password string) (*dto.LoginResponse, error) {
	login, err := s.repo.FindByUserID(userID)
	if err != nil {
		return nil, apperrors.NewUnauthorized("Invalid user ID or password", err)
	}

	// Decrypt the stored password to compare
	// The Node.js logic seems to decrypt the stored password and compare with input plain text.
	// Or it might encrypt the input and compare. Let's assume stored is encrypted.
	decryptedStoredPwd, err := utils.Decrypt(login.Pwd)
	if err != nil {
		return nil, apperrors.NewInternal("Error validating credentials", err)
	}

	if decryptedStoredPwd != password {
		return nil, apperrors.NewUnauthorized("Invalid user ID or password", errors.New("password mismatch"))
	}

	// Generate JWT
	accessToken, refreshToken, err := utils.GenerateToken(login.UserID, login.UserType, s.jwtSecret, s.accessTTL, s.refreshTTL)
	if err != nil {
		return nil, err
	}

	return &dto.LoginResponse{
		Token:        accessToken,
		RefreshToken: refreshToken,
		User:         login, // In reality, we might fetch user profile here
	}, nil
}
