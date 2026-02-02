package handlers

import (
	"net/http"

	"b2b-diagnostic-aggregator/apis/internal/dto"
	"b2b-diagnostic-aggregator/apis/internal/middleware"
	"b2b-diagnostic-aggregator/apis/internal/repository"
	"b2b-diagnostic-aggregator/apis/internal/service"

	"github.com/gin-gonic/gin"
)

type LeadHandler struct {
	svc service.LeadService
}

func NewLeadHandler(svc service.LeadService) *LeadHandler {
	return &LeadHandler{svc: svc}
}

func (h *LeadHandler) GetAll(c *gin.Context) {
	var query dto.LeadListQuery
	if !middleware.BindQuery(c, &query) {
		return
	}
	page := query.PaginationQuery.Normalize("createdOn")
	filter := repository.LeadListFilter{
		Page:      page.Page,
		PageSize:  page.PageSize,
		SortBy:    page.SortBy,
		SortOrder: page.SortOrder,
		ClientID:  query.ClientID,
		StatusID:  query.StatusID,
		PackageID: query.PackageID,
	}

	data, total, err := h.svc.ListLeads(filter)
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

func (h *LeadHandler) GetByID(c *gin.Context) {
	var params dto.IDParam
	if !middleware.BindUri(c, &params) {
		return
	}
	data, err := h.svc.GetLeadByID(params.ID)
	if err != nil {
		respondError(c, err)
		return
	}
	respondData(c, http.StatusOK, data, "Success", nil)
}

func (h *LeadHandler) Create(c *gin.Context) {
	var req dto.LeadRequest
	if !middleware.BindJSON(c, &req) {
		return
	}
	lead := req.ToDomain()
	if err := h.svc.CreateLead(&lead); err != nil {
		respondError(c, err)
		return
	}
	respondData(c, http.StatusCreated, lead, "Lead created successfully", nil)
}

func (h *LeadHandler) Update(c *gin.Context) {
	var params dto.IDParam
	if !middleware.BindUri(c, &params) {
		return
	}
	var req dto.LeadRequest
	if !middleware.BindJSON(c, &req) {
		return
	}
	lead := req.ToDomain()
	if err := h.svc.UpdateLead(params.ID, &lead); err != nil {
		respondError(c, err)
		return
	}
	respondData(c, http.StatusOK, lead, "Lead updated successfully", nil)
}

func (h *LeadHandler) Delete(c *gin.Context) {
	var params dto.IDParam
	if !middleware.BindUri(c, &params) {
		return
	}
	// In Go version, we should probably get actorID from context (set by AuthMiddleware)
	actorIDInterface, _ := c.Get("userId")
	actorID, _ := actorIDInterface.(int64)

	if err := h.svc.DeleteLead(params.ID, actorID); err != nil {
		respondError(c, err)
		return
	}
	respondMessage(c, http.StatusOK, "Lead deleted successfully")
}

func (h *LeadHandler) BulkUpdateStatus(c *gin.Context) {
	var req dto.BulkUpdateLeadStatusRequest
	if !middleware.BindJSON(c, &req) {
		return
	}
	if err := h.svc.BulkUpdateLeadStatus(req.LeadIDs, req.LeadStatusID, req.LastUpdatedBy); err != nil {
		respondError(c, err)
		return
	}
	respondMessage(c, http.StatusOK, "Lead statuses updated successfully")
}
