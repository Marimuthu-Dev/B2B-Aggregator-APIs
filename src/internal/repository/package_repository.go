package repository

import (
	"b2b-diagnostic-aggregator/apis/internal/models"
	"gorm.io/gorm"
)

type PackageRepository interface {
	FindAll() ([]models.Package, error)
	FindByID(id int) (*models.Package, error)
	Create(p *models.Package) error
	Update(p *models.Package) error
	Delete(id int) error
	FindAllActive() ([]models.Package, error)
	FindByName(name string) (*models.Package, error)
	SearchByName(searchTerm string) ([]models.Package, error)
	CreateWithTests(p *models.Package, testIDs []int) error
}

type packageRepository struct {
	db *gorm.DB
}

func NewPackageRepository(db *gorm.DB) PackageRepository {
	return &packageRepository{db: db}
}

func (r *packageRepository) FindAll() ([]models.Package, error) {
	var packages []models.Package
	err := r.db.Find(&packages).Error
	return packages, err
}

func (r *packageRepository) FindByID(id int) (*models.Package, error) {
	var p models.Package
	err := r.db.First(&p, id).Error
	return &p, err
}

func (r *packageRepository) Create(p *models.Package) error {
	return r.db.Create(p).Error
}

func (r *packageRepository) Update(p *models.Package) error {
	return r.db.Save(p).Error
}

func (r *packageRepository) Delete(id int) error {
	return r.db.Delete(&models.Package{}, id).Error
}

func (r *packageRepository) FindAllActive() ([]models.Package, error) {
	var packages []models.Package
	err := r.db.Where("IsActive = ?", true).Find(&packages).Error
	return packages, err
}

func (r *packageRepository) FindByName(name string) (*models.Package, error) {
	var p models.Package
	err := r.db.Where("PackageName = ?", name).First(&p).Error
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *packageRepository) SearchByName(searchTerm string) ([]models.Package, error) {
	var packages []models.Package
	err := r.db.Where("PackageName LIKE ?", "%"+searchTerm+"%").Find(&packages).Error
	return packages, err
}

func (r *packageRepository) CreateWithTests(p *models.Package, testIDs []int) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Create package
		if err := tx.Create(p).Error; err != nil {
			return err
		}

		// Create mappings
		mappings := make([]models.PackageTestMapping, len(testIDs))
		for i, testID := range testIDs {
			mappings[i] = models.PackageTestMapping{
				PackageID:     p.PackageID,
				TestID:        testID,
				IsActive:      true,
				CreatedBy:     p.CreatedBy,
				LastUpdatedBy: p.LastUpdatedBy,
			}
		}

		if err := tx.Create(&mappings).Error; err != nil {
			return err
		}

		return nil
	})
}
