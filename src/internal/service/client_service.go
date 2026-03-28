package service

import (
	"context"
	"errors"
	"log/slog"
	"mime/multipart"
	"time"

	"b2b-diagnostic-aggregator/apis/internal/apperrors"
	"b2b-diagnostic-aggregator/apis/internal/domain"
	"b2b-diagnostic-aggregator/apis/internal/dto"
	"b2b-diagnostic-aggregator/apis/internal/repository"
	"b2b-diagnostic-aggregator/apis/internal/timeutil"

	"gorm.io/gorm"
)

type ClientService interface {
	ListClients(filter repository.ClientListFilter) ([]domain.Client, int64, error)
	GetClientByID(id int64) (*domain.Client, error)
	GetClientByContactNumber(contactNumber string) (*domain.Client, error)
	CreateClient(c *domain.Client, createdBy int64) error
	CreateClientWithMoU(ctx context.Context, c *domain.Client, createdBy int64, mou *multipart.FileHeader) error
	UpdateClient(id int64, update *dto.ClientUpdateRequest, lastUpdatedBy int64) (*domain.Client, error)
	UpdateClientWithMoU(ctx context.Context, id int64, update *dto.ClientUpdateRequest, lastUpdatedBy int64, mou *multipart.FileHeader) (*domain.Client, error)
	DeleteClient(id int64) error
	GetActiveClients() ([]domain.Client, error)
	GetClientsByCity(cityID int8) ([]domain.Client, error)
	GetClientsByState(stateID int8) ([]domain.Client, error)
}

type clientService struct {
	repo  repository.ClientRepository
	blobs BlobService
}

func NewClientService(repo repository.ClientRepository, blobs BlobService) ClientService {
	return &clientService{repo: repo, blobs: blobs}
}

func (s *clientService) ListClients(filter repository.ClientListFilter) ([]domain.Client, int64, error) {
	return s.repo.List(filter)
}

func (s *clientService) GetClientByID(id int64) (*domain.Client, error) {
	client, err := s.repo.FindByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperrors.NewNotFound("Client not found", err)
	}
	return client, err
}

func (s *clientService) GetClientByContactNumber(contactNumber string) (*domain.Client, error) {
	client, err := s.repo.FindByContactNumber(contactNumber)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperrors.NewNotFound("Client not found", err)
	}
	return client, err
}

func (s *clientService) CreateClient(c *domain.Client, createdBy int64) error {
	now := time.Now()
	c.CreatedBy = createdBy
	c.CreatedOn = timeutil.FromTime(now)
	c.LastUpdatedBy = createdBy
	c.LastUpdatedOn = timeutil.FromTime(now)
	return s.repo.Create(c)
}

func (s *clientService) CreateClientWithMoU(ctx context.Context, c *domain.Client, createdBy int64, mou *multipart.FileHeader) error {
	if mou != nil {
		if s.blobs == nil {
			return apperrors.NewInternal("MoU storage is not configured", nil)
		}
		if err := s.blobs.ValidatePDF(mou); err != nil {
			return apperrors.NewBadRequest(err.Error(), err)
		}
	}

	now := time.Now()
	c.CreatedBy = createdBy
	c.CreatedOn = timeutil.FromTime(now)
	c.LastUpdatedBy = createdBy
	c.LastUpdatedOn = timeutil.FromTime(now)
	if err := s.repo.Create(c); err != nil {
		return err
	}
	if mou == nil {
		return nil
	}

	rc, err := mou.Open()
	if err != nil {
		s.rollbackClientAfterFailedMoU(c.ClientID, "open MoU file")
		return apperrors.NewInternal("Failed to read MoU upload", err)
	}
	defer func() { _ = rc.Close() }()

	url, err := s.blobs.UploadOrReplace(ctx, rc, c.ClientID)
	if err != nil {
		slog.Error("CreateClientWithMoU: blob upload failed", slog.Int64("clientID", c.ClientID), slog.Any("err", err))
		s.rollbackClientAfterFailedMoU(c.ClientID, "MoU upload")
		return apperrors.NewInternal("Failed to upload MoU document", err)
	}
	if err := s.repo.UpdateClientMoUURL(c.ClientID, url); err != nil {
		slog.Error("CreateClientWithMoU: save MoU URL failed (client and blob exist; fix URL or re-run update)",
			slog.Int64("clientID", c.ClientID), slog.String("blobURL", url), slog.Any("err", err))
		return apperrors.NewInternal("Failed to save MoU document URL", err)
	}
	c.MoUDocumentURL = &url
	return nil
}

// rollbackClientAfterFailedMoU deletes the client row so we do not keep a client without a stored MoU
// when the create+MoU flow failed after insert (upload or URL persistence).
func (s *clientService) rollbackClientAfterFailedMoU(clientID int64, phase string) {
	if err := s.repo.Delete(clientID); err != nil {
		slog.Error("CreateClientWithMoU: rollback delete failed — orphaned client row may remain",
			slog.Int64("clientID", clientID),
			slog.String("phase", phase),
			slog.Any("err", err),
		)
		return
	}
	slog.Info("CreateClientWithMoU: rolled back client row after MoU failure",
		slog.Int64("clientID", clientID),
		slog.String("phase", phase),
	)
}

