package service

import (
	"errors"

	"b2b-diagnostic-aggregator/apis/internal/apperrors"
	"b2b-diagnostic-aggregator/apis/internal/domain"
	"b2b-diagnostic-aggregator/apis/internal/repository"

	"gorm.io/gorm"
)

type TestService interface {
	GetAllTests() ([]domain.Test, error)
	GetActiveTests() ([]domain.Test, error)
	GetTestByID(id int) (*domain.Test, error)
}

type testService struct {
	repo repository.TestRepository
}

func NewTestService(repo repository.TestRepository) TestService {
	return &testService{repo: repo}
}

func (s *testService) GetAllTests() ([]domain.Test, error) {
	return s.repo.FindAll()
}

func (s *testService) GetActiveTests() ([]domain.Test, error) {
	return s.repo.FindAllActive()
}

func (s *testService) GetTestByID(id int) (*domain.Test, error) {
	test, err := s.repo.FindByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperrors.NewNotFound("Test not found", err)
	}
	return test, err
}
