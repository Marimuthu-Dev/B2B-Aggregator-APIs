package handlers

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"strings"

	"b2b-diagnostic-aggregator/apis/internal/apperrors"
	"b2b-diagnostic-aggregator/apis/internal/dto"
	"b2b-diagnostic-aggregator/apis/internal/middleware"
	persistencemodels "b2b-diagnostic-aggregator/apis/internal/persistence/models"
	"b2b-diagnostic-aggregator/apis/internal/repository"
	"b2b-diagnostic-aggregator/apis/internal/service"

	"github.com/gin-gonic/gin"
)

const (
	labMultipartFormField          = "data"
	labCollectionPincodesFileField = "file"
	labMouDocumentField            = "mou_document"
)

// parsePincodesCSV reads a CSV with header "Pincodes". Supports:
// - One row with comma-separated values: Pincodes\n10001,10002,10003
// - One value per row (single column): Pincodes\n10001\n10002\n10003
// Returns comma-separated pincodes and nil, or empty string and error.
func parsePincodesCSV(data []byte) (string, error) {
	if len(data) == 0 {
		return "", apperrors.NewBadRequest("CSV file is empty", nil)
	}
	reader := csv.NewReader(strings.NewReader(string(data)))
	reader.FieldsPerRecord = -1
	rows, err := reader.ReadAll()
	if err != nil {
		return "", apperrors.NewBadRequest("Invalid CSV: "+err.Error(), err)
	}
	if len(rows) == 0 {
		return "", nil
	}
	headers := rows[0]
	pincodeCol := -1
	for i, h := range headers {
		if strings.TrimSpace(strings.ToLower(h)) == "pincodes" {
			pincodeCol = i
			break
		}
	}
	if pincodeCol < 0 {
		return "", apperrors.NewBadRequest("CSV must have a header column named 'Pincodes'", nil)
	}
	var pincodes []string
	for i := 1; i < len(rows); i++ {
		for _, cell := range rows[i] {
			v := strings.TrimSpace(cell)
			if v != "" {
				pincodes = append(pincodes, v)
			}
		}
	}
	return strings.Join(pincodes, ","), nil
}

func ignoreLabMapLocationURLIfUnsupported(create *dto.LabRequest, update *dto.LabUpdateRequest) {
	if persistencemodels.HasLabMapLocationURLColumn() {
		return
	}
	if create != nil {
		create.MapLocationURL = nil
	}
	if update != nil {
		update.MapLocationURL = nil
	}
}

type LabHandler struct {
	svc service.LabService
}

func NewLabHandler(svc service.LabService) *LabHandler {
	return &LabHandler{svc: svc}
}

func (h *LabHandler) GetAll(c *gin.Context) {
	var query dto.LabListQuery
	if !middleware.BindQuery(c, &query) {
		return
	}
	mouStatuses, mouExpiryParsed, parseErr := dto.ParseMouListFilters(query.MouListQuery)
	if parseErr != nil {
		respondError(c, apperrors.NewBadRequest(parseErr.Error(), parseErr))
		return
	}
	var mouExpiryRange *repository.MouExpiryDateRange
	if mouExpiryParsed != nil {
		mouExpiryRange = &repository.MouExpiryDateRange{From: mouExpiryParsed.From, To: mouExpiryParsed.To}
	}
	page := query.PaginationQuery.Normalize("createdOn", 500) // default 500 so GET without params returns all labs
	filter := repository.LabListFilter{
		Page:           page.Page,
		PageSize:       page.PageSize,
		SortBy:         page.SortBy,
		SortOrder:      page.SortOrder,
		CityID:         query.CityID,
		StateID:        query.StateID,
		IsActive:       query.IsActive,
		MouStatuses:    mouStatuses,
		MouExpiryRange: mouExpiryRange,
		Search:         query.Search,
	}

	data, total, err := h.svc.ListLabs(filter)
	if err != nil {
		respondError(c, err)
		return
	}
	respondData(c, http.StatusOK, data, "Success", gin.H{
		"Count":    len(data),
		"Page":     filter.Page,
		"PageSize": filter.PageSize,
		"Total":    total,
	})
}

