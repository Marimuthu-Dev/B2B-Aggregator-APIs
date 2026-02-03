package app

import (
	"time"

	"b2b-diagnostic-aggregator/apis/internal/handlers"
	"b2b-diagnostic-aggregator/apis/internal/middleware"

	"github.com/gin-gonic/gin"
)

func registerMiddleware(r *gin.Engine, dbReady bool) {
	r.Use(middleware.RecoveryMiddleware())
	r.Use(middleware.ContextMiddleware())
	r.Use(middleware.LoggingMiddleware())
	r.Use(middleware.ValidationErrorMiddleware())
	if !dbReady {
		r.Use(func(c *gin.Context) {
			if c.Request.URL.Path == "/ping" {
				c.Next()
				return
			}
			c.JSON(503, gin.H{
				"success":   false,
				"message":   "Database unavailable",
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			})
			c.Abort()
		})
	}
}

func registerRoutes(r *gin.Engine, jwtSecret string, deps routeDeps) {
	registerPublicRoutes(r, deps)
	registerProtectedRoutes(r, jwtSecret, deps)
}

type routeDeps struct {
	packageHandler *handlers.PackageHandler
	loginHandler   *handlers.LoginHandler
	clientHandler  *handlers.ClientHandler
	labHandler     *handlers.LabHandler
	leadHandler    *handlers.LeadHandler
}

func registerPublicRoutes(r *gin.Engine, deps routeDeps) {
	v1 := r.Group("/api/v1")
	login := v1.Group("/login")
	{
		login.POST("", deps.loginHandler.Login)
		login.POST("/forgot-password", deps.loginHandler.ForgotPasswordReset)
		login.POST("/forgot-password-key", deps.loginHandler.CreateForgotPasswordKey)
		login.GET("/forgot-password-key", deps.loginHandler.GetForgotPasswordKey)
		login.POST("/change-password", deps.loginHandler.ChangePassword)
		login.GET("/profile", deps.loginHandler.GetProfile) // public with X-Domain + userId or mobileNumber
	}
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "pong"})
	})
}

func registerProtectedRoutes(r *gin.Engine, jwtSecret string, deps routeDeps) {
	v1 := r.Group("/api/v1")
	api := v1.Group("")
	api.Use(middleware.AuthMiddleware(jwtSecret))
	{
		registerPackageRoutes(api, deps.packageHandler)
		registerClientRoutes(api, deps.clientHandler)
		registerLabRoutes(api, deps.labHandler)
		registerLeadRoutes(api, deps.leadHandler)
	}
}

func registerPackageRoutes(api *gin.RouterGroup, handler *handlers.PackageHandler) {
	packages := api.Group("/packages")
	{
		packages.GET("/", handler.GetAll)
		packages.GET("/:id", handler.GetByID)
		packages.POST("/", handler.Create)
		packages.POST("/with-tests", handler.CreateWithTests)
		packages.DELETE("/:id", handler.Delete)
	}
}

func registerClientRoutes(api *gin.RouterGroup, handler *handlers.ClientHandler) {
	clients := api.Group("/clients")
	{
		clients.GET("/", handler.GetAll)
		clients.GET("/:id", handler.GetByID)
		clients.GET("/contact", handler.GetByContactNumber)
		clients.POST("/", handler.Create)
		clients.PUT("/:id", handler.Update)
		clients.DELETE("/:id", handler.Delete)
	}
}

func registerLabRoutes(api *gin.RouterGroup, handler *handlers.LabHandler) {
	labs := api.Group("/labs")
	{
		labs.GET("/", handler.GetAll)
		labs.GET("/:id", handler.GetByID)
		labs.GET("/contact", handler.GetByContactNumber)
		labs.POST("/", handler.Create)
		labs.PUT("/:id", handler.Update)
		labs.DELETE("/:id", handler.Delete)
	}
}

func registerLeadRoutes(api *gin.RouterGroup, handler *handlers.LeadHandler) {
	leads := api.Group("/leads")
	{
		leads.GET("/", handler.GetAll)
		leads.GET("/:id", handler.GetByID)
		leads.POST("/", handler.Create)
		leads.PUT("/:id", handler.Update)
		leads.DELETE("/:id", handler.Delete)
		leads.POST("/bulk-status", handler.BulkUpdateStatus)
	}
}
