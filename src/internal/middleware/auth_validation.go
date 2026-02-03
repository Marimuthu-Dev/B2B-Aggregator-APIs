package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// RequireDomain reads X-Domain header and sets it in context; aborts with 200 + success:false if missing (Node-compatible)
func RequireDomain() gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := c.GetHeader("X-Domain")
		if raw == "" {
			raw = c.GetHeader("x-domain")
		}
		if raw == "" {
			c.JSON(http.StatusOK, gin.H{
				"success":   false,
				"message":   "Domain header is required",
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			})
			c.Abort()
			return
		}
		domain := strings.TrimSpace(strings.ToLower(raw))
		c.Set("domain", domain)
		c.Next()
	}
}

// GetDomain returns the domain from context (must be used after RequireDomain)
func GetDomain(c *gin.Context) string {
	v, _ := c.Get("domain")
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
