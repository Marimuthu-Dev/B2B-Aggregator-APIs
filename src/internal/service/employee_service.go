package service

import (
	"errors"
	"time"

	"b2b-diagnostic-aggregator/apis/internal/apperrors"
	"b2b-diagnostic-aggregator/apis/internal/domain"
	"b2b-diagnostic-aggregator/apis/internal/repository"

	"gorm.io/gorm"
)

type EmployeeService interface {
	GetAll() ([]domain.Employee, error)
	GetByID(id int64) (*domain.Employee, error)
	GetByContactNumber(contactNumber string) (*domain.Employee, error)
	Create(e *domain.Employee) error
	Update(id int64, e *domain.Employee) error
	Delete(id int64) error
}

type employeeService struct {
	repo repository.EmployeeRepository
}

func NewEmployeeService(repo repository.EmployeeRepository) EmployeeService {
	return &employeeService{repo: repo}
}

func (s *employeeService) GetAll() ([]domain.Employee, error) {
	return s.repo.FindAll()
}

func (s *employeeService) GetByID(id int64) (*domain.Employee, error) {
	emp, err := s.repo.FindByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperrors.NewNotFound("Employee not found", err)
	}
	return emp, err
}

func (s *employeeService) GetByContactNumber(contactNumber string) (*domain.Employee, error) {
	emp, err := s.repo.FindByMobileNumber(contactNumber)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperrors.NewNotFound("Employee not found", err)
	}
	return emp, err
}

func (s *employeeService) Create(e *domain.Employee) error {
	now := time.Now()
	e.CreatedOn = now
	e.LastUpdatedOn = now
	return s.repo.Create(e)
}

func (s *employeeService) Update(id int64, e *domain.Employee) error {
	exists, err := s.repo.ExistsByID(id)
	if err != nil {
		return err
	}
	if !exists {
		return apperrors.NewNotFound("Employee not found", gorm.ErrRecordNotFound)
	}
	e.UID = id
	e.LastUpdatedOn = time.Now()
	return s.repo.Update(e)
}

func (s *employeeService) Delete(id int64) error {
	exists, err := s.repo.ExistsByID(id)
	if err != nil {
		return err
	}
	if !exists {
		return apperrors.NewNotFound("Employee not found", gorm.ErrRecordNotFound)
	}
	return s.repo.Delete(id)
}
