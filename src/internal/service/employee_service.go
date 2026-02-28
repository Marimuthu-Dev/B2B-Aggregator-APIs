package service

import (
	"errors"
	"time"

	"b2b-diagnostic-aggregator/apis/internal/apperrors"
	"b2b-diagnostic-aggregator/apis/internal/domain"
	"b2b-diagnostic-aggregator/apis/internal/dto"
	"b2b-diagnostic-aggregator/apis/internal/repository"

	"gorm.io/gorm"
)

type EmployeeService interface {
	GetAll() ([]domain.Employee, error)
	GetByID(id int64) (*domain.Employee, error)
	GetByContactNumber(contactNumber string) (*domain.Employee, error)
	Create(e *domain.Employee, createdBy int64) error
	Update(id int64, update *dto.EmployeeUpdateRequest, lastUpdatedBy int64) (*domain.Employee, error)
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

func (s *employeeService) Create(e *domain.Employee, createdBy int64) error {
	now := time.Now()
	e.CreatedBy = createdBy
	e.CreatedOn = now
	e.LastUpdatedBy = createdBy
	e.LastUpdatedOn = now
	return s.repo.Create(e)
}

func (s *employeeService) Update(id int64, update *dto.EmployeeUpdateRequest, lastUpdatedBy int64) (*domain.Employee, error) {
	existing, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.NewNotFound("Employee not found", err)
		}
		return nil, err
	}
	e := *existing
	if update.FullName != nil {
		e.FullName = *update.FullName
	}
	if update.Address != nil {
		e.Address = *update.Address
	}
	if update.CityID != nil {
		e.CityID = *update.CityID
	}
	if update.StateID != nil {
		e.StateID = *update.StateID
	}
	if update.Pincode != nil {
		e.Pincode = *update.Pincode
	}
	if update.MobileNumber != nil {
		e.MobileNumber = *update.MobileNumber
	}
	if update.CompanyEmailID != nil {
		e.CompanyEmailID = *update.CompanyEmailID
	}
	if update.Designation != nil {
		e.Designation = *update.Designation
	}
	if update.Department != nil {
		e.Department = *update.Department
	}
	e.UID = id
	e.LastUpdatedBy = lastUpdatedBy
	e.LastUpdatedOn = time.Now()
	if err := s.repo.Update(&e); err != nil {
		return nil, err
	}
	return &e, nil
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
