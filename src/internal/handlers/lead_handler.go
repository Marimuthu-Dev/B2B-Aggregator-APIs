package handlers

import (
	"net/http"
	"strconv"

	"b2b-diagnostic-aggregator/apis/internal/models"
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
	data, err := h.svc.GetAllLeads()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data, "message": "Success"})
}

func (h *LeadHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid ID"})
		return
	}
	data, err := h.svc.GetLeadByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "Lead not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data, "message": "Success"})
}

func (h *LeadHandler) Create(c *gin.Context) {
	var lead models.Lead
	if err := c.ShouldBindJSON(&lead); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}
	if err := h.svc.CreateLead(&lead); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": lead, "message": "Lead created successfully"})
}

func (h *LeadHandler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid ID"})
		return
	}
	var lead models.Lead
	if err := c.ShouldBindJSON(&lead); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}
	if err := h.svc.UpdateLead(id, &lead); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": lead, "message": "Lead updated successfully"})
}

func (h *LeadHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid ID"})
		return
	}
	// In Go version, we should probably get actorID from context (set by AuthMiddleware)
	actorIDInterface, _ := c.Get("userId")
	actorID, _ := actorIDInterface.(int64)

	if err := h.svc.DeleteLead(id, actorID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Lead deleted successfully"})
}

type BulkUpdateLeadStatusRequest struct {
	LeadIDs       []int64 `json:"leadIds" binding:"required"`
	LeadStatusID  int8    `json:"leadStatusId" binding:"required"`
	LastUpdatedBy int64   `json:"lastUpdatedBy" binding:"required"`
}

func (h *LeadHandler) BulkUpdateStatus(c *gin.Context) {
	var req BulkUpdateLeadStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}
	if err := h.svc.BulkUpdateLeadStatus(req.LeadIDs, req.LeadStatusID, req.LastUpdatedBy); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Lead statuses updated successfully"})
}
