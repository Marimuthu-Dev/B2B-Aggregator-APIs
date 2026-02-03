package utils

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTClaims matches Node.js payload: userId and userType (1=employee, 2=client, 3=lab)
type JWTClaims struct {
	UserID   int64 `json:"userId"`
	UserType int   `json:"userType"`
	jwt.RegisteredClaims
}

// Legacy: Role kept for backward compatibility when reading from token
func (c *JWTClaims) Role() string {
	switch c.UserType {
	case 1:
		return "employee"
	case 2:
		return "client"
	case 3:
		return "lab"
	default:
		return ""
	}
}

// GenerateToken creates access and refresh tokens with userId and userType (Node-compatible)
func GenerateToken(userID int64, userType int, secret string, accessTTL, refreshTTL time.Duration) (string, string, error) {
	accessClaims := &JWTClaims{
		UserID:   userID,
		UserType: userType,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(accessTTL)),
		},
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessString, err := accessToken.SignedString([]byte(secret))
	if err != nil {
		return "", "", err
	}

	refreshClaims := &JWTClaims{
		UserID:   userID,
		UserType: userType,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(refreshTTL)),
		},
	}
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshString, err := refreshToken.SignedString([]byte(secret))
	if err != nil {
		return "", "", err
	}

	return accessString, refreshString, nil
}

// Legacy: GenerateTokenWithRole for backward compatibility (maps role to userType)
func GenerateTokenWithRole(userID int64, role string, secret string, accessTTL, refreshTTL time.Duration) (string, string, error) {
	userType := 0
	switch role {
	case "employee":
		userType = 1
	case "client":
		userType = 2
	case "lab":
		userType = 3
	}
	return GenerateToken(userID, userType, secret, accessTTL, refreshTTL)
}

func ValidateToken(tokenString string, secret string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, jwt.ErrSignatureInvalid
}
