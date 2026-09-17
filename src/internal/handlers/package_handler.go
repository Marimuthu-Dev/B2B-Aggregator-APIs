package handlers

import (
	"net/http"
	"strconv"

	"b2b-diagnostic-aggregator/apis/internal/apperrors"
	"b2b-diagnostic-aggregator/apis/internal/dto"
	"b2b-diagnostic-aggregator/apis/internal/middleware"
	"b2b-diagnostic-aggregator/apis/internal/repository"
	"b2b-diagnostic-aggregator/apis/internal/service"
	"b2b-diagnostic-aggregator/apis/pkg/utils"

	"github.com/gin-gonic/gin"
)

type PackageHandler struct {
	svc      service.PackageService
	storeSvc service.StoreService
}

func NewPackageHandler(svc service.PackageService, storeSvc service.StoreService) *PackageHandler {
	return &PackageHandler{svc: svc, storeSvc: storeSvc}
}

func (h *PackageHandler) GetAll(c *gin.Context) {
	var query dto.PackageListQuery
	if !middleware.BindQuery(c, &query) {
		return
	}
	page := query.PaginationQuery.Normalize("createdOn", 0)
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
		"Count":    len(packages),
		"Page":     filter.Page,
		"PageSize": filter.PageSize,
		"Total":    total,
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
	userID, ok := middleware.GetUserID(c)
	if !ok {
		respondError(c, apperrors.NewUnauthorized("Authentication required", nil))
		return
	}
	var req dto.PackageRequest
	if !middleware.BindJSON(c, &req) {
		return
	}
	pkg := req.ToDomain()
	if err := h.svc.CreatePackage(&pkg, userID); err != nil {
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
	userID, ok := middleware.GetUserID(c)
	if !ok {
		respondError(c, apperrors.NewUnauthorized("Authentication required", nil))
		return
	}
	var req dto.CreatePackageWithTestsRequest
	if !middleware.BindJSON(c, &req) {
		return
	}
	pkg := req.PackageRequest.ToDomain()
	result, err := h.svc.CreatePackageWithTests(&pkg, req.TestIDs, userID)
	if err != nil {
		respondError(c, err)
		return
	}

	if result.RetVal == 2 {
		data := gin.H{
			"PackageID": result.Package.PackageID,
			"PackageName": result.Package.PackageName,
			"Description": result.Package.Description,
			"IsActive": result.Package.IsActive,
			"CreatedBy": result.Package.CreatedBy,
			"CreatedOn": result.Package.CreatedOn,
			"LastUpdatedBy": result.Package.LastUpdatedBy,
			"LastUpdatedOn": result.Package.LastUpdatedOn,
			"TestCount": len(result.TestIDs),
			"TestIDs": result.TestIDs,
		}
		respondData(c, http.StatusOK, data, result.Message, nil)
		return
	}
	data := gin.H{
		"PackageID": result.Package.PackageID,
		"PackageName": result.Package.PackageName,
		"Description": result.Package.Description,
		"IsActive": result.Package.IsActive,
		"CreatedBy": result.Package.CreatedBy,
		"CreatedOn": result.Package.CreatedOn,
		"LastUpdatedBy": result.Package.LastUpdatedBy,
		"LastUpdatedOn": result.Package.LastUpdatedOn,
		"TestCount": len(result.TestIDs),
		"TestIDs": result.TestIDs,
	}
	respondData(c, http.StatusCreated, data, result.Message, nil)
}

func (h *PackageHandler) GetAllWithTestsDetails(c *gin.Context) {
	data, err := h.svc.GetAllPackagesWithTestsDetails()
	if err != nil {
		respondError(c, err)
		return
	}
	respondData(c, http.StatusOK, data, "Packages retrieved successfully with test details", gin.H{"Count": len(data)})
}

func (h *PackageHandler) UpdatePackageStatus(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		respondError(c, apperrors.NewUnauthorized("Authentication required", nil))
		return
	}
	var params dto.PackageIDParam
	if !middleware.BindUri(c, &params) {
		return
	}
	if !middleware.RequirePositiveID(c, params.ID) {
		return
	}
	var req dto.PackageStatusUpdateRequest
	if !middleware.BindJSON(c, &req) {
		return
	}
	result, err := h.svc.UpdatePackageStatus(params.ID, req.IsActive, userID)
	if err != nil {
		respondError(c, err)
		return
	}
	msg := "Package status updated successfully. " +
		formatInt(result.UpdatedTestMappingsCount) + " test mapping(s), " +
		formatInt(result.UpdatedClientMappingsCount) + " client mapping(s), " +
		formatInt(result.UpdatedLabMappingsCount) + " lab mapping(s) also updated. " +
		"Total: " + formatInt(result.TotalMappingsUpdated) + " mapping(s)."
	respondData(c, http.StatusOK, result.Package, msg, nil)
}

func formatInt(n int) string {
	return strconv.Itoa(n)
}

func (h *PackageHandler) CreatePackageClientMapping(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		respondError(c, apperrors.NewUnauthorized("Authentication required", nil))
		return
	}
	var req dto.PackageClientMappingRequest
	if !middleware.BindJSON(c, &req) {
		return
	}
	result, err := h.svc.CreatePackageClientMapping(req.PackageID, req.ClientID, req.Price, userID, userID)
	if err != nil {
		respondError(c, err)
		return
	}
	if result.RetVal == 2 {
		respondData(c, http.StatusOK, result.Mapping, result.Message, nil)
		return
	}
	respondData(c, http.StatusCreated, result.Mapping, result.Message, nil)
}

