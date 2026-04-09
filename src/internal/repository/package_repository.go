package repository

import (
	"sort"

	"b2b-diagnostic-aggregator/apis/internal/domain"
	persistencemodels "b2b-diagnostic-aggregator/apis/internal/persistence/models"

	"gorm.io/gorm"
)

type PackageRepository interface {
	FindAll() ([]domain.Package, error)
	List(filter PackageListFilter) ([]domain.Package, int64, error)
	FindByID(id int64) (*domain.Package, error)
	ExistsByID(id int64) (bool, error)
	Create(p *domain.Package) error
	Update(p *domain.Package) error
	Delete(id int64) error
	FindAllActive() ([]domain.Package, error)
	FindByName(name string) (*domain.Package, error)
	SearchByName(searchTerm string) ([]domain.Package, error)
	CreateWithTests(p *domain.Package, testIDs []int64) error
	FindAllPackageTestMappings() ([]persistencemodels.PackageTestMapping, error)
	FindPackagesByExactTestIds(testIDs []int64) ([]int64, error)
	UpdatePackageStatusCascade(packageID int64, isActive bool, lastUpdatedBy int64) (testCount, clientCount, labCount int, err error)
}

type packageRepository struct {
	db *gorm.DB
}

func NewPackageRepository(db *gorm.DB) PackageRepository {
	return &packageRepository{db: db}
}

func (r *packageRepository) FindAll() ([]domain.Package, error) {
	var packages []persistencemodels.Package
	err := r.db.Find(&packages).Error
	return mapPackagesToDomain(packages), err
}

func (r *packageRepository) List(filter PackageListFilter) ([]domain.Package, int64, error) {
	query := r.db.Model(&persistencemodels.Package{})
	if filter.IsActive != nil {
		query = query.Where("IsActive = ?", *filter.IsActive)
	}
	if filter.Search != "" {
		query = query.Where("PackageName LIKE ?", "%"+filter.Search+"%")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	sortColumn := mapPackageSortColumn(filter.SortBy)
	order := normalizeSortOrder(filter.SortOrder)
	offset := (filter.Page - 1) * filter.PageSize

	var packages []persistencemodels.Package
	err := query.Order(sortColumn + " " + order).Limit(filter.PageSize).Offset(offset).Find(&packages).Error
	return mapPackagesToDomain(packages), total, err
}

func mapPackageSortColumn(sortBy string) string {
	switch sortBy {
	case "name":
		return "PackageName"
	case "createdOn":
		return "CreatedOn"
	default:
		return "PackageID"
	}
}

func (r *packageRepository) FindByID(id int64) (*domain.Package, error) {
	var p persistencemodels.Package
	err := r.db.First(&p, id).Error
	if err != nil {
		return nil, err
	}
	domainPackage := mapPackageToDomain(p)
	return &domainPackage, nil
}

func (r *packageRepository) ExistsByID(id int64) (bool, error) {
	var count int64
	if err := r.db.Model(&persistencemodels.Package{}).Where("PackageID = ?", id).Limit(1).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *packageRepository) Create(p *domain.Package) error {
	persist := mapPackageToPersistence(*p)
	if err := r.db.Create(&persist).Error; err != nil {
		return err
	}
	*p = mapPackageToDomain(persist)
	return nil
}

func (r *packageRepository) Update(p *domain.Package) error {
	persist := mapPackageToPersistence(*p)
	if err := r.db.Save(&persist).Error; err != nil {
		return err
	}
	*p = mapPackageToDomain(persist)
	return nil
}

func (r *packageRepository) Delete(id int64) error {
	return r.db.Delete(&persistencemodels.Package{}, id).Error
}

func (r *packageRepository) FindAllActive() ([]domain.Package, error) {
	var packages []persistencemodels.Package
	err := r.db.Where("IsActive = ?", true).Find(&packages).Error
	return mapPackagesToDomain(packages), err
}

func (r *packageRepository) FindByName(name string) (*domain.Package, error) {
	var p persistencemodels.Package
	err := r.db.Where("PackageName = ?", name).First(&p).Error
	if err != nil {
		return nil, err
	}
	domainPackage := mapPackageToDomain(p)
	return &domainPackage, nil
}

func (r *packageRepository) SearchByName(searchTerm string) ([]domain.Package, error) {
	var packages []persistencemodels.Package
	err := r.db.Where("PackageName LIKE ?", "%"+searchTerm+"%").Find(&packages).Error
	return mapPackagesToDomain(packages), err
}

func (r *packageRepository) CreateWithTests(p *domain.Package, testIDs []int64) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Create package
		persist := mapPackageToPersistence(*p)
		if err := tx.Create(&persist).Error; err != nil {
			return err
		}
		*p = mapPackageToDomain(persist)

		// Create mappings
		mappings := make([]persistencemodels.PackageTestMapping, len(testIDs))
		for i, testID := range testIDs {
			mappings[i] = persistencemodels.PackageTestMapping{
				PackageID:     persist.PackageID,
				TestID:        testID,
				IsActive:      true,
				CreatedBy:     persist.CreatedBy,
				LastUpdatedBy: persist.LastUpdatedBy,
			}
		}

		if err := tx.Create(&mappings).Error; err != nil {
			return err
		}

		return nil
	})
}

