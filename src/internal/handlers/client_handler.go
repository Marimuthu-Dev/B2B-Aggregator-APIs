package handlers

import (
	"net/http"
	"strings"

	"b2b-diagnostic-aggregator/apis/internal/apperrors"
	"b2b-diagnostic-aggregator/apis/internal/dto"
	"b2b-diagnostic-aggregator/apis/internal/middleware"
	"b2b-diagnostic-aggregator/apis/internal/repository"
	"b2b-diagnostic-aggregator/apis/internal/service"

	"github.com/gin-gonic/gin"
)

type ClientHandler struct {
	svc service.ClientService
}

func NewClientHandler(svc service.ClientService) *ClientHandler {
	return &ClientHandler{svc: svc}
}

func (h *ClientHandler) GetAll(c *gin.Context) {
	var query dto.ClientListQuery
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
	page := query.PaginationQuery.Normalize("createdOn", 500) // default 500 so GET without params returns all clients
	filter := repository.ClientListFilter{
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

	data, total, err := h.svc.ListClients(filter)
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

func (h *ClientHandler) GetByID(c *gin.Context) {
	var params dto.IDParam
	if !middleware.BindUri(c, &params) {
		return
	}
	data, err := h.svc.GetClientByID(params.ID)
	if err != nil {
		respondError(c, err)
		return
	}
	respondData(c, http.StatusOK, data, "Success", nil)
}

// GetMoUDownloadURL returns a short-lived SAS URL to view/download the client's MoU PDF (requires auth).
func (h *ClientHandler) GetMoUDownloadURL(c *gin.Context) {
	var params dto.IDParam
	if !middleware.BindUri(c, &params) {
		return
	}
	if !middleware.RequirePositiveID(c, params.ID) {
		return
	}
	data, err := h.svc.GetClientMoUDownloadURL(c.Request.Context(), params.ID)
	if err != nil {
		respondError(c, err)
		return
	}
	respondData(c, http.StatusOK, data, "Success", nil)
}

func (h *ClientHandler) GetByContactNumber(c *gin.Context) {
	var query dto.ContactNumberQuery
	if !middleware.BindQuery(c, &query) {
		return
	}
	data, err := h.svc.GetClientByContactNumber(query.ContactNumber)
	if err != nil {
		respondError(c, err)
		return
	}
	respondData(c, http.StatusOK, data, "Success", nil)
}

func (h *ClientHandler) Create(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		respondError(c, apperrors.NewUnauthorized("Authentication required", nil))
		return
	}
	if isMultipartForm(c) {
		req, mou, err := dto.ParseClientMultipartCreate(c)
		if err != nil {
			respondError(c, apperrors.NewBadRequest(err.Error(), err))
			return
		}
		client := req.ToDomain()
		if err := h.svc.CreateClientWithMoU(c.Request.Context(), &client, userID, mou, req.Brands); err != nil {
			respondError(c, err)
			return
		}
		respondData(c, http.StatusCreated, client, "Client created successfully", nil)
		return
	}
	var req dto.ClientRequest
	if !middleware.BindJSON(c, &req) {
		return
	}
	if err := dto.ValidateClientBrandNames(req.Brands); err != nil {
		respondError(c, apperrors.NewBadRequest(err.Error(), err))
		return
	}
	client := req.ToDomain()
	if err := h.svc.CreateClient(&client, userID, req.Brands); err != nil {
		respondError(c, err)
		return
	}
	respondData(c, http.StatusCreated, client, "Client created successfully", nil)
}

func (h *ClientHandler) Update(c *gin.Context) {
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
	if isMultipartForm(c) {
		update, mou, err := dto.ParseClientMultipartUpdate(c)
		if err != nil {
			respondError(c, apperrors.NewBadRequest(err.Error(), err))
			return
		}
		hasFile := mou != nil
		hasFields := update != nil && update.HasAtLeastOneField()
		if !hasFile && !hasFields {
			respondError(c, apperrors.NewBadRequest("At least one field or mou_document is required", nil))
			return
		}
		client, err := h.svc.UpdateClientWithMoU(c.Request.Context(), params.ID, update, userID, mou)
		if err != nil {
			respondError(c, err)
			return
		}
		respondData(c, http.StatusOK, client, "Client updated successfully", nil)
		return
	}
	var req dto.ClientUpdateRequest
	if !middleware.BindJSON(c, &req) {
		return
	}
	if req.Brands != nil {
		if err := dto.ValidateClientBrandNames(*req.Brands); err != nil {
			respondError(c, apperrors.NewBadRequest(err.Error(), err))
			return
		}
	}
	if !req.HasAtLeastOneField() {
		respondError(c, apperrors.NewBadRequest("At least one field is required in the payload to update", nil))
		return
	}
	client, err := h.svc.UpdateClient(params.ID, &req, userID)
	if err != nil {
		respondError(c, err)
		return
	}
	respondData(c, http.StatusOK, client, "Client updated successfully", nil)
}

func isMultipartForm(c *gin.Context) bool {
	ct := c.ContentType()
	if ct == "" {
		return false
	}
	base := strings.ToLower(strings.TrimSpace(strings.Split(ct, ";")[0]))
	return base == "multipart/form-data"
}

func (h *ClientHandler) Delete(c *gin.Context) {
	var params dto.IDParam
	if !middleware.BindUri(c, &params) {
		return
	}
	if !middleware.RequirePositiveID(c, params.ID) {
		return
	}
	if err := h.svc.DeleteClient(params.ID); err != nil {
		respondError(c, err)
		return
	}
	respondMessage(c, http.StatusOK, "Client deleted successfully")
}
