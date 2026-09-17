package middleware

import (
	"net/http"
	"strings"
	"time"

	"b2b-diagnostic-aggregator/apis/pkg/utils"

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
				"Success":   false,
				"Message":   "Domain header is required",
				"Timestamp": time.Now().UTC().Format(time.RFC3339),
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

// GetUserID returns the authenticated user ID from context (set by AuthMiddleware from JWT).
// Use for setting CreatedBy / LastUpdatedBy in POST/PUT; returns (0, false) if not found or invalid.
func GetUserID(c *gin.Context) (int64, bool) {
	v, ok := c.Get("userId")
	if !ok {
		return 0, false
	}
	switch id := v.(type) {
	case int64:
		return id, true
	case int:
		return int64(id), true
	default:
		return 0, false
	}
}

// GetUserType returns the authenticated user type from context (set by AuthMiddleware from JWT).
// Values match login: 1=employee, 2=client, 3=lab. Returns (0, false) if not found or invalid.
func GetUserType(c *gin.Context) (int, bool) {
	v, ok := c.Get("userType")
	if !ok {
		return 0, false
	}
	switch t := v.(type) {
	case int:
		return t, true
	case int64:
		return int(t), true
	case float64:
		return int(t), true
	default:
		return 0, false
	}
}

// GetStoreID returns jwt.userId when userType is store (4). Query/body storeId must not override this.
func GetStoreID(c *gin.Context) (int64, bool) {
	userType, ok := GetUserType(c)
	if !ok || userType != utils.UserTypeStore {
		return 0, false
	}
	return GetUserID(c)
}
