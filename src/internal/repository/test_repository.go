package repository

import (
	"b2b-diagnostic-aggregator/apis/internal/domain"
	persistencemodels "b2b-diagnostic-aggregator/apis/internal/persistence/models"

	"gorm.io/gorm"
)

type TestRepository interface {
	FindAll() ([]domain.Test, error)
	FindAllActive() ([]domain.Test, error)
	FindByID(id int) (*domain.Test, error)
	FindByIDs(ids []int) ([]domain.Test, error)
	ExistsByID(id int) (bool, error)
}

type testRepository struct {
	db *gorm.DB
}

func NewTestRepository(db *gorm.DB) TestRepository {
	return &testRepository{db: db}
}

func (r *testRepository) FindAll() ([]domain.Test, error) {
	var tests []persistencemodels.Test
	err := r.db.Find(&tests).Error
	return mapTestsToDomain(tests), err
}

func (r *testRepository) FindAllActive() ([]domain.Test, error) {
	var tests []persistencemodels.Test
	err := r.db.Where("IsActive = ?", true).Find(&tests).Error
	return mapTestsToDomain(tests), err
}

func (r *testRepository) FindByID(id int) (*domain.Test, error) {
	var t persistencemodels.Test
	err := r.db.First(&t, id).Error
	if err != nil {
		return nil, err
	}
	domainTest := mapTestToDomain(t)
	return &domainTest, nil
}

func (r *testRepository) FindByIDs(ids []int) ([]domain.Test, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var tests []persistencemodels.Test
	if err := r.db.Where("TestID IN ?", ids).Find(&tests).Error; err != nil {
		return nil, err
	}
	return mapTestsToDomain(tests), nil
}

func (r *testRepository) ExistsByID(id int) (bool, error) {
	var count int64
	if err := r.db.Model(&persistencemodels.Test{}).Where("TestID = ?", id).Limit(1).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func mapTestToDomain(p persistencemodels.Test) domain.Test {
	return domain.Test{
		TestID:        p.TestID,
		TestName:      p.TestName,
		Category:      p.Category,
		IsActive:      p.IsActive,
		CreatedBy:     p.CreatedBy,
		CreatedOn:     p.CreatedOn,
		LastUpdatedBy: p.LastUpdatedBy,
		LastUpdatedOn: p.LastUpdatedOn,
	}
}

func mapTestsToDomain(tests []persistencemodels.Test) []domain.Test {
	if len(tests) == 0 {
		return nil
	}
	mapped := make([]domain.Test, len(tests))
	for i, t := range tests {
		mapped[i] = mapTestToDomain(t)
	}
	return mapped
}
