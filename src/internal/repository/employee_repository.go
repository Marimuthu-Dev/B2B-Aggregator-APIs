package repository

import (
	"fmt"
	"strings"

	"b2b-diagnostic-aggregator/apis/internal/domain"
	persistencemodels "b2b-diagnostic-aggregator/apis/internal/persistence/models"
	"b2b-diagnostic-aggregator/apis/internal/timeutil"

	"gorm.io/gorm"
)

type EmployeeRepository interface {
	FindAll() ([]domain.Employee, error)
	FindByID(id int64) (*domain.Employee, error)
	FindByMobileNumber(mobileNumber string) (*domain.Employee, error)
	ExistsByID(id int64) (bool, error)
	ExistsByMobileNumber(mobileNumber string, excludeUID int64) (bool, error)
	Create(e *domain.Employee) error
	Update(e *domain.Employee) error
	Delete(id int64) error
}

type employeeRepository struct {
	db *gorm.DB
}

func NewEmployeeRepository(db *gorm.DB) EmployeeRepository {
	return &employeeRepository{db: db}
}

func (r *employeeRepository) FindAll() ([]domain.Employee, error) {
	var list []persistencemodels.Employee
	if err := r.db.Find(&list).Error; err != nil {
		return nil, err
	}
	return mapEmployeesToDomain(list), nil
}

func (r *employeeRepository) FindByID(id int64) (*domain.Employee, error) {
	var m persistencemodels.Employee
	if err := r.db.First(&m, id).Error; err != nil {
		return nil, err
	}
	d := mapEmployeeToDomain(m)
	return &d, nil
}

func (r *employeeRepository) FindByMobileNumber(mobileNumber string) (*domain.Employee, error) {
	var m persistencemodels.Employee
	if err := r.db.Where("MobileNumber = ?", mobileNumber).First(&m).Error; err != nil {
		return nil, err
	}
	d := mapEmployeeToDomain(m)

	// Console log: print DB record details (excluding mobile number)
	fmt.Printf("Employee DB record: UID=%d FullName=%s Address=%s CityID=%d StateID=%d Pincode=%s CompanyEmailID=%s Designation=%s Department=%s IsPriceViewAccess=%v CreatedBy=%d CreatedOn=%v LastUpdatedBy=%d LastUpdatedOn=%v\n",
		d.UID, d.FullName, d.Address, d.CityID, d.StateID, d.Pincode, d.CompanyEmailID, d.Designation, d.Department, d.IsPriceViewAccess, d.CreatedBy, d.CreatedOn.ToTime(), d.LastUpdatedBy, d.LastUpdatedOn.ToTime())

	return &d, nil
}

func (r *employeeRepository) ExistsByID(id int64) (bool, error) {
	var count int64
	if err := r.db.Model(&persistencemodels.Employee{}).Where("UID = ?", id).Limit(1).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *employeeRepository) ExistsByMobileNumber(mobileNumber string, excludeUID int64) (bool, error) {
	mobileNumber = strings.TrimSpace(mobileNumber)
	if mobileNumber == "" {
		return false, nil
	}
	q := r.db.Model(&persistencemodels.Employee{}).Where("MobileNumber = ?", mobileNumber)
	if excludeUID > 0 {
		q = q.Where("UID <> ?", excludeUID)
	}
	var count int64
	// Do not use Limit() with Count(): SQL Server rejects ORDER BY on an aggregate.
	if err := q.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *employeeRepository) Create(e *domain.Employee) error {
	p := mapEmployeeToPersistence(*e)
	if err := r.db.Create(&p).Error; err != nil {
		return err
	}
	*e = mapEmployeeToDomain(p)
	return nil
}

func (r *employeeRepository) Update(e *domain.Employee) error {
	p := mapEmployeeToPersistence(*e)
	if err := r.db.Save(&p).Error; err != nil {
		return err
	}
	*e = mapEmployeeToDomain(p)
	return nil
}

func (r *employeeRepository) Delete(id int64) error {
	return r.db.Delete(&persistencemodels.Employee{}, id).Error
}

func mapEmployeeToDomain(p persistencemodels.Employee) domain.Employee {
	return domain.Employee{
		UID:            p.UID,
		FullName:       p.FullName,
		Address:        p.Address,
		CityID:         p.CityID,
		StateID:        p.StateID,
		Pincode:        p.Pincode,
		MobileNumber:   p.MobileNumber,
		CompanyEmailID: p.CompanyEmailID,
		Designation:    p.Designation,
		Department:        p.Department,
		IsPriceViewAccess: p.IsPriceViewAccess,
		CreatedBy:         p.CreatedBy,
		CreatedOn:      timeutil.FromTime(p.CreatedOn),
		LastUpdatedBy:  p.LastUpdatedBy,
		LastUpdatedOn:  timeutil.FromTime(p.LastUpdatedOn),
	}
}

func mapEmployeeToPersistence(d domain.Employee) persistencemodels.Employee {
	return persistencemodels.Employee{
		UID:            d.UID,
		FullName:       d.FullName,
		Address:        d.Address,
		CityID:         d.CityID,
		StateID:        d.StateID,
		Pincode:        d.Pincode,
		MobileNumber:   d.MobileNumber,
		CompanyEmailID: d.CompanyEmailID,
		Designation:    d.Designation,
		Department:        d.Department,
		IsPriceViewAccess: d.IsPriceViewAccess,
		CreatedBy:         d.CreatedBy,
		CreatedOn:      d.CreatedOn.ToTime(),
		LastUpdatedBy:  d.LastUpdatedBy,
		LastUpdatedOn:  d.LastUpdatedOn.ToTime(),
	}
}

func mapEmployeesToDomain(list []persistencemodels.Employee) []domain.Employee {
	if len(list) == 0 {
		return nil
	}
	out := make([]domain.Employee, len(list))
	for i := range list {
		out[i] = mapEmployeeToDomain(list[i])
	}
	return out
}
