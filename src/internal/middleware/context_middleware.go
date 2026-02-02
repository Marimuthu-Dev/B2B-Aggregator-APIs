package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	ContextIDKey    = "contextId"
	ContextIDHeader = "X-Context-Id"
)

func ContextMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		contextID := c.GetHeader(ContextIDHeader)
		if contextID == "" {
			contextID = generateContextID()
		}

		c.Set(ContextIDKey, contextID)
		c.Writer.Header().Set(ContextIDHeader, contextID)

		c.Next()
	}
}

func generateContextID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return time.Now().UTC().Format("20060102150405.000000000")
	}
	return hex.EncodeToString(buf)
}