func (h *LabHandler) GetByID(c *gin.Context) {
	var params dto.IDParam
	if !middleware.BindUri(c, &params) {
		return
	}
	data, err := h.svc.GetLabByID(params.ID)
	if err != nil {
		respondError(c, err)
		return
	}
	respondData(c, http.StatusOK, data, "Success", nil)
}

func (h *LabHandler) GetByContactNumber(c *gin.Context) {
	var query dto.ContactNumberQuery
	if !middleware.BindQuery(c, &query) {
		return
	}
	data, err := h.svc.GetLabByContactNumber(query.ContactNumber)
	if err != nil {
		respondError(c, err)
		return
	}
	respondData(c, http.StatusOK, data, "Success", nil)
}

// GetLabMoUDownloadURL returns a short-lived SAS URL to view/download the lab MoU PDF.
func (h *LabHandler) GetLabMoUDownloadURL(c *gin.Context) {
	var params dto.IDParam
	if !middleware.BindUri(c, &params) {
		return
	}
	if !middleware.RequirePositiveID(c, params.ID) {
		return
	}
	data, err := h.svc.GetLabMoUDownloadURL(c.Request.Context(), params.ID)
	if err != nil {
		respondError(c, err)
		return
	}
	respondData(c, http.StatusOK, data, "Success", nil)
}

func (h *LabHandler) Create(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		respondError(c, apperrors.NewUnauthorized("Authentication required", nil))
		return
	}
	var req dto.LabRequest
	contentType := c.GetHeader("Content-Type")
	if strings.Contains(contentType, "multipart/form-data") {
		dataStr := c.PostForm(labMultipartFormField)
		if dataStr == "" {
			respondError(c, apperrors.NewBadRequest("multipart request must include 'data' field with JSON body", nil))
			return
		}
		if err := json.Unmarshal([]byte(dataStr), &req); err != nil {
			respondError(c, apperrors.NewBadRequest("Invalid JSON in data field: "+err.Error(), err))
			return
		}
		ignoreLabMapLocationURLIfUnsupported(&req, nil)
		file, err := c.FormFile(labCollectionPincodesFileField)
		if err == nil && file != nil {
			f, openErr := file.Open()
			if openErr != nil {
				respondError(c, apperrors.NewBadRequest("Failed to read file", openErr))
				return
			}
			buf, readErr := io.ReadAll(f)
			_ = f.Close()
			if readErr != nil {
				respondError(c, apperrors.NewBadRequest("Failed to read file", readErr))
				return
			}
			pincodesStr, parseErr := parsePincodesCSV(buf)
			if parseErr != nil {
				respondError(c, parseErr)
				return
			}
			fas := dto.FlexArrayString(pincodesStr)
			req.CollectionPincodes = &fas
		}
		var mou *multipart.FileHeader
		mouFh, mouErr := c.FormFile(labMouDocumentField)
		if mouErr != nil {
			if !errors.Is(mouErr, http.ErrMissingFile) {
				respondError(c, apperrors.NewBadRequest("mou_document: "+mouErr.Error(), mouErr))
				return
			}
		} else {
			mou = mouFh
		}
		lab := req.ToDomain()
		if mou != nil {
			if err := h.svc.CreateLabWithMoU(c.Request.Context(), &lab, userID, mou); err != nil {
				respondError(c, err)
				return
			}
		} else {
			if err := h.svc.CreateLab(&lab, userID); err != nil {
				respondError(c, err)
				return
			}
		}
		respondData(c, http.StatusCreated, lab, "Lab created successfully", nil)
		return
	}
	if !middleware.BindJSON(c, &req) {
		return
	}
	ignoreLabMapLocationURLIfUnsupported(&req, nil)
	lab := req.ToDomain()
	if err := h.svc.CreateLab(&lab, userID); err != nil {
		respondError(c, err)
		return
	}
	respondData(c, http.StatusCreated, lab, "Lab created successfully", nil)
}

