package handlers

import (
	"net/http"

	"b2b-diagnostic-aggregator/apis/internal/dto"
	"b2b-diagnostic-aggregator/apis/internal/middleware"
	"b2b-diagnostic-aggregator/apis/internal/repository"
	"b2b-diagnostic-aggregator/apis/internal/service"

	"github.com/gin-gonic/gin"
)

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
	page := query.PaginationQuery.Normalize("createdOn", 500) // default 500 so GET without params returns all labs
	filter := repository.LabListFilter{
		Page:      page.Page,
		PageSize:  page.PageSize,
		SortBy:    page.SortBy,
		SortOrder: page.SortOrder,
		CityID:    query.CityID,
		StateID:   query.StateID,
		IsActive:  query.IsActive,
	}

	data, total, err := h.svc.ListLabs(filter)
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

func (h *LabHandler) Create(c *gin.Context) {
	var req dto.LabRequest
	if !middleware.BindJSON(c, &req) {
		return
	}
	lab := req.ToDomain()
	if err := h.svc.CreateLab(&lab); err != nil {
		respondError(c, err)
		return
	}
	respondData(c, http.StatusCreated, lab, "Lab created successfully", nil)
}

func (h *LabHandler) Update(c *gin.Context) {
	var params dto.IDParam
	if !middleware.BindUri(c, &params) {
		return
	}
	var req dto.LabRequest
	if !middleware.BindJSON(c, &req) {
		return
	}
	lab := req.ToDomain()
	if err := h.svc.UpdateLab(params.ID, &lab); err != nil {
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
	if err := h.svc.DeleteLab(params.ID); err != nil {
		respondError(c, err)
		return
	}
	respondMessage(c, http.StatusOK, "Lab deleted successfully")
}
