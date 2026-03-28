package service

import (
	"context"
	"errors"
	"log/slog"
	"mime/multipart"
	"strings"
	"time"

	"b2b-diagnostic-aggregator/apis/internal/apperrors"
	"b2b-diagnostic-aggregator/apis/internal/domain"
	"b2b-diagnostic-aggregator/apis/internal/dto"
	"b2b-diagnostic-aggregator/apis/internal/repository"
	"b2b-diagnostic-aggregator/apis/internal/timeutil"

	"gorm.io/gorm"
)

type LabService interface {
	ListLabs(filter repository.LabListFilter) ([]domain.Lab, int64, error)
	GetLabByID(id int64) (*domain.Lab, error)
	GetLabByContactNumber(contactNumber string) (*domain.Lab, error)
	CreateLab(l *domain.Lab, createdBy int64) error
	CreateLabWithMoU(ctx context.Context, l *domain.Lab, createdBy int64, mou *multipart.FileHeader) error
	UpdateLab(id int64, update *dto.LabUpdateRequest, lastUpdatedBy int64) (*domain.Lab, error)
	UpdateLabWithMoU(ctx context.Context, id int64, update *dto.LabUpdateRequest, lastUpdatedBy int64, mou *multipart.FileHeader) (*domain.Lab, error)
	GetLabMoUDownloadURL(ctx context.Context, labID int64) (*dto.LabMoUDownloadURLResponse, error)
	DeleteLab(id int64) error
	GetActiveLabs() ([]domain.Lab, error)
	GetLabsByCity(cityID int8) ([]domain.Lab, error)
	GetLabsByState(stateID int8) ([]domain.Lab, error)
}

type labService struct {
	repo  repository.LabRepository
	blobs BlobService
}

func NewLabService(repo repository.LabRepository, blobs BlobService) LabService {
	return &labService{repo: repo, blobs: blobs}
}

func (s *labService) ListLabs(filter repository.LabListFilter) ([]domain.Lab, int64, error) {
	return s.repo.List(filter)
}

func (s *labService) GetLabByID(id int64) (*domain.Lab, error) {
	lab, err := s.repo.FindByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperrors.NewNotFound("Lab not found", err)
	}
	return lab, err
}

func (s *labService) GetLabByContactNumber(contactNumber string) (*domain.Lab, error) {
	lab, err := s.repo.FindByContactNumber(contactNumber)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperrors.NewNotFound("Lab not found", err)
	}
	return lab, err
}

func (s *labService) CreateLab(l *domain.Lab, createdBy int64) error {
	now := time.Now()
	l.CreatedBy = &createdBy
	l.CreatedOn = timeutil.FromTimePtr(&now)
	l.LastUpdatedBy = &createdBy
	l.LastUpdatedOn = timeutil.FromTimePtr(&now)
	return s.repo.Create(l)
}

func (s *labService) CreateLabWithMoU(ctx context.Context, l *domain.Lab, createdBy int64, mou *multipart.FileHeader) error {
	if mou != nil {
		if s.blobs == nil {
			return apperrors.NewInternal("MoU storage is not configured", nil)
		}
		if err := s.blobs.ValidatePDF(mou); err != nil {
			return apperrors.NewBadRequest(err.Error(), err)
		}
	}
	now := time.Now()
	l.CreatedBy = &createdBy
	l.CreatedOn = timeutil.FromTimePtr(&now)
	l.LastUpdatedBy = &createdBy
	l.LastUpdatedOn = timeutil.FromTimePtr(&now)
	if err := s.repo.Create(l); err != nil {
		return err
	}
	if mou == nil {
		return nil
	}
	rc, err := mou.Open()
	if err != nil {
		s.rollbackLabAfterFailedMoU(l.LabID, "open MoU file")
		return apperrors.NewInternal("Failed to read MoU upload", err)
	}
	defer func() { _ = rc.Close() }()
	url, err := s.blobs.UploadOrReplaceLabMoU(ctx, rc, l.LabID)
	if err != nil {
		slog.Error("CreateLabWithMoU: blob upload failed", slog.Int64("labID", l.LabID), slog.Any("err", err))
		s.rollbackLabAfterFailedMoU(l.LabID, "MoU upload")
		return apperrors.NewInternal("Failed to upload MoU document", err)
	}
	if err := s.repo.UpdateLabMoUURL(l.LabID, url); err != nil {
		slog.Error("CreateLabWithMoU: save MoU URL failed (lab and blob exist; fix URL or re-run update)",
			slog.Int64("labID", l.LabID), slog.String("blobURL", url), slog.Any("err", err))
		return apperrors.NewInternal("Failed to save MoU document URL", err)
	}
	l.MoUDocumentURL = &url
	return nil
}

