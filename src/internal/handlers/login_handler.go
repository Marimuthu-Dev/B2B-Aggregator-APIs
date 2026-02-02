package handlers

import (
	"net/http"

	"b2b-diagnostic-aggregator/apis/internal/dto"
	"b2b-diagnostic-aggregator/apis/internal/middleware"
	"b2b-diagnostic-aggregator/apis/internal/service"

	"github.com/gin-gonic/gin"
)

type LoginHandler struct {
	svc service.LoginService
}

func NewLoginHandler(svc service.LoginService) *LoginHandler {
	return &LoginHandler{svc: svc}
}

func (h *LoginHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if !middleware.BindJSON(c, &req) {
		return
	}

	resp, err := h.svc.Login(req.UserID, req.Password)
	if err != nil {
		respondError(c, err)
		return
	}

	respondData(c, http.StatusOK, resp, "Authenticated", nil)
}
