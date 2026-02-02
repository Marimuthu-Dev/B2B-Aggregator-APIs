package dto

type LoginRequest struct {
	UserID   int64  `json:"userId" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	User         interface{} `json:"user"`
	Token        string      `json:"token"`
	RefreshToken string      `json:"refreshToken"`
}
