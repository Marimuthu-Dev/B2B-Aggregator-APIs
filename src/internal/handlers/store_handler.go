package handlers

import (
	"net/http"

	"b2b-diagnostic-aggregator/apis/internal/apperrors"
	"b2b-diagnostic-aggregator/apis/internal/domain"
	"b2b-diagnostic-aggregator/apis/internal/dto"
	"b2b-diagnostic-aggregator/apis/internal/middleware"
	persistencemodels "b2b-diagnostic-aggregator/apis/internal/persistence/models"
	"b2b-diagnostic-aggregator/apis/internal/repository"
	"b2b-diagnostic-aggregator/apis/internal/service"
	"b2b-diagnostic-aggregator/apis/pkg/utils"

	"github.com/gin-gonic/gin"
)

type StoreHandler struct {
	svc service.StoreService
}

func NewStoreHandler(svc service.StoreService) *StoreHandler {
	return &StoreHandler{svc: svc}
}

func requireStoreMaster(c *gin.Context) bool {
	if persistencemodels.HasStoreMasterTable() {
		return true
	}
	respondError(c, apperrors.NewNotFound("Store master is not available for this environment", nil))
	return false
}

func (h *StoreHandler) GetAll(c *gin.Context) {
	if !requireStoreMaster(c) {
		return
	}
	var query dto.StoreListQuery
	if !middleware.BindQuery(c, &query) {
		return
	}
	userType, userID, ok := storeCaller(c)
	if !ok {
		respondError(c, apperrors.NewUnauthorized("Authentication required", nil))
		return
	}
	scope, err := resolveStoreListScope(userType, userID, query.ClientID)
	if err != nil {
		respondError(c, err)
		return
	}

	page := query.PaginationQuery.Normalize("createdOn", 0)
	filter := repository.StoreListFilter{
		Page:      page.Page,
		PageSize:  page.PageSize,
		SortBy:    page.SortBy,
		SortOrder: page.SortOrder,
		ClientID:  scope.ClientID,
		StoreID:   scope.StoreID,
		IsActive:  query.IsActive,
		Search:    query.Search,
	}
	data, total, err := h.svc.ListStores(filter)
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

func (h *StoreHandler) GetByID(c *gin.Context) {
	if !requireStoreMaster(c) {
		return
	}
	var params dto.IDParam
	if !middleware.BindUri(c, &params) {
		return
	}
	if !middleware.RequirePositiveID(c, params.ID) {
		return
	}
	userType, userID, ok := storeCaller(c)
	if !ok {
		respondError(c, apperrors.NewUnauthorized("Authentication required", nil))
		return
	}
	switch userType {
	case utils.UserTypeEmployee, utils.UserTypeClient, utils.UserTypeStore:
	default:
		respondError(c, apperrors.NewForbidden("You are not authorized for this activity.", nil))
		return
	}
	data, err := h.svc.GetStoreByID(params.ID)
	if err != nil {
		respondError(c, err)
		return
	}
	if !storeReadableByCaller(userType, userID, data) {
		respondError(c, apperrors.NewNotFound("Store not found", nil))
		return
	}
	respondData(c, http.StatusOK, data, "Success", nil)
}

func (h *StoreHandler) Create(c *gin.Context) {
	if !requireStoreMaster(c) {
		return
	}
	userType, userID, ok := storeCaller(c)
	if !ok {
		respondError(c, apperrors.NewUnauthorized("Authentication required", nil))
		return
	}
	if userType != utils.UserTypeEmployee && userType != utils.UserTypeClient {
		respondError(c, apperrors.NewForbidden("You are not authorized for this activity.", nil))
		return
	}
	var req dto.StoreRequest
	if !middleware.BindJSON(c, &req) {
		return
	}
	if userType == utils.UserTypeClient && req.ClientID != userID {
		respondError(c, apperrors.NewForbidden("You can only add stores for your own client ID.", nil))
		return
	}
	store := req.ToDomain()
	if err := h.svc.CreateStore(&store, userID, req.Password); err != nil {
		respondError(c, err)
		return
	}
	respondData(c, http.StatusCreated, store, "Store created successfully", nil)
}

func (h *StoreHandler) Update(c *gin.Context) {
	if !requireStoreMaster(c) {
		return
	}
	userType, userID, ok := storeCaller(c)
	if !ok {
		respondError(c, apperrors.NewUnauthorized("Authentication required", nil))
		return
	}
	if userType != utils.UserTypeEmployee && userType != utils.UserTypeClient {
		respondError(c, apperrors.NewForbidden("You are not authorized for this activity.", nil))
		return
	}
	var params dto.IDParam
	if !middleware.BindUri(c, &params) {
		return
	}
	if !middleware.RequirePositiveID(c, params.ID) {
		return
	}
	if userType == utils.UserTypeClient {
		store, err := h.svc.GetStoreByID(params.ID)
		if err != nil {
			respondError(c, err)
			return
		}
		if !storeReadableByCaller(userType, userID, store) {
			respondError(c, apperrors.NewForbidden("You are not authorized to update this store.", nil))
			return
		}
	}
	var req dto.StoreUpdateRequest
	if !middleware.BindJSON(c, &req) {
		return
	}
	if !req.HasAtLeastOneField() {
		respondError(c, apperrors.NewBadRequest("At least one field is required in the payload to update", nil))
		return
	}
	store, err := h.svc.UpdateStore(params.ID, &req, userID)
	if err != nil {
		respondError(c, err)
		return
	}
	respondData(c, http.StatusOK, store, "Store updated successfully", nil)
}

func storeCaller(c *gin.Context) (userType int, userID int64, ok bool) {
	userID, idOK := middleware.GetUserID(c)
	userType, typeOK := middleware.GetUserType(c)
	return userType, userID, idOK && typeOK && userID > 0
}

type storeListScope struct {
	ClientID *int64
	StoreID  *int64
}

// resolveStoreListScope applies GET /stores visibility:
// userType 1 (employee): optional clientId; omitted means all stores
// userType 2 (client): JWT userId is always ClientID (query clientId cannot widen or switch tenants)
// userType 3 (lab): forbidden
// userType 4 (store): JWT userId is always StoreID
func resolveStoreListScope(userType int, userID int64, queryClientID *int64) (storeListScope, error) {
	switch userType {
	case utils.UserTypeEmployee:
		return storeListScope{ClientID: queryClientID}, nil
	case utils.UserTypeClient:
		id := userID
		return storeListScope{ClientID: &id}, nil
	case utils.UserTypeLab:
		return storeListScope{}, apperrors.NewForbidden("You are not authorized for this activity.", nil)
	case utils.UserTypeStore:
		id := userID
		return storeListScope{StoreID: &id}, nil
	default:
		return storeListScope{}, apperrors.NewForbidden("You are not authorized for this activity.", nil)
	}
}

func storeReadableByCaller(userType int, userID int64, store *domain.Store) bool {
	if store == nil {
		return false
	}
	switch userType {
	case utils.UserTypeEmployee:
		return true
	case utils.UserTypeClient:
		return store.ClientID == userID
	case utils.UserTypeStore:
		return store.StoreID == userID
	default:
		return false
	}
}
