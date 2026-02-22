package handlers

import (
	"net/http"
	"strconv"

	"b2b-diagnostic-aggregator/apis/internal/apperrors"
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
	page := query.PaginationQuery.Normalize("createdOn", 0)
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
	count, err := h.svc.BulkUpdateLeadStatus(req.LeadIDs, req.LeadStatusID, req.LastUpdatedBy)
	if err != nil {
		respondError(c, err)
		return
	}
	respondData(c, http.StatusOK, gin.H{"updatedCount": count}, "Lead statuses updated successfully", nil)
}

func (h *LeadHandler) BulkImportCsv(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil || file == nil {
		respondError(c, apperrors.NewBadRequest("CSV file is required", err))
		return
	}
	clientIDStr := c.PostForm("ClientID")
	packageIDStr := c.PostForm("PackageID")
	if clientIDStr == "" || packageIDStr == "" {
		respondError(c, apperrors.NewBadRequest("ClientID and PackageID are required in the request body", nil))
		return
	}
	clientID, err1 := strconv.ParseInt(clientIDStr, 10, 64)
	packageID, err2 := strconv.Atoi(packageIDStr)
	if err1 != nil || err2 != nil || clientID <= 0 || packageID <= 0 {
		respondError(c, apperrors.NewBadRequest("ClientID and PackageID must be positive integers", nil))
		return
	}

	f, err := file.Open()
	if err != nil {
		respondError(c, apperrors.NewBadRequest("Failed to read file", err))
		return
	}
	defer f.Close()
	buf := make([]byte, file.Size+1)
	n, _ := f.Read(buf)
	buf = buf[:n]

	inserted, err := h.svc.BulkImportFromCSV(buf, clientID, packageID)
	if err != nil {
		respondError(c, err)
		return
	}
	respondData(c, http.StatusCreated, gin.H{"insertedCount": inserted}, "Leads imported successfully", nil)
}
