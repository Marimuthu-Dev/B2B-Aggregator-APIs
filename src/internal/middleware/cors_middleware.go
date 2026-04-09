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
		AllowOrigins: nil, // allow all (with credentials, request Origin is reflected)
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		// Include common casing variants; preflight also merges Access-Control-Request-Headers.
		AllowHeaders: []string{
			"Content-Type", "Authorization",
			"x-domain", "X-Domain",
			"Accept", "Accept-Language", "X-Requested-With",
		},
		AllowCredentials: true,
	}
}

func mergeAllowHeaders(defaults []string, requestHeaders string) string {
	if strings.TrimSpace(requestHeaders) == "" {
		return strings.Join(defaults, ", ")
	}
	seen := make(map[string]struct{})
	var out []string
	for _, d := range defaults {
		d = strings.TrimSpace(d)
		if d == "" {
			continue
		}
		seen[strings.ToLower(d)] = struct{}{}
		out = append(out, d)
	}
	for _, part := range strings.Split(requestHeaders, ",") {
		p := strings.TrimSpace(part)
		if p == "" {
			continue
		}
		key := strings.ToLower(p)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, p)
	}
	return strings.Join(out, ", ")
}

// CORSMiddleware returns a Gin handler that sets CORS headers and handles OPTIONS preflight.
// Register it first so OPTIONS requests get a 204 and never hit route-not-found (404).
func CORSMiddleware(cfg CORSConfig) gin.HandlerFunc {
	methods := strings.Join(cfg.AllowMethods, ", ")
	defaultHeaders := cfg.AllowHeaders
	allowCreds := "true"

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		// With credentials: true, browser does not accept "*"; reflect request origin.
		if cfg.AllowCredentials && origin != "" {
			c.Header(corsAllowOrigin, origin)
			c.Header("Vary", "Origin")
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
		reqHdrs := c.GetHeader(corsRequestHeaders)
		c.Header(corsAllowHeaders, mergeAllowHeaders(defaultHeaders, reqHdrs))
		c.Header(corsAllowCredentials, allowCreds)
		// Cache preflight in the browser (reduces OPTIONS chatter during local dev).
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
