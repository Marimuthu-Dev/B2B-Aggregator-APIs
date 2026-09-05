package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"b2b-diagnostic-aggregator/apis/internal/apperrors"
	"b2b-diagnostic-aggregator/apis/internal/config"
	"b2b-diagnostic-aggregator/apis/internal/domain"
	"b2b-diagnostic-aggregator/apis/internal/dto"
	"b2b-diagnostic-aggregator/apis/internal/repository"
	"b2b-diagnostic-aggregator/apis/internal/timeutil"

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
	repo       repository.EmployeeRepository
	emails     *repository.EmailOutboxRepository
	forgotRepo repository.ForgotPasswordRepository
	emailCfg   config.OutboundEmailConfig
	portalURL  string
}

func NewEmployeeService(
	repo repository.EmployeeRepository,
	emails *repository.EmailOutboxRepository,
	forgotRepo repository.ForgotPasswordRepository,
	emailCfg config.OutboundEmailConfig,
	employeePortalURL string,
) EmployeeService {
	return &employeeService{
		repo:       repo,
		emails:     emails,
		forgotRepo: forgotRepo,
		emailCfg:   emailCfg,
		portalURL:  employeePortalURL,
	}
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
	if err := s.ensureEmployeeMobileUnique(e.MobileNumber, 0); err != nil {
		return err
	}
	now := time.Now()
	e.CreatedBy = createdBy
	e.CreatedOn = timeutil.FromTime(now)
	e.LastUpdatedBy = createdBy
	e.LastUpdatedOn = timeutil.FromTime(now)
	// Set default IsActive to true if not provided
	e.IsActive = true
	if err := s.repo.Create(e); err != nil {
		return err
	}
	s.queueEmployeeCreatedEmail(context.Background(), e)
	return nil
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
	if update.IsActive != nil {
		e.IsActive = *update.IsActive
	}
	if err := s.ensureEmployeeMobileUnique(e.MobileNumber, id); err != nil {
		return nil, err
	}
	e.UID = id
	e.LastUpdatedBy = lastUpdatedBy
	e.LastUpdatedOn = timeutil.FromTime(time.Now())
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

func (s *employeeService) ensureEmployeeMobileUnique(mobile string, excludeUID int64) error {
	mobile = strings.TrimSpace(mobile)
	if mobile == "" {
		return nil
	}
	taken, err := s.repo.ExistsByMobileNumber(mobile, excludeUID)
	if err != nil {
		return err
	}
	if taken {
		return apperrors.NewBadRequest("MobileNumber mobile already exists with system", nil)
	}
	return nil
}
