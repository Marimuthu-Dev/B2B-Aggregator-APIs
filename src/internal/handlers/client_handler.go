package handlers

import (
	"net/http"

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
	page := query.PaginationQuery.Normalize("createdOn")
	filter := repository.ClientListFilter{
		Page:      page.Page,
		PageSize:  page.PageSize,
		SortBy:    page.SortBy,
		SortOrder: page.SortOrder,
		CityID:    query.CityID,
		StateID:   query.StateID,
		IsActive:  query.IsActive,
	}

	data, total, err := h.svc.ListClients(filter)
	if err != nil {
		respondError(c, err)
		return
	}
	respondData(c, http.StatusOK, data, "Success", gin.H{
		"count":    len(data),
		"page":     filter.Page,
		"pageSize": filter.PageSize,
		"total":    total,
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
	var req dto.ClientRequest
	if !middleware.BindJSON(c, &req) {
		return
	}
	client := req.ToDomain()
	if err := h.svc.CreateClient(&client); err != nil {
		respondError(c, err)
		return
	}
	respondData(c, http.StatusCreated, client, "Client created successfully", nil)
}

func (h *ClientHandler) Update(c *gin.Context) {
	var params dto.IDParam
	if !middleware.BindUri(c, &params) {
		return
	}
	var req dto.ClientRequest
	if !middleware.BindJSON(c, &req) {
		return
	}
	client := req.ToDomain()
	if err := h.svc.UpdateClient(params.ID, &client); err != nil {
		respondError(c, err)
		return
	}
	respondData(c, http.StatusOK, client, "Client updated successfully", nil)
}

func (h *ClientHandler) Delete(c *gin.Context) {
	var params dto.IDParam
	if !middleware.BindUri(c, &params) {
		return
	}
	if err := h.svc.DeleteClient(params.ID); err != nil {
		respondError(c, err)
		return
	}
	respondMessage(c, http.StatusOK, "Client deleted successfully")
}
