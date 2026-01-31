package handlers

import (
	"net/http"
	"strconv"

	"b2b-diagnostic-aggregator/apis/internal/models"
	"b2b-diagnostic-aggregator/apis/internal/service"
	"github.com/gin-gonic/gin"
)

type PackageHandler struct {
	svc service.PackageService
}

func NewPackageHandler(svc service.PackageService) *PackageHandler {
	return &PackageHandler{svc: svc}
}

func (h *PackageHandler) GetAll(c *gin.Context) {
	packages, err := h.svc.GetAllPackages()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": packages, "count": len(packages)})
}

func (h *PackageHandler) GetByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid ID"})
		return
	}

	pkg, err := h.svc.GetPackageByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "Package not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": pkg})
}

func (h *PackageHandler) Create(c *gin.Context) {
	var pkg models.Package
	if err := c.ShouldBindJSON(&pkg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}

	if err := h.svc.CreatePackage(&pkg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": pkg, "message": "Package created successfully"})
}

func (h *PackageHandler) Delete(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid ID"})
		return
	}

	if err := h.svc.DeletePackage(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Package deleted successfully"})
}

type CreatePackageWithTestsRequest struct {
	models.Package
	TestIDs []int `json:"testIds" binding:"required"`
}

func (h *PackageHandler) CreateWithTests(c *gin.Context) {
	var req CreatePackageWithTestsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}

	if err := h.svc.CreatePackageWithTests(&req.Package, req.TestIDs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    req.Package,
		"message": "Package created successfully with test mappings",
	})
}
