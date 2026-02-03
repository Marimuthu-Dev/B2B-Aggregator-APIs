package dto

// LoginRequest supports domain + mobileNumber + password (Node-style) or legacy userId + password
type LoginRequest struct {
	Domain       string `json:"-"`            // from X-Domain header
	MobileNumber string `json:"mobileNumber"` // required when using domain
	Password     string `json:"Password" binding:"required"`
	UserID       int64  `json:"userId"` // legacy: optional when domain+mobileNumber provided
}

// LoginResponse returns user data and tokens
type LoginResponse struct {
	User         interface{} `json:"user"`
	Token        string      `json:"token"`
	RefreshToken string      `json:"refreshToken"`
}

// CreateForgotPasswordKeyRequest creates a forgot-password key for a mobile number
type CreateForgotPasswordKeyRequest struct {
	Domain       string `json:"-"`
	MobileNumber string `json:"mobileNumber" binding:"required"`
}

// GetForgotPasswordKeyRequest gets the latest forgot-password key (query params)
type GetForgotPasswordKeyRequest struct {
	Domain       string `form:"-"`
	MobileNumber string `form:"mobileNumber" binding:"required"`
}

// ForgotPasswordResetRequest resets password using forgot-password key
type ForgotPasswordResetRequest struct {
	ForgetPasswordKey string `json:"forgetPasswordKey" binding:"required"`
	Password          string `json:"Password" binding:"required"`
}

// ChangePasswordRequest changes password using old password
type ChangePasswordRequest struct {
	Domain       string `json:"-"`
	MobileNumber string `json:"mobileNumber" binding:"required"`
	OldPassword  string `json:"OldPassword" binding:"required"`
	NewPassword  string `json:"NewPassword" binding:"required"`
}

// GetProfileRequest gets profile by userId or mobileNumber (query params)
type GetProfileRequest struct {
	Domain       string `form:"-"`
	UserID       string `form:"userId"`
	MobileNumber string `form:"mobileNumber"`
}

// ForgotPasswordKeyResponse for get-forgot-password-key response
type ForgotPasswordKeyResponse struct {
	ForgetPasswordKey string `json:"forgetPasswordKey"`
	Expiry            string `json:"expiry"` // RFC3339
}