func (r *packageRepository) FindAllPackageTestMappings() ([]persistencemodels.PackageTestMapping, error) {
	var out []persistencemodels.PackageTestMapping
	err := r.db.Find(&out).Error
	return out, err
}

func (r *packageRepository) FindPackagesByExactTestIds(testIDs []int64) ([]int64, error) {
	var mappings []persistencemodels.PackageTestMapping
	if err := r.db.Where("IsActive = ?", true).Find(&mappings).Error; err != nil {
		return nil, err
	}
	// Group by PackageID
	packageTests := make(map[int64][]int64)
	for _, m := range mappings {
		packageTests[m.PackageID] = append(packageTests[m.PackageID], m.TestID)
	}
	// Sort input for comparison
	sortedInput := make([]int64, len(testIDs))
	copy(sortedInput, testIDs)
	sort.Slice(sortedInput, func(i, j int) bool { return sortedInput[i] < sortedInput[j] })
	var match []int64
	for pkgID, ids := range packageTests {
		sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
		if len(ids) == len(sortedInput) && int64SlicesEqual(ids, sortedInput) {
			match = append(match, pkgID)
		}
	}
	return match, nil
}

func int64SlicesEqual(a, b []int64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func (r *packageRepository) UpdatePackageStatusCascade(packageID int64, isActive bool, lastUpdatedBy int64) (testCount, clientCount, labCount int, err error) {
	err = r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&persistencemodels.Package{}).Where("PackageID = ?", packageID).Updates(map[string]interface{}{
			"IsActive": isActive, "LastUpdatedBy": lastUpdatedBy,
		}).Error; err != nil {
			return err
		}
		res := tx.Model(&persistencemodels.PackageTestMapping{}).Where("PackageID = ?", packageID).Updates(map[string]interface{}{
			"IsActive": isActive, "LastUpdatedBy": lastUpdatedBy,
		})
		if res.Error != nil {
			return res.Error
		}
		testCount = int(res.RowsAffected)
		res = tx.Model(&persistencemodels.PackageClientMapping{}).Where("PackageID = ?", packageID).Updates(map[string]interface{}{
			"IsActive": isActive, "LastUpdatedBy": lastUpdatedBy,
		})
		if res.Error != nil {
			return res.Error
		}
		clientCount = int(res.RowsAffected)
		res = tx.Model(&persistencemodels.PackageLabMapping{}).Where("PackageID = ?", packageID).Updates(map[string]interface{}{
			"IsActive": isActive, "LastUpdatedBy": lastUpdatedBy,
		})
		if res.Error != nil {
			return res.Error
		}
		labCount = int(res.RowsAffected)
		return nil
	})
	return testCount, clientCount, labCount, err
}