func (h *PackageHandler) GetAllPackageClientMappings(c *gin.Context) {
	userType, typeOK := middleware.GetUserType(c)
	if !typeOK {
		respondError(c, apperrors.NewUnauthorized("Authentication required", nil))
		return
	}
	if userType == utils.UserTypeLab {
		respondError(c, apperrors.NewForbidden("You are not authorized for this activity.", nil))
		return
	}

	userID, idOK := middleware.GetUserID(c)
	var clientID *int64

	switch userType {
	case utils.UserTypeClient:
		if !idOK || userID <= 0 {
			respondError(c, apperrors.NewUnauthorized("Authentication required", nil))
			return
		}
		clientID = &userID
	case utils.UserTypeStore:
		if !idOK || userID <= 0 {
			respondError(c, apperrors.NewUnauthorized("Authentication required", nil))
			return
		}
		store, err := h.storeSvc.GetStoreByID(userID)
		if err != nil {
			respondError(c, err)
			return
		}
		clientID = &store.ClientID
	case utils.UserTypeEmployee:
		var query dto.PackageClientMappingListQuery
		if !middleware.BindQuery(c, &query) {
			return
		}
		clientID = query.ClientID
	default:
		respondError(c, apperrors.NewForbidden("You are not authorized for this activity.", nil))
		return
	}

	data, err := h.svc.GetAllPackageClientMappings(clientID)
	if err != nil {
		respondError(c, err)
		return
	}
	respondData(c, http.StatusOK, data, "Package-Client mappings retrieved successfully", gin.H{"Count": len(data)})
}

func (h *PackageHandler) UpdatePackageClientMappingStatus(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		respondError(c, apperrors.NewUnauthorized("Authentication required", nil))
		return
	}
	var params dto.PackageMappingIDParam
	if !middleware.BindUri(c, &params) {
		return
	}
	if !middleware.RequirePositiveID(c, params.ID) {
		return
	}
	var req dto.PackageMappingStatusUpdateRequest
	if !middleware.BindJSON(c, &req) {
		return
	}
	if req.IsActive == nil {
		respondError(c, apperrors.NewBadRequest("IsActive is required", nil))
		return
	}
	result, err := h.svc.UpdatePackageClientMappingStatus(params.ID, *req.IsActive, userID)
	if err != nil {
		respondError(c, err)
		return
	}
	respondData(c, http.StatusOK, result.Mapping, result.Message, nil)
}

func (h *PackageHandler) CreatePackageLabMapping(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		respondError(c, apperrors.NewUnauthorized("Authentication required", nil))
		return
	}
	var req dto.PackageLabMappingRequest
	if !middleware.BindJSON(c, &req) {
		return
	}
	result, err := h.svc.CreatePackageLabMapping(req.PackageID, req.LabID, req.Price, userID, userID)
	if err != nil {
		respondError(c, err)
		return
	}
	if result.RetVal == 2 {
		respondData(c, http.StatusOK, result.Mapping, result.Message, nil)
		return
	}
	respondData(c, http.StatusCreated, result.Mapping, result.Message, nil)
}

func (h *PackageHandler) GetAllPackageLabMappings(c *gin.Context) {
	userType, typeOK := middleware.GetUserType(c)
	if !typeOK {
		respondError(c, apperrors.NewUnauthorized("Authentication required", nil))
		return
	}

	userID, idOK := middleware.GetUserID(c)

	var query dto.PackageLabMappingListQuery
	if !middleware.BindQuery(c, &query) {
		return
	}

	var labID *int64
	switch userType {
	case utils.UserTypeLab:
		if !idOK || userID <= 0 {
			respondError(c, apperrors.NewUnauthorized("Authentication required", nil))
			return
		}
		labID = &userID
	case utils.UserTypeEmployee:
		labID = query.LabID
	default:
		respondError(c, apperrors.NewForbidden("You are not authorized for this activity.", nil))
		return
	}

	isActive := true
	if query.IsActive != nil {
		isActive = *query.IsActive
	}
	filter := repository.PackageLabMappingListFilter{
		PackageID: query.PackageID,
		LabID:     labID,
		IsActive:  isActive,
	}
	data, err := h.svc.GetAllPackageLabMappings(filter)
	if err != nil {
		respondError(c, err)
		return
	}
	respondData(c, http.StatusOK, data, "Package-Lab mappings retrieved successfully", gin.H{"Count": len(data)})
}

func (h *PackageHandler) UpdatePackageLabMappingStatus(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		respondError(c, apperrors.NewUnauthorized("Authentication required", nil))
		return
	}
	var params dto.PackageMappingIDParam
	if !middleware.BindUri(c, &params) {
		return
	}
	if !middleware.RequirePositiveID(c, params.ID) {
		return
	}
	var req dto.PackageMappingStatusUpdateRequest
	if !middleware.BindJSON(c, &req) {
		return
	}
	if req.IsActive == nil {
		respondError(c, apperrors.NewBadRequest("IsActive is required", nil))
		return
	}
	result, err := h.svc.UpdatePackageLabMappingStatus(params.ID, *req.IsActive, userID)
	if err != nil {
		respondError(c, err)
		return
	}
	respondData(c, http.StatusOK, result.Mapping, result.Message, nil)
}
