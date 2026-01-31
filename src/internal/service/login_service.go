package service

import (
	"errors"

	"b2b-diagnostic-aggregator/apis/internal/models"
	"b2b-diagnostic-aggregator/apis/internal/repository"
	"b2b-diagnostic-aggregator/apis/pkg/utils"
)

type LoginService interface {
	Login(userID int64, password string) (*models.LoginResponse, error)
}

type loginService struct {
	repo repository.LoginRepository
}

func NewLoginService(repo repository.LoginRepository) LoginService {
	return &loginService{repo: repo}
}

func (s *loginService) Login(userID int64, password string) (*models.LoginResponse, error) {
	login, err := s.repo.FindByUserID(userID)
	if err != nil {
		return nil, errors.New("invalid user ID or password")
	}

	// Decrypt the stored password to compare
	// The Node.js logic seems to decrypt the stored password and compare with input plain text.
	// Or it might encrypt the input and compare. Let's assume stored is encrypted.
	decryptedStoredPwd, err := utils.Decrypt(login.Pwd)
	if err != nil {
		return nil, errors.New("error decrypting stored password")
	}

	if decryptedStoredPwd != password {
		return nil, errors.New("invalid user ID or password")
	}

	// Generate JWT
	accessToken, refreshToken, err := utils.GenerateToken(login.UserID, login.UserType)
	if err != nil {
		return nil, err
	}

	return &models.LoginResponse{
		Token:        accessToken,
		RefreshToken: refreshToken,
		User:         login, // In reality, we might fetch user profile here
	}, nil
}
