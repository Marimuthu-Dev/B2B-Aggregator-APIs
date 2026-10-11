package service

import (
	"b2b-diagnostic-aggregator/apis/internal/dto"
	"b2b-diagnostic-aggregator/apis/internal/repository"
)

type LocationService interface {
	GetAllStates() ([]dto.StateResponse, error)
	GetAllCities() ([]dto.CityResponse, error)
	GetCitiesByStateID(stateID int32) ([]dto.CityResponse, error)
}

type locationService struct {
	repo repository.LocationRepository
}

func NewLocationService(repo repository.LocationRepository) LocationService {
	return &locationService{repo: repo}
}

func (s *locationService) GetAllStates() ([]dto.StateResponse, error) {
	states, err := s.repo.GetAllStates()
	if err != nil {
		return nil, err
	}
	res := make([]dto.StateResponse, len(states))
	for i, st := range states {
		res[i] = dto.StateResponse{
			ID:   st.StateID,
			Name: st.StateName,
		}
	}
	return res, nil
}

func (s *locationService) GetAllCities() ([]dto.CityResponse, error) {
	cities, err := s.repo.GetAllCities()
	if err != nil {
		return nil, err
	}
	res := make([]dto.CityResponse, len(cities))
	for i, ct := range cities {
		res[i] = dto.CityResponse{
			ID:      ct.CityID,
			Name:    ct.CityName,
			StateID: ct.StateID,
		}
	}
	return res, nil
}

func (s *locationService) GetCitiesByStateID(stateID int32) ([]dto.CityResponse, error) {
	cities, err := s.repo.GetCitiesByStateID(stateID)
	if err != nil {
		return nil, err
	}
	res := make([]dto.CityResponse, len(cities))
	for i, ct := range cities {
		res[i] = dto.CityResponse{
			ID:      ct.CityID,
			Name:    ct.CityName,
			StateID: ct.StateID,
		}
	}
	return res, nil
}