func (s *labService) rollbackLabAfterFailedMoU(labID int64, phase string) {
	if err := s.repo.Delete(labID); err != nil {
		slog.Error("CreateLabWithMoU: rollback delete failed — orphaned lab row may remain",
			slog.Int64("labID", labID), slog.String("phase", phase), slog.Any("err", err))
		return
	}
	slog.Info("CreateLabWithMoU: rolled back lab row after MoU failure", slog.Int64("labID", labID), slog.String("phase", phase))
}

func applyLabUpdatePatch(l *domain.Lab, update *dto.LabUpdateRequest) {
	if update.LabName != nil {
		l.LabName = *update.LabName
	}
	if update.Address != nil {
		l.Address = update.Address
	}
	if update.CityID != nil {
		l.CityID = update.CityID
	}
	if update.StateID != nil {
		l.StateID = update.StateID
	}
	if update.Pincode != nil {
		l.Pincode = update.Pincode
	}
	if update.ContactPerson1Name != nil {
		l.ContactPerson1Name = update.ContactPerson1Name
	}
	if update.ContactPerson1Number != nil {
		l.ContactPerson1Number = update.ContactPerson1Number
	}
	if update.ContactPerson1EmailID != nil {
		l.ContactPerson1EmailID = update.ContactPerson1EmailID
	}
	if update.ContactPerson1Designation != nil {
		l.ContactPerson1Designation = update.ContactPerson1Designation
	}
	if update.ContactPerson1Name1 != nil {
		l.ContactPerson1Name1 = update.ContactPerson1Name1
	}
	if update.ContactPerson1Number1 != nil {
		l.ContactPerson1Number1 = update.ContactPerson1Number1
	}
	if update.ContactPerson1EmailID1 != nil {
		l.ContactPerson1EmailID1 = update.ContactPerson1EmailID1
	}
	if update.ContactPerson1Designation1 != nil {
		l.ContactPerson1Designation1 = update.ContactPerson1Designation1
	}
	if update.CategoryID != nil {
		l.CategoryID = update.CategoryID
	}
	if update.GSTIN_UIN != nil {
		l.GSTIN_UIN = update.GSTIN_UIN
	}
	if update.PANNumber != nil {
		l.PANNumber = update.PANNumber
	}
	if t := update.GetMOUStartDate(); t != nil {
		l.MOUStartDate = t
	}
	if t := update.GetMOUEndDate(); t != nil {
		l.MOUEndDate = t
	}
	if update.AccreditationID != nil {
		l.AccreditationID = update.AccreditationID
	}
	if t := update.GetAccreditationExpirationDate(); t != nil {
		l.AccreditationExpirationDate = t
	}
	if s := update.GetCollectionTypes(); s != nil {
		l.CollectionTypes = s
	}
	if s := update.GetServicesID(); s != nil {
		l.ServicesID = s
	}
	if s := update.GetCollectionPincodes(); s != nil {
		l.CollectionPincodes = s
	}
	if update.LabGrade != nil {
		l.LabGrade = update.LabGrade
	}
	if update.IsActive != nil {
		l.IsActive = update.IsActive
	}
}

func (s *labService) UpdateLab(id int64, update *dto.LabUpdateRequest, lastUpdatedBy int64) (*domain.Lab, error) {
	existing, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.NewNotFound("Lab not found", err)
		}
		return nil, err
	}
	l := *existing
	applyLabUpdatePatch(&l, update)
	l.LabID = id
	l.LastUpdatedBy = &lastUpdatedBy
	now := time.Now()
	l.LastUpdatedOn = timeutil.FromTimePtr(&now)
	if err := s.repo.Update(&l); err != nil {
		return nil, err
	}
	return &l, nil
}