func (h *LabHandler) Update(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		respondError(c, apperrors.NewUnauthorized("Authentication required", nil))
		return
	}
	var params dto.IDParam
	if !middleware.BindUri(c, &params) {
		return
	}
	if !middleware.RequirePositiveID(c, params.ID) {
		return
	}
	var req dto.LabUpdateRequest
	contentType := c.GetHeader("Content-Type")
	if strings.Contains(contentType, "multipart/form-data") {
		dataStr := strings.TrimSpace(c.PostForm(labMultipartFormField))
		var mou *multipart.FileHeader
		mouFh, mouErr := c.FormFile(labMouDocumentField)
		if mouErr != nil {
			if !errors.Is(mouErr, http.ErrMissingFile) {
				respondError(c, apperrors.NewBadRequest("mou_document: "+mouErr.Error(), mouErr))
				return
			}
		} else {
			mou = mouFh
		}
		if dataStr != "" {
			if err := json.Unmarshal([]byte(dataStr), &req); err != nil {
				respondError(c, apperrors.NewBadRequest("Invalid JSON in data field: "+err.Error(), err))
				return
			}
		}
		ignoreLabMapLocationURLIfUnsupported(nil, &req)
		file, err := c.FormFile(labCollectionPincodesFileField)
		if err == nil && file != nil {
			f, openErr := file.Open()
			if openErr != nil {
				respondError(c, apperrors.NewBadRequest("Failed to read file", openErr))
				return
			}
			buf, readErr := io.ReadAll(f)
			_ = f.Close()
			if readErr != nil {
				respondError(c, apperrors.NewBadRequest("Failed to read file", readErr))
				return
			}
			pincodesStr, parseErr := parsePincodesCSV(buf)
			if parseErr != nil {
				respondError(c, parseErr)
				return
			}
			fas := dto.FlexArrayString(pincodesStr)
			req.CollectionPincodes = &fas
		}
		if !req.HasAtLeastOneField() && mou == nil {
			respondError(c, apperrors.NewBadRequest("multipart request must include 'data' (JSON), CSV 'file' for pincodes, and/or 'mou_document'", nil))
			return
		}
		if mou != nil {
			lab, updErr := h.svc.UpdateLabWithMoU(c.Request.Context(), params.ID, &req, userID, mou)
			if updErr != nil {
				respondError(c, updErr)
				return
			}
			respondData(c, http.StatusOK, lab, "Lab updated successfully", nil)
			return
		}
		lab, updErr := h.svc.UpdateLab(params.ID, &req, userID)
		if updErr != nil {
			respondError(c, updErr)
			return
		}
		respondData(c, http.StatusOK, lab, "Lab updated successfully", nil)
		return
	}
	if !middleware.BindJSON(c, &req) {
		return
	}
	ignoreLabMapLocationURLIfUnsupported(nil, &req)
	if !req.HasAtLeastOneField() {
		respondError(c, apperrors.NewBadRequest("At least one field is required in the payload to update", nil))
		return
	}
	lab, err := h.svc.UpdateLab(params.ID, &req, userID)
	if err != nil {
		respondError(c, err)
		return
	}
	respondData(c, http.StatusOK, lab, "Lab updated successfully", nil)
}

func (h *LabHandler) Delete(c *gin.Context) {
	var params dto.IDParam
	if !middleware.BindUri(c, &params) {
		return
	}
	if !middleware.RequirePositiveID(c, params.ID) {
		return
	}
	if err := h.svc.DeleteLab(params.ID); err != nil {
		respondError(c, err)
		return
	}
	respondMessage(c, http.StatusOK, "Lab deleted successfully")
}
