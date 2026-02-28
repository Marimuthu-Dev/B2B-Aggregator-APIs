package service

import (
	"errors"
	"time"

	"b2b-diagnostic-aggregator/apis/internal/apperrors"
	"b2b-diagnostic-aggregator/apis/internal/domain"
	"b2b-diagnostic-aggregator/apis/internal/repository"
	persistencemodels "b2b-diagnostic-aggregator/apis/internal/persistence/models"

	"gorm.io/gorm"
)

type PackageService interface {
	ListPackages(filter repository.PackageListFilter) ([]domain.Package, int64, error)
	GetPackageByID(id int) (*domain.Package, error)
	CreatePackage(p *domain.Package, createdBy int64) error
	UpdatePackage(p *domain.Package, lastUpdatedBy int64) error
	DeletePackage(id int) error
	GetActivePackages() ([]domain.Package, error)
	CreatePackageWithTests(p *domain.Package, testIDs []int, createdBy int64) (*CreatePackageWithTestsResult, error)
	GetAllPackagesWithTestsDetails() ([]domain.PackageWithTestsDetail, error)
	UpdatePackageStatus(packageID int, isActive bool, lastUpdatedBy int64) (*UpdatePackageStatusResult, error)
	CreatePackageClientMapping(packageID int, clientID int64, price float64, createdBy, lastUpdatedBy int64) (*PackageClientMappingResult, error)
	GetAllPackageClientMappings() ([]domain.PackageClientMappingView, error)
	UpdatePackageClientMappingStatus(id int, isActive bool, lastUpdatedBy int64) (*PackageClientMappingUpdateResult, error)
	CreatePackageLabMapping(packageID int, labID int64, price float64, createdBy, lastUpdatedBy int64) (*PackageLabMappingResult, error)
	GetAllPackageLabMappings() ([]domain.PackageLabMappingView, error)
	UpdatePackageLabMappingStatus(id int, isActive bool, lastUpdatedBy int64) (*PackageLabMappingUpdateResult, error)
}

type CreatePackageWithTestsResult struct {
	RetVal          int
	Package         *domain.Package
	TestIDs         []int
	Message         string
}

type UpdatePackageStatusResult struct {
	Package                    *domain.Package
	UpdatedTestMappingsCount   int
	UpdatedClientMappingsCount int
	UpdatedLabMappingsCount    int
	TotalMappingsUpdated       int
}

type PackageClientMappingResult struct {
	RetVal  int
	Mapping *domain.PackageClientMappingView
	Message string
}

type PackageClientMappingUpdateResult struct {
	RetVal  int
	Mapping *domain.PackageClientMappingView
	Message string
}

type PackageLabMappingResult struct {
	RetVal  int
	Mapping *domain.PackageLabMappingView
	Message string
}

type PackageLabMappingUpdateResult struct {
	RetVal  int
	Mapping *domain.PackageLabMappingView
	Message string
}

type packageService struct {
	repo        repository.PackageRepository
	testRepo    repository.TestRepository
	clientRepo  repository.ClientRepository
	labRepo     repository.LabRepository
	clientMapRepo repository.PackageClientMappingRepository
	labMapRepo  repository.PackageLabMappingRepository
}

