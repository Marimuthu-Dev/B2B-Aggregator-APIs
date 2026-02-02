package repository

import (
	"b2b-diagnostic-aggregator/apis/internal/domain"
	persistencemodels "b2b-diagnostic-aggregator/apis/internal/persistence/models"
)

func mapPackageToDomain(p persistencemodels.Package) domain.Package {
	return domain.Package{
		PackageID:     p.PackageID,
		PackageName:   p.PackageName,
		Description:   p.Description,
		IsActive:      p.IsActive,
		CreatedBy:     p.CreatedBy,
		CreatedOn:     p.CreatedOn,
		LastUpdatedBy: p.LastUpdatedBy,
		LastUpdatedOn: p.LastUpdatedOn,
	}
}

func mapPackageToPersistence(d domain.Package) persistencemodels.Package {
	return persistencemodels.Package{
		PackageID:     d.PackageID,
		PackageName:   d.PackageName,
		Description:   d.Description,
		IsActive:      d.IsActive,
		CreatedBy:     d.CreatedBy,
		CreatedOn:     d.CreatedOn,
		LastUpdatedBy: d.LastUpdatedBy,
		LastUpdatedOn: d.LastUpdatedOn,
	}
}

func mapPackagesToDomain(packages []persistencemodels.Package) []domain.Package {
	if len(packages) == 0 {
		return nil
	}
	mapped := make([]domain.Package, len(packages))
	for i, pkg := range packages {
		mapped[i] = mapPackageToDomain(pkg)
	}
	return mapped
}
