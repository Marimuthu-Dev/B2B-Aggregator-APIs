package handlers

import (
	"net/http"
	"strconv"

	"b2b-diagnostic-aggregator/apis/internal/models"
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
	data, err := h.svc.GetAllLabs()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data, "message": "Success"})
}

func (h *LabHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid ID"})
		return
	}
	data, err := h.svc.GetLabByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "Lab not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data, "message": "Success"})
}

func (h *LabHandler) GetByContactNumber(c *gin.Context) {
	contactNumber := c.Query("contactNumber")
	data, err := h.svc.GetLabByContactNumber(contactNumber)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "Lab not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data, "message": "Success"})
}

func (h *LabHandler) Create(c *gin.Context) {
	var lab models.Lab
	if err := c.ShouldBindJSON(&lab); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}
	if err := h.svc.CreateLab(&lab); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": lab, "message": "Lab created successfully"})
}

func (h *LabHandler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid ID"})
		return
	}
	var lab models.Lab
	if err := c.ShouldBindJSON(&lab); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}
	if err := h.svc.UpdateLab(id, &lab); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": lab, "message": "Lab updated successfully"})
}

func (h *LabHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid ID"})
		return
	}
	if err := h.svc.DeleteLab(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Lab deleted successfully"})
}
