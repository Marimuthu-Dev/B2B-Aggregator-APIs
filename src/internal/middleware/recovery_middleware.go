package middleware

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func RecoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"success":   false,
					"message":   "Internal server error",
					"timestamp": time.Now().UTC().Format(time.RFC3339),
				})
			}
		}()

		c.Next()
	}
}
