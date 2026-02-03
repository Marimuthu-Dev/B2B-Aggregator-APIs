package handlers

import (
	"net/http"
	"strings"

	"b2b-diagnostic-aggregator/apis/internal/apperrors"
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

// Login accepts domain + mobileNumber + Password (X-Domain header) or legacy userId + Password
func (h *LoginHandler) Login(c *gin.Context) {
	domain := middleware.GetDomain(c)
	if domain == "" {
		raw := c.GetHeader("X-Domain")
		if raw == "" {
			raw = c.GetHeader("x-domain")
		}
		domain = strings.TrimSpace(strings.ToLower(raw))
	}
	if domain != "" {
		c.Set("domain", domain)
	}

	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err).SetType(gin.ErrorTypeBind)
		c.Abort()
		return
	}
	req.Domain = domain
	req.MobileNumber = strings.TrimSpace(req.MobileNumber)

	if req.Domain != "" && req.MobileNumber == "" {
		respondError(c, apperrors.NewBadRequest("mobileNumber is required when using X-Domain", nil))
		return
	}
	if req.Domain == "" && req.UserID == 0 {
		respondError(c, apperrors.NewBadRequest("either X-Domain + mobileNumber or userId is required", nil))
		return
	}

	resp, err := h.svc.Login(req)
	if err != nil {
		respondError(c, err)
		return
	}

	respondData(c, http.StatusOK, resp, "Authenticated", nil)
}

// CreateForgotPasswordKey creates a forgot-password key for the given mobile number (X-Domain required)
func (h *LoginHandler) CreateForgotPasswordKey(c *gin.Context) {
	domain := middleware.GetDomain(c)
	if domain == "" {
		raw := c.GetHeader("X-Domain")
		if raw == "" {
			raw = c.GetHeader("x-domain")
		}
		domain = strings.TrimSpace(strings.ToLower(raw))
	}
	if domain == "" {
		respondError(c, apperrors.NewBadRequest("Domain header is required", nil))
		return
	}

	var req dto.CreateForgotPasswordKeyRequest
	if !middleware.BindJSON(c, &req) {
		return
	}
	req.Domain = domain
	req.MobileNumber = strings.TrimSpace(req.MobileNumber)

	n, err := h.svc.CreateForgotPasswordRecord(req.Domain, req.MobileNumber)
	if err != nil {
		respondError(c, err)
		return
	}

	respondData(c, http.StatusOK, n, "Forgot password key generated successfully", nil)
}

// GetForgotPasswordKey returns the latest non-expired forgot-password key (query: mobileNumber)
func (h *LoginHandler) GetForgotPasswordKey(c *gin.Context) {
	domain := middleware.GetDomain(c)
	if domain == "" {
		raw := c.GetHeader("X-Domain")
		if raw == "" {
			raw = c.GetHeader("x-domain")
		}
		domain = strings.TrimSpace(strings.ToLower(raw))
	}
	if domain == "" {
		respondError(c, apperrors.NewBadRequest("Domain header is required", nil))
		return
	}

	var req dto.GetForgotPasswordKeyRequest
	if !middleware.BindQuery(c, &req) {
		return
	}
	req.Domain = domain
	req.MobileNumber = strings.TrimSpace(req.MobileNumber)

	result, err := h.svc.GetLatestForgotPasswordKey(req.Domain, req.MobileNumber)
	if err != nil {
		respondError(c, err)
		return
	}

	respondData(c, http.StatusOK, result, "Forgot password key fetched successfully", nil)
}

// ForgotPasswordReset resets password using forgetPasswordKey and new Password
func (h *LoginHandler) ForgotPasswordReset(c *gin.Context) {
	var req dto.ForgotPasswordResetRequest
	if !middleware.BindJSON(c, &req) {
		return
	}

	ok, err := h.svc.ForgotPasswordReset(req.ForgetPasswordKey, req.Password)
	if err != nil {
		respondError(c, err)
		return
	}
	if !ok {
		respondError(c, apperrors.NewBadRequest("Invalid or expired forgot password key", nil))
		return
	}

	respondData(c, http.StatusOK, 1, "Password updated successfully", nil)
}

// ChangePassword changes password using old password (X-Domain + mobileNumber + OldPassword + NewPassword)
func (h *LoginHandler) ChangePassword(c *gin.Context) {
	domain := middleware.GetDomain(c)
	if domain == "" {
		raw := c.GetHeader("X-Domain")
		if raw == "" {
			raw = c.GetHeader("x-domain")
		}
		domain = strings.TrimSpace(strings.ToLower(raw))
	}
	if domain == "" {
		respondError(c, apperrors.NewBadRequest("Domain header is required", nil))
		return
	}

	var req dto.ChangePasswordRequest
	if !middleware.BindJSON(c, &req) {
		return
	}
	req.Domain = domain
	req.MobileNumber = strings.TrimSpace(req.MobileNumber)

	ok, err := h.svc.ChangePassword(req.Domain, req.MobileNumber, req.OldPassword, req.NewPassword)
	if err != nil {
		respondError(c, err)
		return
	}
	if !ok {
		respondError(c, apperrors.NewBadRequest("Old password does not match", nil))
		return
	}

	respondData(c, http.StatusOK, 1, "Password updated successfully", nil)
}

// GetProfile returns user profile by userId or mobileNumber (query), X-Domain required
func (h *LoginHandler) GetProfile(c *gin.Context) {
	domain := middleware.GetDomain(c)
	if domain == "" {
		raw := c.GetHeader("X-Domain")
		if raw == "" {
			raw = c.GetHeader("x-domain")
		}
		domain = strings.TrimSpace(strings.ToLower(raw))
	}
	if domain == "" {
		respondError(c, apperrors.NewBadRequest("Domain header is required", nil))
		return
	}

	var req dto.GetProfileRequest
	if !middleware.BindQuery(c, &req) {
		return
	}
	req.Domain = domain
	req.UserID = strings.TrimSpace(req.UserID)
	req.MobileNumber = strings.TrimSpace(req.MobileNumber)

	if req.UserID == "" && req.MobileNumber == "" {
		respondError(c, apperrors.NewBadRequest("Either userId or mobileNumber is required", nil))
		return
	}

	var userID, mobileNumber *string
	if req.UserID != "" {
		userID = &req.UserID
	}
	if req.MobileNumber != "" {
		mobileNumber = &req.MobileNumber
	}

	result, err := h.svc.GetProfile(req.Domain, userID, mobileNumber)
	if err != nil {
		respondError(c, err)
		return
	}

	respondData(c, http.StatusOK, result, "Profile fetched successfully", nil)
}
