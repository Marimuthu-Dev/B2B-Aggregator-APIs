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
	packageHandler        *handlers.PackageHandler
	loginHandler          *handlers.LoginHandler
	clientHandler         *handlers.ClientHandler
	clientLocationHandler *handlers.ClientLocationHandler
	employeeHandler       *handlers.EmployeeHandler
	labHandler            *handlers.LabHandler
	leadHandler           *handlers.LeadHandler
	testHandler           *handlers.TestHandler
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
		registerClientLocationRoutes(api, deps.clientLocationHandler)
		registerEmployeeRoutes(api, deps.employeeHandler)
		registerLabRoutes(api, deps.labHandler)
		registerLeadRoutes(api, deps.leadHandler)
		registerTestRoutes(api, deps.testHandler)
	}
}

func registerPackageRoutes(api *gin.RouterGroup, handler *handlers.PackageHandler) {
	packages := api.Group("/packages")
	{
		packages.GET("/", handler.GetAll)
		packages.GET("/with-tests-details", handler.GetAllWithTestsDetails)
		packages.GET("/:id", handler.GetByID)
		packages.POST("/", handler.Create)
		packages.POST("/with-tests", handler.CreateWithTests)
		packages.PUT("/:id", handler.UpdatePackageStatus)
		packages.DELETE("/:id", handler.Delete)
		packages.POST("/client-mapping", handler.CreatePackageClientMapping)
		packages.GET("/client-mapping", handler.GetAllPackageClientMappings)
		packages.PUT("/client-mapping/:id", handler.UpdatePackageClientMappingStatus)
		packages.POST("/lab-mapping", handler.CreatePackageLabMapping)
		packages.GET("/lab-mapping", handler.GetAllPackageLabMappings)
		packages.PUT("/lab-mapping/:id", handler.UpdatePackageLabMappingStatus)
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

func registerClientLocationRoutes(api *gin.RouterGroup, handler *handlers.ClientLocationHandler) {
	loc := api.Group("/client/:client_id/locations")
	{
		loc.GET("/", handler.GetAllByClientID)
		loc.GET("/:id", handler.GetByID)
		loc.POST("/", handler.Create)
		loc.PUT("/:id", handler.Update)
		loc.DELETE("/:id", handler.Delete)
	}
}

func registerEmployeeRoutes(api *gin.RouterGroup, handler *handlers.EmployeeHandler) {
	employees := api.Group("/employees")
	{
		employees.GET("/", handler.GetAll)
		employees.GET("/search", handler.GetByContactNumber)
		employees.GET("/:id", handler.GetByID)
		employees.POST("/", handler.Create)
		employees.PUT("/:id", handler.Update)
		employees.DELETE("/:id", handler.Delete)
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
		leads.POST("/bulk-csv", handler.BulkImportCsv)
	}
}

func registerTestRoutes(api *gin.RouterGroup, handler *handlers.TestHandler) {
	tests := api.Group("/tests")
	{
		tests.GET("/", handler.GetAll)
		tests.GET("/active", handler.GetActive)
		tests.GET("/:id", handler.GetByID)
	}
}
