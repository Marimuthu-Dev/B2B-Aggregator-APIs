package repository

import (
	"strings"

	"b2b-diagnostic-aggregator/apis/internal/domain"
	persistencemodels "b2b-diagnostic-aggregator/apis/internal/persistence/models"

	"gorm.io/gorm"
)

type StoreListFilter struct {
	Page      int
	PageSize  int
	SortBy    string
	SortOrder string
	ClientID  *int64
	IsActive  *bool
	Search    string
}

type StoreRepository interface {
	Create(s *domain.Store) error
	CreateWithLogin(s *domain.Store, login *domain.Login) error
	Update(s *domain.Store) error
	FindByID(id int64) (*domain.Store, error)
	FindByContactNumber(contactNumber string) (*domain.Store, error)
	ExistsByContactNumber(contactNumber string, excludeStoreID int64) (bool, error)
	ExistsByEmailID(emailID string, excludeStoreID int64) (bool, error)
	List(filter StoreListFilter) ([]domain.Store, int64, error)
}

type storeRepository struct {
	db *gorm.DB
}

func NewStoreRepository(db *gorm.DB) StoreRepository {
	return &storeRepository{db: db}
}

func (r *storeRepository) Create(s *domain.Store) error {
	persist := mapStoreToPersistence(*s)
	if err := r.db.Create(&persist).Error; err != nil {
		return err
	}
	*s = mapStoreToDomain(persist)
	return nil
}

func (r *storeRepository) CreateWithLogin(s *domain.Store, login *domain.Login) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		persist := mapStoreToPersistence(*s)
		if err := tx.Create(&persist).Error; err != nil {
			return err
		}
		*s = mapStoreToDomain(persist)
		loginRow := persistencemodels.Login{
			UserID:        s.StoreID,
			Pwd:           login.Pwd,
			UserType:      login.UserType,
			CreatedOn:     s.CreatedOn.ToTime(),
			LastUpdatedOn: s.LastUpdatedOn.ToTime(),
		}
		if err := tx.Create(&loginRow).Error; err != nil {
			return err
		}
		login.UserID = s.StoreID
		login.RecordID = loginRow.RecordID
		return nil
	})
}

func (r *storeRepository) Update(s *domain.Store) error {
	persist := mapStoreToPersistence(*s)
	if err := r.db.Save(&persist).Error; err != nil {
		return err
	}
	*s = mapStoreToDomain(persist)
	return nil
}

func (r *storeRepository) FindByID(id int64) (*domain.Store, error) {
	var row persistencemodels.Store
	if err := r.db.First(&row, id).Error; err != nil {
		return nil, err
	}
	d := mapStoreToDomain(row)
	return &d, nil
}

func (r *storeRepository) FindByContactNumber(contactNumber string) (*domain.Store, error) {
	var row persistencemodels.Store
	err := r.db.Where("ContactNumber = ?", contactNumber).First(&row).Error
	if err != nil {
		return nil, err
	}
	d := mapStoreToDomain(row)
	return &d, nil
}

func (r *storeRepository) ExistsByContactNumber(contactNumber string, excludeStoreID int64) (bool, error) {
	q := r.db.Model(&persistencemodels.Store{}).Where("ContactNumber = ?", contactNumber)
	if excludeStoreID > 0 {
		q = q.Where("StoreID <> ?", excludeStoreID)
	}
	var count int64
	// Do not use Limit() with Count(): SQL Server rejects ORDER BY StoreID on an aggregate.
	if err := q.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *storeRepository) ExistsByEmailID(emailID string, excludeStoreID int64) (bool, error) {
	q := r.db.Model(&persistencemodels.Store{}).Where("EmailID = ?", emailID)
	if excludeStoreID > 0 {
		q = q.Where("StoreID <> ?", excludeStoreID)
	}
	var count int64
	if err := q.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *storeRepository) List(filter StoreListFilter) ([]domain.Store, int64, error) {
	query := r.db.Model(&persistencemodels.Store{})
	if filter.ClientID != nil {
		query = query.Where("ClientID = ?", *filter.ClientID)
	}
	if filter.IsActive != nil {
		query = query.Where("IsActive = ?", *filter.IsActive)
	}
	if trimmed := strings.TrimSpace(filter.Search); trimmed != "" {
		term := "%" + trimmed + "%"
		query = query.Where(
			"(StoreName LIKE ? OR ContactNumber LIKE ? OR EmailID LIKE ?)",
			term, term, term,
		)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	sortColumn := mapStoreSortColumn(filter.SortBy)
	order := normalizeSortOrder(filter.SortOrder)
	offset := (filter.Page - 1) * filter.PageSize

	var rows []persistencemodels.Store
	err := query.Order(sortColumn + " " + order).Limit(filter.PageSize).Offset(offset).Find(&rows).Error
	return mapStoresToDomain(rows), total, err
}

func mapStoreSortColumn(sortBy string) string {
	switch sortBy {
	case "name":
		return "StoreName"
	case "clientId":
		return "ClientID"
	case "createdOn":
		return "CreatedOn"
	default:
		return "StoreID"
	}
}