func (s *labService) UpdateLabWithMoU(ctx context.Context, id int64, update *dto.LabUpdateRequest, lastUpdatedBy int64, mou *multipart.FileHeader) (*domain.Lab, error) {
	hasFile := mou != nil
	hasFields := update != nil && update.HasAtLeastOneField()
	if !hasFile && !hasFields {
		return nil, apperrors.NewBadRequest("At least one field in data or mou_document is required", nil)
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
			return nil, apperrors.NewNotFound("Lab not found", err)
		}
		return nil, err
	}
	snapshot := *existing

	if hasFields && !hasFile {
		l := *existing
		applyLabUpdatePatch(&l, update)
		l.LabID = id
		l.LastUpdatedBy = &lastUpdatedBy
		now := time.Now()
		l.LastUpdatedOn = timeutil.FromTimePtr(&now)
		if err := s.repo.Update(&l); err != nil {
			return nil, err
		}
		return &l, nil
	}

	if hasFields {
		l := *existing
		applyLabUpdatePatch(&l, update)
		l.LabID = id
		l.LastUpdatedBy = &lastUpdatedBy
		now := time.Now()
		l.LastUpdatedOn = timeutil.FromTimePtr(&now)
		if err := s.repo.Update(&l); err != nil {
			return nil, err
		}
	}

	rc, err := mou.Open()
	if err != nil {
		if hasFields {
			s.rollbackLabUpdateAfterMoUFailure(id, &snapshot, "open MoU file")
		}
		return nil, apperrors.NewInternal("Failed to read MoU upload", err)
	}
	defer func() { _ = rc.Close() }()

	url, err := s.blobs.UploadOrReplaceLabMoU(ctx, rc, id)
	if err != nil {
		slog.Error("UpdateLabWithMoU: blob upload failed", slog.Int64("labID", id), slog.Any("err", err))
		if hasFields {
			s.rollbackLabUpdateAfterMoUFailure(id, &snapshot, "MoU upload")
		}
		return nil, apperrors.NewInternal("Failed to upload MoU document", err)
	}

	latest, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	out := *latest
	out.MoUDocumentURL = &url
	out.LabID = id
	out.LastUpdatedBy = &lastUpdatedBy
	now := time.Now()
	out.LastUpdatedOn = timeutil.FromTimePtr(&now)
	if err := s.repo.Update(&out); err != nil {
		slog.Error("UpdateLabWithMoU: save MoU URL failed after successful upload",
			slog.Int64("labID", id), slog.String("blobURL", url), slog.Any("err", err))
		return nil, err
	}
	return &out, nil
}

func (s *labService) rollbackLabUpdateAfterMoUFailure(labID int64, snapshot *domain.Lab, phase string) {
	restored := *snapshot
	restored.LabID = labID
	if err := s.repo.Update(&restored); err != nil {
		slog.Error("UpdateLabWithMoU: rollback failed — lab row may still reflect partial update",
			slog.Int64("labID", labID), slog.String("phase", phase), slog.Any("err", err))
		return
	}
	slog.Info("UpdateLabWithMoU: rolled back lab row after MoU failure", slog.Int64("labID", labID), slog.String("phase", phase))
}

func (s *labService) GetLabMoUDownloadURL(ctx context.Context, labID int64) (*dto.LabMoUDownloadURLResponse, error) {
	if s.blobs == nil {
		return nil, apperrors.NewInternal("MoU storage is not configured", nil)
	}
	lab, err := s.repo.FindByID(labID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.NewNotFound("Lab not found", err)
		}
		return nil, err
	}
	if lab.MoUDocumentURL == nil || strings.TrimSpace(*lab.MoUDocumentURL) == "" {
		return nil, apperrors.NewNotFound("No MoU document for this lab", nil)
	}
	urlStr, exp, err := s.blobs.LabMoUDownloadSASURL(ctx, labID)
	if err != nil {
		slog.Error("GetLabMoUDownloadURL: SAS signing failed", slog.Int64("labID", labID), slog.Any("err", err))
		return nil, apperrors.NewInternal("Failed to generate download link", err)
	}
	return &dto.LabMoUDownloadURLResponse{URL: urlStr, ExpiresAt: exp}, nil
}

func (s *labService) DeleteLab(id int64) error {
	exists, err := s.repo.ExistsByID(id)
	if err != nil {
		return err
	}
	if !exists {
		return apperrors.NewNotFound("Lab not found", gorm.ErrRecordNotFound)
	}
	return s.repo.Delete(id)
}

func (s *labService) GetActiveLabs() ([]domain.Lab, error) {
	return s.repo.FindAllActive()
}

func (s *labService) GetLabsByCity(cityID int8) ([]domain.Lab, error) {
	return s.repo.FindByCity(cityID)
}

func (s *labService) GetLabsByState(stateID int8) ([]domain.Lab, error) {
	return s.repo.FindByState(stateID)
}
