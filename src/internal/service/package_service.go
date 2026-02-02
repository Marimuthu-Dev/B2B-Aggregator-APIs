package service

import (
	"errors"

	"b2b-diagnostic-aggregator/apis/internal/apperrors"
	"b2b-diagnostic-aggregator/apis/internal/domain"
	"b2b-diagnostic-aggregator/apis/internal/repository"

	"gorm.io/gorm"
)

type PackageService interface {
	ListPackages(filter repository.PackageListFilter) ([]domain.Package, int64, error)
	GetPackageByID(id int) (*domain.Package, error)
	CreatePackage(p *domain.Package) error
	UpdatePackage(p *domain.Package) error
	DeletePackage(id int) error
	GetActivePackages() ([]domain.Package, error)
	CreatePackageWithTests(p *domain.Package, testIDs []int) error
}

type packageService struct {
	repo repository.PackageRepository
}

func NewPackageService(repo repository.PackageRepository) PackageService {
	return &packageService{repo: repo}
}

func (s *packageService) ListPackages(filter repository.PackageListFilter) ([]domain.Package, int64, error) {
	return s.repo.List(filter)
}

func (s *packageService) GetPackageByID(id int) (*domain.Package, error) {
	pkg, err := s.repo.FindByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperrors.NewNotFound("Package not found", err)
	}
	return pkg, err
}

func (s *packageService) CreatePackage(p *domain.Package) error {
	return s.repo.Create(p)
}

func (s *packageService) UpdatePackage(p *domain.Package) error {
	exists, err := s.repo.ExistsByID(p.PackageID)
	if err != nil {
		return err
	}
	if !exists {
		return apperrors.NewNotFound("Package not found", gorm.ErrRecordNotFound)
	}
	return s.repo.Update(p)
}

func (s *packageService) DeletePackage(id int) error {
	exists, err := s.repo.ExistsByID(id)
	if err != nil {
		return err
	}
	if !exists {
		return apperrors.NewNotFound("Package not found", gorm.ErrRecordNotFound)
	}
	return s.repo.Delete(id)
}

func (s *packageService) GetActivePackages() ([]domain.Package, error) {
	return s.repo.FindAllActive()
}

func (s *packageService) CreatePackageWithTests(p *domain.Package, testIDs []int) error {
	return s.repo.CreateWithTests(p, testIDs)
}
