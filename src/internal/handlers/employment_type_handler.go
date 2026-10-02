package handlers

import (
	"net/http"

	"b2b-diagnostic-aggregator/apis/internal/service"

	"github.com/gin-gonic/gin"
)

type EmploymentTypeHandler struct {
	svc service.EmploymentTypeService
}

func NewEmploymentTypeHandler(svc service.EmploymentTypeService) *EmploymentTypeHandler {
	return &EmploymentTypeHandler{svc: svc}
}

func (h *EmploymentTypeHandler) GetActive(c *gin.Context) {
	data, err := h.svc.GetActiveEmploymentTypes()
	if err != nil {
		respondError(c, err)
		return
	}
	respondData(c, http.StatusOK, data, "Employment types fetched successfully", nil)
}
