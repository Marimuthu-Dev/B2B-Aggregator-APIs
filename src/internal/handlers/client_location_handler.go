package handlers

import (
	"net/http"

	"b2b-diagnostic-aggregator/apis/internal/dto"
	"b2b-diagnostic-aggregator/apis/internal/middleware"
	"b2b-diagnostic-aggregator/apis/internal/service"

	"github.com/gin-gonic/gin"
)

type ClientLocationHandler struct {
	svc service.ClientLocationService
}

func NewClientLocationHandler(svc service.ClientLocationService) *ClientLocationHandler {
	return &ClientLocationHandler{svc: svc}
}

func (h *ClientLocationHandler) GetAllByClientID(c *gin.Context) {
	var params dto.ClientIDPathParam
	if !middleware.BindUri(c, &params) {
		return
	}
	data, err := h.svc.GetByClientID(params.ClientID)
	if err != nil {
		respondError(c, err)
		return
	}
	respondData(c, http.StatusOK, data, "Success", gin.H{"count": len(data)})
}

func (h *ClientLocationHandler) GetByID(c *gin.Context) {
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

func (h *ClientLocationHandler) Create(c *gin.Context) {
	var pathParams dto.ClientIDPathParam
	if !middleware.BindUri(c, &pathParams) {
		return
	}
	var req dto.ClientLocationRequest
	if !middleware.BindJSON(c, &req) {
		return
	}
	loc := req.ToDomain(pathParams.ClientID)
	if err := h.svc.Create(&loc); err != nil {
		respondError(c, err)
		return
	}
	respondData(c, http.StatusCreated, loc, "Client location created successfully", nil)
}

func (h *ClientLocationHandler) Update(c *gin.Context) {
	var params dto.IDParam
	if !middleware.BindUri(c, &params) {
		return
	}
	existing, err := h.svc.GetByID(params.ID)
	if err != nil {
		respondError(c, err)
		return
	}
	var req dto.ClientLocationRequest
	if !middleware.BindJSON(c, &req) {
		return
	}
	loc := req.ToDomain(existing.ClientID)
	loc.ClientLocationID = params.ID
	if err := h.svc.Update(params.ID, &loc); err != nil {
		respondError(c, err)
		return
	}
	respondData(c, http.StatusOK, loc, "Client location updated successfully", nil)
}

func (h *ClientLocationHandler) Delete(c *gin.Context) {
	var params dto.IDParam
	if !middleware.BindUri(c, &params) {
		return
	}
	if err := h.svc.Delete(params.ID); err != nil {
		respondError(c, err)
		return
	}
	respondMessage(c, http.StatusOK, "Client location deleted successfully")
}
