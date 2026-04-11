package middleware

import (
	"net/http"
	"strings"
	"time"

	"b2b-diagnostic-aggregator/apis/pkg/utils"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"Success":   false,
				"Message":   "Authorization header is required",
				"Timestamp": time.Now().UTC().Format(time.RFC3339),
			})
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			c.JSON(http.StatusUnauthorized, gin.H{
				"Success":   false,
				"Message":   "Authorization header format must be Bearer {token}",
				"Timestamp": time.Now().UTC().Format(time.RFC3339),
			})
			c.Abort()
			return
		}

		claims, err := utils.ValidateToken(parts[1], secret)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"Success":   false,
				"Message":   "Invalid or expired token",
				"Timestamp": time.Now().UTC().Format(time.RFC3339),
			})
			c.Abort()
			return
		}

		// Set user information in context (Node-compatible: userId, userType)
		c.Set("userId", claims.UserID)
		c.Set("userType", claims.UserType)
		c.Set("role", claims.Role())

		c.Next()
	}
}
