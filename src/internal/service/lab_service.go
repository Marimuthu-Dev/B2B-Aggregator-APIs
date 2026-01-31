package service

import (
	"b2b-diagnostic-aggregator/apis/internal/models"
	"b2b-diagnostic-aggregator/apis/internal/repository"
)

type LabService interface {
	GetAllLabs() ([]models.Lab, error)
	GetLabByID(id int64) (*models.Lab, error)
	GetLabByContactNumber(contactNumber string) (*models.Lab, error)
	CreateLab(l *models.Lab) error
	UpdateLab(id int64, l *models.Lab) error
	DeleteLab(id int64) error
	GetActiveLabs() ([]models.Lab, error)
	GetLabsByCity(cityID int8) ([]models.Lab, error)
	GetLabsByState(stateID int8) ([]models.Lab, error)
}

type labService struct {
	repo repository.LabRepository
}

func NewLabService(repo repository.LabRepository) LabService {
	return &labService{repo: repo}
}

func (s *labService) GetAllLabs() ([]models.Lab, error) {
	return s.repo.FindAll()
}

func (s *labService) GetLabByID(id int64) (*models.Lab, error) {
	return s.repo.FindByID(id)
}

func (s *labService) GetLabByContactNumber(contactNumber string) (*models.Lab, error) {
	return s.repo.FindByContactNumber(contactNumber)
}

func (s *labService) CreateLab(l *models.Lab) error {
	return s.repo.Create(l)
}

func (s *labService) UpdateLab(id int64, l *models.Lab) error {
	l.LabID = id
	return s.repo.Update(l)
}

func (s *labService) DeleteLab(id int64) error {
	return s.repo.Delete(id)
}

func (s *labService) GetActiveLabs() ([]models.Lab, error) {
	return s.repo.FindAllActive()
}

func (s *labService) GetLabsByCity(cityID int8) ([]models.Lab, error) {
	return s.repo.FindByCity(cityID)
}

func (s *labService) GetLabsByState(stateID int8) ([]models.Lab, error) {
	return s.repo.FindByState(stateID)
}
