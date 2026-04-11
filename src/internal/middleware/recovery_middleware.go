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
					"Success":   false,
					"Message":   "Internal server error",
					"Timestamp": time.Now().UTC().Format(time.RFC3339),
				})
			}
		}()

		c.Next()
	}
}
