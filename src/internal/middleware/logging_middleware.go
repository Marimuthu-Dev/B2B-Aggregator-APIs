package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	requestIDKey = "requestId"
	traceIDKey   = "traceId"
)

func LoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		requestID := resolveRequestID(c)
		traceID := resolveTraceID(c, requestID)
		contextID := resolveContextID(c)

		c.Set(requestIDKey, requestID)
		c.Set(traceIDKey, traceID)
		c.Writer.Header().Set("X-Request-Id", requestID)
		c.Writer.Header().Set("X-Trace-Id", traceID)

		c.Next()

		duration := time.Since(start)
		status := c.Writer.Status()

		timestamp := time.Now().UTC().Format(time.RFC3339)
		log.Printf(`{"timestamp":"%s","context_id":"%s","level":"info","event":"http_request","method":"%s","path":"%s","status":%d,"duration_ms":%d,"request_id":"%s","trace_id":"%s"}`,
			timestamp, contextID, method, path, status, duration.Milliseconds(), requestID, traceID)
	}
}

func resolveRequestID(c *gin.Context) string {
	if value := c.GetHeader("X-Request-Id"); value != "" {
		return value
	}
	return generateID()
}

func resolveTraceID(c *gin.Context, requestID string) string {
	if value := c.GetHeader("X-Trace-Id"); value != "" {
		return value
	}
	return requestID
}

func resolveContextID(c *gin.Context) string {
	if value := c.GetHeader(ContextIDHeader); value != "" {
		return value
	}
	if value, ok := c.Get(ContextIDKey); ok {
		if contextID, ok := value.(string); ok && contextID != "" {
			return contextID
		}
	}
	return generateID()
}

func generateID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return time.Now().UTC().Format("20060102150405.000000000")
	}
	return hex.EncodeToString(buf)
}
