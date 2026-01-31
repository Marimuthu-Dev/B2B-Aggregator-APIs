package service

import (
	"b2b-diagnostic-aggregator/apis/internal/models"
	"b2b-diagnostic-aggregator/apis/internal/repository"
)

type PackageService interface {
	GetAllPackages() ([]models.Package, error)
	GetPackageByID(id int) (*models.Package, error)
	CreatePackage(p *models.Package) error
	UpdatePackage(p *models.Package) error
	DeletePackage(id int) error
	GetActivePackages() ([]models.Package, error)
	CreatePackageWithTests(p *models.Package, testIDs []int) error
}

type packageService struct {
	repo repository.PackageRepository
}

func NewPackageService(repo repository.PackageRepository) PackageService {
	return &packageService{repo: repo}
}

func (s *packageService) GetAllPackages() ([]models.Package, error) {
	return s.repo.FindAll()
}

func (s *packageService) GetPackageByID(id int) (*models.Package, error) {
	return s.repo.FindByID(id)
}

func (s *packageService) CreatePackage(p *models.Package) error {
	return s.repo.Create(p)
}

func (s *packageService) UpdatePackage(p *models.Package) error {
	return s.repo.Update(p)
}

func (s *packageService) DeletePackage(id int) error {
	return s.repo.Delete(id)
}

func (s *packageService) GetActivePackages() ([]models.Package, error) {
	return s.repo.FindAllActive()
}

func (s *packageService) CreatePackageWithTests(p *models.Package, testIDs []int) error {
	return s.repo.CreateWithTests(p, testIDs)
}
