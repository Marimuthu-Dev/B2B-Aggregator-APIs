package handlers

import (
	"net/http"

	"b2b-diagnostic-aggregator/apis/internal/dto"
	"b2b-diagnostic-aggregator/apis/internal/middleware"
	"b2b-diagnostic-aggregator/apis/internal/service"

	"github.com/gin-gonic/gin"
)

type TestHandler struct {
	svc service.TestService
}

func NewTestHandler(svc service.TestService) *TestHandler {
	return &TestHandler{svc: svc}
}

func (h *TestHandler) GetAll(c *gin.Context) {
	data, err := h.svc.GetAllTests()
	if err != nil {
		respondError(c, err)
		return
	}
	respondData(c, http.StatusOK, data, "Tests retrieved successfully", gin.H{"count": len(data)})
}

func (h *TestHandler) GetActive(c *gin.Context) {
	data, err := h.svc.GetActiveTests()
	if err != nil {
		respondError(c, err)
		return
	}
	respondData(c, http.StatusOK, data, "Active tests retrieved successfully", gin.H{"count": len(data)})
}

func (h *TestHandler) GetByID(c *gin.Context) {
	var params dto.TestIDParam
	if !middleware.BindUri(c, &params) {
		return
	}
	data, err := h.svc.GetTestByID(params.ID)
	if err != nil {
		respondError(c, err)
		return
	}
	respondData(c, http.StatusOK, data, "Test retrieved successfully", nil)
}