func applyClientUpdatePatch(c *domain.Client, update *dto.ClientUpdateRequest) {
	if update.ClientName != nil {
		c.ClientName = *update.ClientName
	}
	if update.Address != nil {
		c.Address = *update.Address
	}
	if update.CityID != nil {
		c.CityID = *update.CityID
	}
	if update.StateID != nil {
		c.StateID = *update.StateID
	}
	if update.Pincode != nil {
		c.Pincode = *update.Pincode
	}
	if update.ContactPerson1Name != nil {
		c.ContactPerson1Name = *update.ContactPerson1Name
	}
	if update.ContactPerson1Number != nil {
		c.ContactPerson1Number = *update.ContactPerson1Number
	}
	if update.ContactPerson1EmailID != nil {
		c.ContactPerson1EmailID = *update.ContactPerson1EmailID
	}
	if update.ContactPerson1Designation != nil {
		c.ContactPerson1Designation = *update.ContactPerson1Designation
	}
	if update.ContactPerson2Name != nil {
		c.ContactPerson2Name = update.ContactPerson2Name
	}
	if update.ContactPerson2Number != nil {
		c.ContactPerson2Number = update.ContactPerson2Number
	}
	if update.ContactPerson2EmailID != nil {
		c.ContactPerson2EmailID = update.ContactPerson2EmailID
	}
	if update.ContactPerson2Designation != nil {
		c.ContactPerson2Designation = update.ContactPerson2Designation
	}
	if update.CategoryID != nil {
		c.CategoryID = update.CategoryID
	}
	if update.GSTIN_UIN != nil {
		c.GSTIN_UIN = update.GSTIN_UIN
	}
	if update.PANNumber != nil {
		c.PANNumber = update.PANNumber
	}
	if update.BusinessVertical != nil {
		c.BusinessVertical = *update.BusinessVertical
	}
	if update.BillingName != nil {
		c.BillingName = update.BillingName
	}
	if update.BillingAdderss != nil {
		c.BillingAdderss = update.BillingAdderss
	}
	if update.BillingPincode != nil {
		c.BillingPincode = update.BillingPincode
	}
	if update.ClientTypeID != nil {
		c.ClientTypeID = update.ClientTypeID
	}
	if update.IsAcitve != nil {
		c.IsAcitve = *update.IsAcitve
	}
	if update.MOUStartDate != nil {
		c.MOUStartDate = timeutil.FromTimePtr(update.MOUStartDate)
	}
	if update.MOUEndDate != nil {
		c.MOUEndDate = timeutil.FromTimePtr(update.MOUEndDate)
	}
}

func (s *clientService) UpdateClient(id int64, update *dto.ClientUpdateRequest, lastUpdatedBy int64) (*domain.Client, error) {
	existing, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.NewNotFound("Client not found", err)
		}
		return nil, err
	}

	c := *existing
	applyClientUpdatePatch(&c, update)
	c.ClientID = id
	c.LastUpdatedBy = lastUpdatedBy
	c.LastUpdatedOn = timeutil.FromTime(time.Now())
	if err := s.repo.Update(&c); err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *clientService) UpdateClientWithMoU(ctx context.Context, id int64, update *dto.ClientUpdateRequest, lastUpdatedBy int64, mou *multipart.FileHeader) (*domain.Client, error) {
	hasFile := mou != nil
	hasFields := update != nil && update.HasAtLeastOneField()
	if !hasFile && !hasFields {
		return nil, apperrors.NewBadRequest("At least one field or mou_document is required", nil)
	}
	if hasFile {
		if s.blobs == nil {
			return nil, apperrors.NewInternal("MoU storage is not configured", nil)
		}
		if err := s.blobs.ValidatePDF(mou); err != nil {
			return nil, apperrors.NewBadRequest(err.Error(), err)
		}
	}

	existing, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.NewNotFound("Client not found", err)
		}
		return nil, err
	}

	c := *existing
	if hasFields {
		applyClientUpdatePatch(&c, update)
	}

	if hasFile {
		rc, err := mou.Open()
		if err != nil {
			return nil, apperrors.NewInternal("Failed to read MoU upload", err)
		}
		defer func() { _ = rc.Close() }()
		url, err := s.blobs.UploadOrReplace(ctx, rc, id)
		if err != nil {
			slog.Error("UpdateClientWithMoU: blob upload failed", slog.Int64("clientID", id), slog.Any("err", err))
			return nil, apperrors.NewInternal("Failed to upload MoU document", err)
		}
		c.MoUDocumentURL = &url
	}

	c.ClientID = id
	c.LastUpdatedBy = lastUpdatedBy
	c.LastUpdatedOn = timeutil.FromTime(time.Now())
	if err := s.repo.Update(&c); err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *clientService) DeleteClient(id int64) error {
	exists, err := s.repo.ExistsByID(id)
	if err != nil {
		return err
	}
	if !exists {
		return apperrors.NewNotFound("Client not found", gorm.ErrRecordNotFound)
	}
	return s.repo.Delete(id)
}

func (s *clientService) GetActiveClients() ([]domain.Client, error) {
	return s.repo.FindAllActive()
}

func (s *clientService) GetClientsByCity(cityID int8) ([]domain.Client, error) {
	return s.repo.FindByCity(cityID)
}

func (s *clientService) GetClientsByState(stateID int8) ([]domain.Client, error) {
	return s.repo.FindByState(stateID)
}
