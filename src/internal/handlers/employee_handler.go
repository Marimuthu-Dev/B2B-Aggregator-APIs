package handlers

import (
	"net/http"

	"b2b-diagnostic-aggregator/apis/internal/apperrors"
	"b2b-diagnostic-aggregator/apis/internal/dto"
	"b2b-diagnostic-aggregator/apis/internal/middleware"
	"b2b-diagnostic-aggregator/apis/internal/service"

	"github.com/gin-gonic/gin"
)

type EmployeeHandler struct {
	svc service.EmployeeService
}

func NewEmployeeHandler(svc service.EmployeeService) *EmployeeHandler {
	return &EmployeeHandler{svc: svc}
}

func (h *EmployeeHandler) GetAll(c *gin.Context) {
	data, err := h.svc.GetAll()
	if err != nil {
		respondError(c, err)
		return
	}
	respondData(c, http.StatusOK, data, "Success", gin.H{"count": len(data)})
}

func (h *EmployeeHandler) GetByID(c *gin.Context) {
	var params dto.IDParam
	if !middleware.BindUri(c, &params) {
		return
	}
	data, err := h.svc.GetByID(params.ID)
	if err != nil {
		respondError(c, err)
		return
	}
	respondData(c, http.StatusOK, data, "Success", nil)
}

func (h *EmployeeHandler) GetByContactNumber(c *gin.Context) {
	var query dto.ContactNumberQuery
	if !middleware.BindQuery(c, &query) {
		return
	}
	data, err := h.svc.GetByContactNumber(query.ContactNumber)
	if err != nil {
		respondError(c, err)
		return
	}
	respondData(c, http.StatusOK, data, "Success", nil)
}

func (h *EmployeeHandler) Create(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		respondError(c, apperrors.NewUnauthorized("Authentication required", nil))
		return
	}
	var req dto.EmployeeRequest
	if !middleware.BindJSON(c, &req) {
		return
	}
	emp := req.ToDomain()
	if err := h.svc.Create(&emp, userID); err != nil {
		respondError(c, err)
		return
	}
	respondData(c, http.StatusCreated, emp, "Employee created successfully", nil)
}

func (h *EmployeeHandler) Update(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		respondError(c, apperrors.NewUnauthorized("Authentication required", nil))
		return
	}
	var params dto.IDParam
	if !middleware.BindUri(c, &params) {
		return
	}
	if !middleware.RequirePositiveID(c, params.ID) {
		return
	}
	var req dto.EmployeeUpdateRequest
	if !middleware.BindJSON(c, &req) {
		return
	}
	if !req.HasAtLeastOneField() {
		respondError(c, apperrors.NewBadRequest("At least one field is required in the payload to update", nil))
		return
	}
	emp, err := h.svc.Update(params.ID, &req, userID)
	if err != nil {
		respondError(c, err)
		return
	}
	respondData(c, http.StatusOK, emp, "Employee updated successfully", nil)
}

func (h *EmployeeHandler) Delete(c *gin.Context) {
	var params dto.IDParam
	if !middleware.BindUri(c, &params) {
		return
	}
	if !middleware.RequirePositiveID(c, params.ID) {
		return
	}
	if err := h.svc.Delete(params.ID); err != nil {
		respondError(c, err)
		return
	}
	respondMessage(c, http.StatusOK, "Employee deleted successfully")
}
