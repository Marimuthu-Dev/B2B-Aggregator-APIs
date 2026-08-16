package handlers

import (
	"net/http"

	"b2b-diagnostic-aggregator/apis/internal/apperrors"
	"b2b-diagnostic-aggregator/apis/internal/domain"
	"b2b-diagnostic-aggregator/apis/internal/dto"
	"b2b-diagnostic-aggregator/apis/internal/middleware"
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

func (h *StoreHandler) GetAll(c *gin.Context) {
	var query dto.StoreListQuery
	if !middleware.BindQuery(c, &query) {
		return
	}
	userType, userID, ok := storeCaller(c)
	if !ok {
		respondError(c, apperrors.NewUnauthorized("Authentication required", nil))
		return
	}
	var storeID *int64
	switch userType {
	case utils.UserTypeEmployee:
		// unscoped; optional clientId from query
	case utils.UserTypeClient:
		query.ClientID = &userID
	case utils.UserTypeStore:
		query.ClientID = nil
		storeID = &userID
	default:
		respondError(c, apperrors.NewForbidden("You are not authorized for this activity.", nil))
		return
	}

	page := query.PaginationQuery.Normalize("createdOn", 0)
	filter := repository.StoreListFilter{
		Page:      page.Page,
		PageSize:  page.PageSize,
		SortBy:    page.SortBy,
		SortOrder: page.SortOrder,
		ClientID:  query.ClientID,
		StoreID:   storeID,
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
	if userType != utils.UserTypeEmployee && userType != utils.UserTypeClient && userType != utils.UserTypeStore {
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
	userType, userID, ok := storeCaller(c)
	if !ok {
		respondError(c, apperrors.NewUnauthorized("Authentication required", nil))
		return
	}
	if userType != utils.UserTypeEmployee {
		respondError(c, apperrors.NewForbidden("You are not authorized for this activity.", nil))
		return
	}
	var req dto.StoreRequest
	if !middleware.BindJSON(c, &req) {
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
	userType, userID, ok := storeCaller(c)
	if !ok {
		respondError(c, apperrors.NewUnauthorized("Authentication required", nil))
		return
	}
	if userType != utils.UserTypeEmployee {
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
