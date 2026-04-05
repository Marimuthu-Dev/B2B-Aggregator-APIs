package repository

import (
	"time"

	persistencemodels "b2b-diagnostic-aggregator/apis/internal/persistence/models"

	"gorm.io/gorm"
)

// PackageLabMappingListFilter optional filters for listing package–lab mappings (nil = no filter).
// IsActive filters plm.IsActive; callers should default omitted query to true.
type PackageLabMappingListFilter struct {
	PackageID *int
	LabID     *int64
	IsActive  bool
}

// PackageLabMappingWithNames is one row from FindAllWithLabAndPackageNames (mapping + joined names).
type PackageLabMappingWithNames struct {
	PackageLabID  int       `gorm:"column:PackageLabID"`
	PackageID     int       `gorm:"column:PackageID"`
	LabID         int64     `gorm:"column:LabID"`
	Price         float64   `gorm:"column:Price"`
	IsActive      bool      `gorm:"column:IsActive"`
	CreatedBy     int64     `gorm:"column:CreatedBy"`
	CreatedOn     time.Time `gorm:"column:CreatedOn"`
	LastUpdatedBy int64     `gorm:"column:LastUpdatedBy"`
	LastUpdatedOn time.Time `gorm:"column:LastUpdatedOn"`
	LabName       string    `gorm:"column:LabName"`
	PackageName   string    `gorm:"column:PackageName"`
}

type PackageLabMappingRepository interface {
	Create(m *persistencemodels.PackageLabMapping) error
	FindByPackageAndLab(packageID int, labID int64) (*persistencemodels.PackageLabMapping, error)
	FindByID(id int) (*persistencemodels.PackageLabMapping, error)
	FindAll() ([]persistencemodels.PackageLabMapping, error)
	FindAllWithLabAndPackageNames(filter PackageLabMappingListFilter) ([]PackageLabMappingWithNames, error)
	FindByPackageID(packageID int) ([]persistencemodels.PackageLabMapping, error)
	Update(m *persistencemodels.PackageLabMapping) error
}

type packageLabMappingRepository struct {
	db *gorm.DB
}

func NewPackageLabMappingRepository(db *gorm.DB) PackageLabMappingRepository {
	return &packageLabMappingRepository{db: db}
}

func (r *packageLabMappingRepository) Create(m *persistencemodels.PackageLabMapping) error {
	return r.db.Create(m).Error
}

func (r *packageLabMappingRepository) FindByPackageAndLab(packageID int, labID int64) (*persistencemodels.PackageLabMapping, error) {
	var m persistencemodels.PackageLabMapping
	err := r.db.Where("PackageID = ? AND LabID = ? AND IsActive = ?", packageID, labID, true).First(&m).Error
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *packageLabMappingRepository) FindByID(id int) (*persistencemodels.PackageLabMapping, error) {
	var m persistencemodels.PackageLabMapping
	err := r.db.First(&m, id).Error
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *packageLabMappingRepository) FindAll() ([]persistencemodels.PackageLabMapping, error) {
	var list []persistencemodels.PackageLabMapping
	err := r.db.Find(&list).Error
	return list, err
}

func (r *packageLabMappingRepository) FindAllWithLabAndPackageNames(filter PackageLabMappingListFilter) ([]PackageLabMappingWithNames, error) {
	plm := (&persistencemodels.PackageLabMapping{}).TableName()
	lm := (&persistencemodels.Lab{}).TableName()
	pm := (&persistencemodels.Package{}).TableName()

	q := r.db.Table(plm+" AS plm").
		Select(`plm.PackageLabID, plm.PackageID, plm.LabID, plm.Price, plm.IsActive, plm.CreatedBy, plm.CreatedOn, plm.LastUpdatedBy, plm.LastUpdatedOn,
			lm.LabName, pm.PackageName`).
		Joins("LEFT JOIN "+lm+" AS lm ON lm.LabID = plm.LabID").
		Joins("LEFT JOIN "+pm+" AS pm ON pm.PackageID = plm.PackageID")
	if filter.PackageID != nil {
		q = q.Where("plm.PackageID = ?", *filter.PackageID)
	}
	if filter.LabID != nil {
		q = q.Where("plm.LabID = ?", *filter.LabID)
	}
	q = q.Where("plm.IsActive = ?", filter.IsActive)

	var rows []PackageLabMappingWithNames
	err := q.Scan(&rows).Error
	return rows, err
}

func (r *packageLabMappingRepository) FindByPackageID(packageID int) ([]persistencemodels.PackageLabMapping, error) {
	var list []persistencemodels.PackageLabMapping
	err := r.db.Where("PackageID = ?", packageID).Find(&list).Error
	return list, err
}

func (r *packageLabMappingRepository) Update(m *persistencemodels.PackageLabMapping) error {
	return r.db.Save(m).Error
}