func NewPackageService(
	repo repository.PackageRepository,
	testRepo repository.TestRepository,
	clientMapRepo repository.PackageClientMappingRepository,
	labMapRepo repository.PackageLabMappingRepository,
	clientRepo repository.ClientRepository,
	labRepo repository.LabRepository,
) PackageService {
	return &packageService{
		repo:         repo,
		testRepo:     testRepo,
		clientRepo:   clientRepo,
		labRepo:      labRepo,
		clientMapRepo: clientMapRepo,
		labMapRepo:   labMapRepo,
	}
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

func (s *packageService) CreatePackage(p *domain.Package, createdBy int64) error {
	now := time.Now()
	p.CreatedBy = createdBy
	p.CreatedOn = now
	p.LastUpdatedBy = createdBy
	p.LastUpdatedOn = now
	return s.repo.Create(p)
}

func (s *packageService) UpdatePackage(p *domain.Package, lastUpdatedBy int64) error {
	existing, err := s.repo.FindByID(p.PackageID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperrors.NewNotFound("Package not found", err)
		}
		return err
	}
	p.CreatedBy = existing.CreatedBy
	p.CreatedOn = existing.CreatedOn
	p.LastUpdatedBy = lastUpdatedBy
	p.LastUpdatedOn = time.Now()
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

func (s *packageService) CreatePackageWithTests(p *domain.Package, testIDs []int, createdBy int64) (*CreatePackageWithTestsResult, error) {
	if len(testIDs) == 0 {
		return nil, apperrors.NewBadRequest("TestIDs is required and must be a non-empty array", nil)
	}
	// Validate all test IDs exist
	found, err := s.testRepo.FindByIDs(testIDs)
	if err != nil {
		return nil, err
	}
	if len(found) != len(testIDs) {
		return nil, apperrors.NewNotFound("One or more test IDs are invalid", nil)
	}
	// Dedupe and sort for comparison
	unique := uniqueInts(testIDs)
	matching, err := s.repo.FindPackagesByExactTestIds(unique)
	if err != nil {
		return nil, err
	}
	if len(matching) > 0 {
		existing, _ := s.repo.FindByID(matching[0])
		if existing != nil {
			return &CreatePackageWithTestsResult{
				RetVal:  2,
				Package: existing,
				TestIDs: unique,
				Message: "Package is already created with these tests",
			}, nil
		}
	}
	now := time.Now()
	p.CreatedBy = createdBy
	p.CreatedOn = now
	p.LastUpdatedBy = createdBy
	p.LastUpdatedOn = now
	if err := s.repo.CreateWithTests(p, unique); err != nil {
		return nil, err
	}
	return &CreatePackageWithTestsResult{
		RetVal:  1,
		Package: p,
		TestIDs: unique,
		Message: "Package created successfully with test mappings",
	}, nil
}

func uniqueInts(a []int) []int {
	seen := make(map[int]struct{})
	var out []int
	for _, v := range a {
		if _, ok := seen[v]; !ok {
			seen[v] = struct{}{}
			out = append(out, v)
		}
	}
	return out
}

func (s *packageService) GetAllPackagesWithTestsDetails() ([]domain.PackageWithTestsDetail, error) {
	packages, err := s.repo.FindAll()
	if err != nil {
		return nil, err
	}
	mappings, err := s.repo.FindAllPackageTestMappings()
	if err != nil {
		return nil, err
	}
	tests, err := s.testRepo.FindAll()
	if err != nil {
		return nil, err
	}
	testMap := make(map[int]domain.TestInPackage)
	for _, t := range tests {
		testMap[t.TestID] = domain.TestInPackage{
			TestID: t.TestID, TestName: t.TestName, Category: t.Category, IsActive: t.IsActive,
		}
	}
	pkgTestIDs := make(map[int][]int)
	for _, m := range mappings {
		pkgTestIDs[m.PackageID] = append(pkgTestIDs[m.PackageID], m.TestID)
	}
	var result []domain.PackageWithTestsDetail
	for _, p := range packages {
		ids := pkgTestIDs[p.PackageID]
		details := make([]domain.TestInPackage, 0, len(ids))
		for _, id := range ids {
			if d, ok := testMap[id]; ok {
				details = append(details, d)
			}
		}
		result = append(result, domain.PackageWithTestsDetail{
			PackageDetails: p,
			TestIDs:       ids,
			TestCount:     len(ids),
			TestDetails:   details,
		})
	}
	return result, nil
}

func (s *packageService) UpdatePackageStatus(packageID int, isActive bool, lastUpdatedBy int64) (*UpdatePackageStatusResult, error) {
	exists, err := s.repo.ExistsByID(packageID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, apperrors.NewNotFound("Package not found", gorm.ErrRecordNotFound)
	}
	testCount, clientCount, labCount, err := s.repo.UpdatePackageStatusCascade(packageID, isActive, lastUpdatedBy)
	if err != nil {
		return nil, err
	}
	pkg, _ := s.repo.FindByID(packageID)
	return &UpdatePackageStatusResult{
		Package:                    pkg,
		UpdatedTestMappingsCount:   testCount,
		UpdatedClientMappingsCount: clientCount,
		UpdatedLabMappingsCount:    labCount,
		TotalMappingsUpdated:       testCount + clientCount + labCount,
	}, nil
}

func (s *packageService) CreatePackageClientMapping(packageID int, clientID int64, price float64, createdBy, lastUpdatedBy int64) (*PackageClientMappingResult, error) {
	if _, err := s.repo.FindByID(packageID); err != nil {
		return nil, apperrors.NewNotFound("Package not found", err)
	}
	if _, err := s.clientRepo.FindByID(clientID); err != nil {
		return nil, apperrors.NewNotFound("Client not found", err)
	}
	existing, _ := s.clientMapRepo.FindByPackageAndClient(packageID, clientID)
	if existing != nil {
		v := mappingToClientView(existing, "", "")
		pkg, _ := s.repo.FindByID(packageID)
		cli, _ := s.clientRepo.FindByID(clientID)
		if pkg != nil {
			v.PackageName = pkg.PackageName
		}
		if cli != nil {
			v.ClientName = cli.ClientName
		}
		return &PackageClientMappingResult{
			RetVal:  2,
			Mapping: v,
			Message: "Package-Client mapping already exists",
		}, nil
	}
	m := &persistencemodels.PackageClientMapping{
		PackageID: packageID, ClientID: clientID, Price: price,
		IsActive: true, CreatedBy: createdBy, LastUpdatedBy: lastUpdatedBy,
	}
	if err := s.clientMapRepo.Create(m); err != nil {
		return nil, err
	}
	v := mappingToClientView(m, "", "")
	pkg, _ := s.repo.FindByID(packageID)
	cli, _ := s.clientRepo.FindByID(clientID)
	if pkg != nil {
		v.PackageName = pkg.PackageName
	}
	if cli != nil {
		v.ClientName = cli.ClientName
	}
	return &PackageClientMappingResult{RetVal: 1, Mapping: v, Message: "Package-Client mapping created successfully"}, nil
}

func mappingToClientView(m *persistencemodels.PackageClientMapping, pkgName, clientName string) *domain.PackageClientMappingView {
	if m == nil {
		return nil
	}
	return &domain.PackageClientMappingView{
		PackageClientID: m.PackageClientID,
		PackageID:       m.PackageID,
		ClientID:        m.ClientID,
		Price:           m.Price,
		IsActive:        m.IsActive,
		CreatedBy:       m.CreatedBy,
		CreatedOn:       m.CreatedOn,
		LastUpdatedBy:   m.LastUpdatedBy,
		LastUpdatedOn:   m.LastUpdatedOn,
		PackageName:     pkgName,
		ClientName:      clientName,
	}
}

func (s *packageService) GetAllPackageClientMappings() ([]domain.PackageClientMappingView, error) {
	list, err := s.clientMapRepo.FindAll()
	if err != nil {
		return nil, err
	}
	packages, _ := s.repo.FindAll()
	clients, _ := s.clientRepo.FindAll()
	pkgMap := make(map[int]string)
	for _, p := range packages {
		pkgMap[p.PackageID] = p.PackageName
	}
	cliMap := make(map[int64]string)
	for _, c := range clients {
		cliMap[c.ClientID] = c.ClientName
	}
	var out []domain.PackageClientMappingView
	for i := range list {
		v := mappingToClientView(&list[i], pkgMap[list[i].PackageID], cliMap[list[i].ClientID])
		out = append(out, *v)
	}
	return out, nil
}

func (s *packageService) UpdatePackageClientMappingStatus(id int, isActive bool, lastUpdatedBy int64) (*PackageClientMappingUpdateResult, error) {
	m, err := s.clientMapRepo.FindByID(id)
	if err != nil || m == nil {
		return nil, apperrors.NewNotFound("Package-Client mapping not found", gorm.ErrRecordNotFound)
	}
	if isActive {
		pkg, _ := s.repo.FindByID(m.PackageID)
		if pkg != nil && !pkg.IsActive {
			v := mappingToClientView(m, "", "")
			pkg2, _ := s.repo.FindByID(m.PackageID)
			cli, _ := s.clientRepo.FindByID(m.ClientID)
			if pkg2 != nil {
				v.PackageName = pkg2.PackageName
			}
			if cli != nil {
				v.ClientName = cli.ClientName
			}
			return &PackageClientMappingUpdateResult{
				RetVal:  2,
				Mapping: v,
				Message: "Cannot activate client mapping because the package is inactive",
			}, nil
		}
	}
	m.IsActive = isActive
	m.LastUpdatedBy = lastUpdatedBy
	if err := s.clientMapRepo.Update(m); err != nil {
		return nil, err
	}
	v := mappingToClientView(m, "", "")
	pkg, _ := s.repo.FindByID(m.PackageID)
	cli, _ := s.clientRepo.FindByID(m.ClientID)
	if pkg != nil {
		v.PackageName = pkg.PackageName
	}
	if cli != nil {
		v.ClientName = cli.ClientName
	}
	return &PackageClientMappingUpdateResult{RetVal: 1, Mapping: v, Message: "Package-Client mapping status updated successfully"}, nil
}

func (s *packageService) CreatePackageLabMapping(packageID int, labID int64, price float64, createdBy, lastUpdatedBy int64) (*PackageLabMappingResult, error) {
	if _, err := s.repo.FindByID(packageID); err != nil {
		return nil, apperrors.NewNotFound("Package not found", err)
	}
	if _, err := s.labRepo.FindByID(labID); err != nil {
		return nil, apperrors.NewNotFound("Lab not found", err)
	}
	existing, _ := s.labMapRepo.FindByPackageAndLab(packageID, labID)
	if existing != nil {
		v := mappingToLabView(existing, "", "")
		pkg, _ := s.repo.FindByID(packageID)
		lab, _ := s.labRepo.FindByID(labID)
		if pkg != nil {
			v.PackageName = pkg.PackageName
		}
		if lab != nil {
			v.LabName = lab.LabName
		}
		return &PackageLabMappingResult{
			RetVal:  2,
			Mapping: v,
			Message: "Package-Lab mapping already exists",
		}, nil
	}
	m := &persistencemodels.PackageLabMapping{
		PackageID: packageID, LabID: labID, Price: price,
		IsActive: true, CreatedBy: createdBy, LastUpdatedBy: lastUpdatedBy,
	}
	if err := s.labMapRepo.Create(m); err != nil {
		return nil, err
	}
	v := mappingToLabView(m, "", "")
	pkg, _ := s.repo.FindByID(packageID)
	lab, _ := s.labRepo.FindByID(labID)
	if pkg != nil {
		v.PackageName = pkg.PackageName
	}
	if lab != nil {
		v.LabName = lab.LabName
	}
	return &PackageLabMappingResult{RetVal: 1, Mapping: v, Message: "Package-Lab mapping created successfully"}, nil
}

func mappingToLabView(m *persistencemodels.PackageLabMapping, pkgName, labName string) *domain.PackageLabMappingView {
	if m == nil {
		return nil
	}
	return &domain.PackageLabMappingView{
		PackageLabID:  m.PackageLabID,
		PackageID:     m.PackageID,
		LabID:         m.LabID,
		Price:         m.Price,
		IsActive:      m.IsActive,
		CreatedBy:     m.CreatedBy,
		CreatedOn:     m.CreatedOn,
		LastUpdatedBy: m.LastUpdatedBy,
		LastUpdatedOn: m.LastUpdatedOn,
		PackageName:   pkgName,
		LabName:       labName,
	}
}

func (s *packageService) GetAllPackageLabMappings() ([]domain.PackageLabMappingView, error) {
	list, err := s.labMapRepo.FindAll()
	if err != nil {
		return nil, err
	}
	packages, _ := s.repo.FindAll()
	labs, _ := s.labRepo.FindAll()
	pkgMap := make(map[int]string)
	for _, p := range packages {
		pkgMap[p.PackageID] = p.PackageName
	}
	labMap := make(map[int64]string)
	for _, l := range labs {
		labMap[l.LabID] = l.LabName
	}
	var out []domain.PackageLabMappingView
	for i := range list {
		v := mappingToLabView(&list[i], pkgMap[list[i].PackageID], labMap[list[i].LabID])
		out = append(out, *v)
	}
	return out, nil
}

func (s *packageService) UpdatePackageLabMappingStatus(id int, isActive bool, lastUpdatedBy int64) (*PackageLabMappingUpdateResult, error) {
	m, err := s.labMapRepo.FindByID(id)
	if err != nil || m == nil {
		return nil, apperrors.NewNotFound("Package-Lab mapping not found", gorm.ErrRecordNotFound)
	}
	if isActive {
		pkg, _ := s.repo.FindByID(m.PackageID)
		if pkg != nil && !pkg.IsActive {
			v := mappingToLabView(m, "", "")
			pkg2, _ := s.repo.FindByID(m.PackageID)
			lab, _ := s.labRepo.FindByID(m.LabID)
			if pkg2 != nil {
				v.PackageName = pkg2.PackageName
			}
			if lab != nil {
				v.LabName = lab.LabName
			}
			return &PackageLabMappingUpdateResult{
				RetVal:  2,
				Mapping: v,
				Message: "Cannot activate lab mapping because the package is inactive",
			}, nil
		}
	}
	m.IsActive = isActive
	m.LastUpdatedBy = lastUpdatedBy
	if err := s.labMapRepo.Update(m); err != nil {
		return nil, err
	}
	v := mappingToLabView(m, "", "")
	pkg, _ := s.repo.FindByID(m.PackageID)
	lab, _ := s.labRepo.FindByID(m.LabID)
	if pkg != nil {
		v.PackageName = pkg.PackageName
	}
	if lab != nil {
		v.LabName = lab.LabName
	}
	return &PackageLabMappingUpdateResult{RetVal: 1, Mapping: v, Message: "Package-Lab mapping status updated successfully"}, nil
}
