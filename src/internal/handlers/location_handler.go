package handlers

import (
	"net/http"
	"strconv"

	"b2b-diagnostic-aggregator/apis/internal/dto"
	"b2b-diagnostic-aggregator/apis/internal/middleware"
	"b2b-diagnostic-aggregator/apis/internal/service"

	"github.com/gin-gonic/gin"
)

type LocationHandler struct {
	svc service.LocationService
}

func NewLocationHandler(svc service.LocationService) *LocationHandler {
	return &LocationHandler{svc: svc}
}

func (h *LocationHandler) GetStates(c *gin.Context) {
	data, err := h.svc.GetAllStates()
	if err != nil {
		respondError(c, err)
		return
	}
	respondData(c, http.StatusOK, data, "States fetched successfully", gin.H{"Count": len(data)})
}

func (h *LocationHandler) GetCities(c *gin.Context) {
	stateIDStr := c.Query("stateId")
	if stateIDStr == "" {
		stateIDStr = c.Query("state_id")
	}

	if stateIDStr != "" {
		stateID, err := strconv.ParseInt(stateIDStr, 10, 32)
		if err == nil && stateID > 0 {
			data, err := h.svc.GetCitiesByStateID(int32(stateID))
			if err != nil {
				respondError(c, err)
				return
			}
			respondData(c, http.StatusOK, data, "Cities fetched successfully", gin.H{"Count": len(data)})
			return
		}
	}

	data, err := h.svc.GetAllCities()
	if err != nil {
		respondError(c, err)
		return
	}
	respondData(c, http.StatusOK, data, "Cities fetched successfully", gin.H{"Count": len(data)})
}

func (h *LocationHandler) GetCitiesByStateID(c *gin.Context) {
	var params dto.StateIDPathParam
	if !middleware.BindUri(c, &params) {
		return
	}
	data, err := h.svc.GetCitiesByStateID(params.StateID)
	if err != nil {
		respondError(c, err)
		return
	}
	respondData(c, http.StatusOK, data, "Cities fetched successfully", gin.H{"Count": len(data)})
}
