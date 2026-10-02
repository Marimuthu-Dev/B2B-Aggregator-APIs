package service

import (
	"b2b-diagnostic-aggregator/apis/internal/dto"
	"b2b-diagnostic-aggregator/apis/internal/repository"
)

type EmploymentTypeService interface {
	GetActiveEmploymentTypes() ([]dto.EmploymentTypeResponse, error)
}

type employmentTypeService struct {
	repo repository.EmploymentTypeRepository
}

func NewEmploymentTypeService(repo repository.EmploymentTypeRepository) EmploymentTypeService {
	return &employmentTypeService{repo: repo}
}

func (s *employmentTypeService) GetActiveEmploymentTypes() ([]dto.EmploymentTypeResponse, error) {
	types, err := s.repo.GetActiveEmploymentTypes()
	if err != nil {
		return nil, err
	}
	res := make([]dto.EmploymentTypeResponse, len(types))
	for i, t := range types {
		res[i] = dto.EmploymentTypeResponse{
			EmploymentTypeID:   t.EmploymentTypeID,
			EmploymentTypeName: t.EmploymentTypeName,
			IsActive:           t.IsActive,
		}
	}
	return res, nil
}
