package handlers

import (
	"net/http"

	"b2b-diagnostic-aggregator/apis/internal/dto"
	"b2b-diagnostic-aggregator/apis/internal/middleware"
	"b2b-diagnostic-aggregator/apis/internal/repository"
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
	var query dto.PackageListQuery
	if !middleware.BindQuery(c, &query) {
		return
	}
	page := query.PaginationQuery.Normalize("createdOn")
	filter := repository.PackageListFilter{
		Page:      page.Page,
		PageSize:  page.PageSize,
		SortBy:    page.SortBy,
		SortOrder: page.SortOrder,
		IsActive:  query.IsActive,
		Search:    query.Search,
	}

	packages, total, err := h.svc.ListPackages(filter)
	if err != nil {
		respondError(c, err)
		return
	}
	respondData(c, http.StatusOK, packages, "", gin.H{
		"count":    len(packages),
		"page":     filter.Page,
		"pageSize": filter.PageSize,
		"total":    total,
	})
}

func (h *PackageHandler) GetByID(c *gin.Context) {
	var params dto.PackageIDParam
	if !middleware.BindUri(c, &params) {
		return
	}

	pkg, err := h.svc.GetPackageByID(params.ID)
	if err != nil {
		respondError(c, err)
		return
	}
	respondData(c, http.StatusOK, pkg, "", nil)
}

func (h *PackageHandler) Create(c *gin.Context) {
	var req dto.PackageRequest
	if !middleware.BindJSON(c, &req) {
		return
	}
	pkg := req.ToDomain()

	if err := h.svc.CreatePackage(&pkg); err != nil {
		respondError(c, err)
		return
	}
	respondData(c, http.StatusCreated, pkg, "Package created successfully", nil)
}

func (h *PackageHandler) Delete(c *gin.Context) {
	var params dto.PackageIDParam
	if !middleware.BindUri(c, &params) {
		return
	}

	if err := h.svc.DeletePackage(params.ID); err != nil {
		respondError(c, err)
		return
	}
	respondMessage(c, http.StatusOK, "Package deleted successfully")
}

func (h *PackageHandler) CreateWithTests(c *gin.Context) {
	var req dto.CreatePackageWithTestsRequest
	if !middleware.BindJSON(c, &req) {
		return
	}

	pkg := req.PackageRequest.ToDomain()
	if err := h.svc.CreatePackageWithTests(&pkg, req.TestIDs); err != nil {
		respondError(c, err)
		return
	}

	respondData(c, http.StatusCreated, pkg, "Package created successfully with test mappings", nil)
}
