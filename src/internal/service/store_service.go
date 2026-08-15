package service

import (
	"errors"
	"strings"
	"time"

	"b2b-diagnostic-aggregator/apis/internal/apperrors"
	"b2b-diagnostic-aggregator/apis/internal/domain"
	"b2b-diagnostic-aggregator/apis/internal/dto"
	"b2b-diagnostic-aggregator/apis/internal/repository"
	"b2b-diagnostic-aggregator/apis/internal/timeutil"
	"b2b-diagnostic-aggregator/apis/pkg/utils"

	"gorm.io/gorm"
)

type StoreService interface {
	ListStores(filter repository.StoreListFilter) ([]domain.Store, int64, error)
	GetStoreByID(id int64) (*domain.Store, error)
	CreateStore(s *domain.Store, createdBy int64, password string) error
	UpdateStore(id int64, update *dto.StoreUpdateRequest, lastUpdatedBy int64) (*domain.Store, error)
}

type storeService struct {
	repo       repository.StoreRepository
	clientRepo repository.ClientRepository
}

func NewStoreService(repo repository.StoreRepository, clientRepo repository.ClientRepository) StoreService {
	return &storeService{repo: repo, clientRepo: clientRepo}
}

func (s *storeService) ListStores(filter repository.StoreListFilter) ([]domain.Store, int64, error) {
	stores, total, err := s.repo.List(filter)
	if err != nil {
		return nil, 0, err
	}
	s.attachClientNames(stores)
	return stores, total, nil
}

func (s *storeService) GetStoreByID(id int64) (*domain.Store, error) {
	store, err := s.repo.FindByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperrors.NewNotFound("Store not found", err)
	}
	if err != nil {
		return nil, err
	}
	s.attachClientName(store)
	return store, nil
}

func (s *storeService) CreateStore(st *domain.Store, createdBy int64, password string) error {
	st.StoreName = strings.TrimSpace(st.StoreName)
	st.Address = strings.TrimSpace(st.Address)
	st.Pincode = strings.TrimSpace(st.Pincode)
	st.ContactNumber = strings.TrimSpace(st.ContactNumber)
	st.EmailID = strings.TrimSpace(st.EmailID)

	client, err := s.clientRepo.FindByID(st.ClientID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return apperrors.NewBadRequest("Client not found", err)
	}
	if err != nil {
		return err
	}
	if !client.IsAcitve {
		return apperrors.NewBadRequest("Client is not active", nil)
	}
	if !client.IsStoreLoginEnabled {
		return apperrors.NewBadRequest("Store login is not enabled for this client", nil)
	}

	if err := s.ensureMobileUnique(st.ContactNumber, 0); err != nil {
		return err
	}
	if err := s.ensureEmailUnique(st.EmailID, 0); err != nil {
		return err
	}

	now := time.Now()
	st.CreatedBy = createdBy
	st.CreatedOn = timeutil.FromTime(now)
	st.LastUpdatedBy = createdBy
	st.LastUpdatedOn = timeutil.FromTime(now)

	pwd := strings.TrimSpace(password)
	if pwd == "" {
		pwd = st.ContactNumber
	}
	encrypted, err := utils.Encrypt(pwd)
	if err != nil {
		return apperrors.NewInternal("Failed to create store login", err)
	}
	login := &domain.Login{
		Pwd:           encrypted,
		UserType:      utils.UserTypeToLoginString(utils.UserTypeStore),
		CreatedOn:     timeutil.FromTime(now),
		LastUpdatedOn: timeutil.FromTime(now),
	}
	if err := s.repo.CreateWithLogin(st, login); err != nil {
		return err
	}
	st.ClientName = client.ClientName
	return nil
}

func (s *storeService) UpdateStore(id int64, update *dto.StoreUpdateRequest, lastUpdatedBy int64) (*domain.Store, error) {
	existing, err := s.repo.FindByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperrors.NewNotFound("Store not found", err)
	}
	if err != nil {
		return nil, err
	}

	st := *existing
	if update.StoreName != nil {
		st.StoreName = strings.TrimSpace(*update.StoreName)
	}
	if update.Address != nil {
		st.Address = strings.TrimSpace(*update.Address)
	}
	if update.CityID != nil {
		st.CityID = *update.CityID
	}
	if update.StateID != nil {
		st.StateID = *update.StateID
	}
	if update.Pincode != nil {
		st.Pincode = strings.TrimSpace(*update.Pincode)
	}
	if update.ContactNumber != nil {
		st.ContactNumber = strings.TrimSpace(*update.ContactNumber)
	}
	if update.EmailID != nil {
		st.EmailID = strings.TrimSpace(*update.EmailID)
	}
	if update.IsActive != nil {
		st.IsActive = *update.IsActive
	}

	if update.ContactNumber != nil && st.ContactNumber != existing.ContactNumber {
		if err := s.ensureMobileUnique(st.ContactNumber, st.StoreID); err != nil {
			return nil, err
		}
	}
	if update.EmailID != nil && !strings.EqualFold(st.EmailID, existing.EmailID) {
		if err := s.ensureEmailUnique(st.EmailID, st.StoreID); err != nil {
			return nil, err
		}
	}

	st.StoreID = id
	st.ClientID = existing.ClientID
	st.LastUpdatedBy = lastUpdatedBy
	st.LastUpdatedOn = timeutil.FromTime(time.Now())
	if err := s.repo.Update(&st); err != nil {
		return nil, err
	}
	s.attachClientName(&st)
	return &st, nil
}

func (s *storeService) ensureMobileUnique(mobile string, excludeStoreID int64) error {
	mobile = strings.TrimSpace(mobile)
	if mobile == "" {
		return apperrors.NewBadRequest("ContactNumber is required", nil)
	}
	taken, err := s.repo.ExistsByContactNumber(mobile, excludeStoreID)
	if err != nil {
		return err
	}
	if taken {
		return apperrors.NewBadRequest("ContactNumber is already used by another store", nil)
	}
	client, err := s.clientRepo.FindByContactNumber(mobile)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if client != nil {
		return apperrors.NewBadRequest("ContactNumber is already used by a client login", nil)
	}
	return nil
}

func (s *storeService) ensureEmailUnique(email string, excludeStoreID int64) error {
	email = strings.TrimSpace(email)
	if email == "" {
		return apperrors.NewBadRequest("EmailID is required", nil)
	}
	taken, err := s.repo.ExistsByEmailID(email, excludeStoreID)
	if err != nil {
		return err
	}
	if taken {
		return apperrors.NewBadRequest("EmailID is already used by another store", nil)
	}
	return nil
}

func (s *storeService) attachClientName(store *domain.Store) {
	if store == nil || store.ClientID == 0 {
		return
	}
	if client, err := s.clientRepo.FindByID(store.ClientID); err == nil && client != nil {
		store.ClientName = client.ClientName
	}
}

func (s *storeService) attachClientNames(stores []domain.Store) {
	for i := range stores {
		s.attachClientName(&stores[i])
	}
}
