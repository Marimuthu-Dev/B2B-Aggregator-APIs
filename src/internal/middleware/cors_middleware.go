package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	corsAllowOrigin      = "Access-Control-Allow-Origin"
	corsAllowMethods     = "Access-Control-Allow-Methods"
	corsAllowHeaders     = "Access-Control-Allow-Headers"
	corsAllowCredentials = "Access-Control-Allow-Credentials"
	corsRequestMethod    = "Access-Control-Request-Method"
	corsRequestHeaders   = "Access-Control-Request-Headers"
)

// CORSConfig matches the Node.js cors() options used in the frontend.
// - origin: "*" (when credentials not used; with credentials we reflect request origin)
// - methods: GET, POST, PUT, DELETE, PATCH, OPTIONS
// - allowedHeaders: Content-Type, Authorization, x-domain
// - credentials: true
type CORSConfig struct {
	AllowOrigins     []string // empty means allow all
	AllowMethods     []string
	AllowHeaders     []string
	AllowCredentials bool
}

// DefaultCORSConfig returns config equivalent to the Node.js snippet.
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowOrigins:     nil, // allow all
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type", "Authorization", "x-domain"},
		AllowCredentials: true,
	}
}

// CORSMiddleware returns a Gin handler that sets CORS headers and handles OPTIONS preflight.
// Register it first so OPTIONS requests get a 204 and never hit route-not-found (404).
func CORSMiddleware(cfg CORSConfig) gin.HandlerFunc {
	methods := strings.Join(cfg.AllowMethods, ", ")
	headers := strings.Join(cfg.AllowHeaders, ", ")
	allowCreds := "true"

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		// With credentials: true, browser does not accept "*"; reflect request origin.
		if cfg.AllowCredentials && origin != "" {
			c.Header(corsAllowOrigin, origin)
		} else if len(cfg.AllowOrigins) == 0 {
			c.Header(corsAllowOrigin, "*")
		} else {
			for _, o := range cfg.AllowOrigins {
				if o == origin || o == "*" {
					if o == "*" {
						c.Header(corsAllowOrigin, "*")
					} else {
						c.Header(corsAllowOrigin, origin)
					}
					break
				}
			}
		}

		c.Header(corsAllowMethods, methods)
		c.Header(corsAllowHeaders, headers)
		c.Header(corsAllowCredentials, allowCreds)

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
