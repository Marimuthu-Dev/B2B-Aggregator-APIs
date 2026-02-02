package main

import (
	"fmt"
	"log"
	"os"

	"b2b-diagnostic-aggregator/apis/internal/config"
	"b2b-diagnostic-aggregator/apis/internal/handlers"
	"b2b-diagnostic-aggregator/apis/internal/logging"
	"b2b-diagnostic-aggregator/apis/internal/middleware"
	"b2b-diagnostic-aggregator/apis/internal/repository"
	"b2b-diagnostic-aggregator/apis/internal/service"

	"github.com/gin-gonic/gin"
)

func main() {
	// Load configuration
	cfg := config.LoadConfig()

	logWriter, err := logging.NewHourlyFileWriter(logging.Config{
		Dir:            cfg.Log.Dir,
		RetentionHours: cfg.Log.RetentionHours,
		Prefix:         "api",
	})
	if err != nil {
		log.Printf("Failed to initialize file logger: %v", err)
		log.SetOutput(os.Stdout)
	} else {
		log.SetOutput(logWriter)
	}
	log.SetFlags(0)

	// Connect to database
	db, err := config.ConnectDatabase(cfg.DB)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Initialize Repositories
	packageRepo := repository.NewPackageRepository(db)
	loginRepo := repository.NewLoginRepository(db)
	clientRepo := repository.NewClientRepository(db)
	labRepo := repository.NewLabRepository(db)
	leadRepo := repository.NewLeadRepository(db)
	leadUow := repository.NewLeadUnitOfWork(db)

	// Initialize Services
	packageSvc := service.NewPackageService(packageRepo)
	loginSvc := service.NewLoginService(loginRepo, cfg.JWT)
	clientSvc := service.NewClientService(clientRepo)
	labSvc := service.NewLabService(labRepo)
	leadSvc := service.NewLeadService(leadRepo, leadUow)

	// Initialize Handlers
	packageHandler := handlers.NewPackageHandler(packageSvc)
	loginHandler := handlers.NewLoginHandler(loginSvc)
	clientHandler := handlers.NewClientHandler(clientSvc)
	labHandler := handlers.NewLabHandler(labSvc)
	leadHandler := handlers.NewLeadHandler(leadSvc)

	// Initialize Gin
	r := gin.Default()
	r.Use(middleware.RecoveryMiddleware())
	r.Use(middleware.ContextMiddleware())
	r.Use(middleware.LoggingMiddleware())
	r.Use(middleware.ValidationErrorMiddleware())

	// Public Routes
	v1 := r.Group("/api/v1")
	v1.POST("/login", loginHandler.Login)
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "pong"})
	})

	// Protected Routes
	api := v1.Group("")
	api.Use(middleware.AuthMiddleware(cfg.JWT.Secret))
	{
		// Packages
		packages := api.Group("/packages")
		{
			packages.GET("/", packageHandler.GetAll)
			packages.GET("/:id", packageHandler.GetByID)
			packages.POST("/", packageHandler.Create)
			packages.POST("/with-tests", packageHandler.CreateWithTests)
			packages.DELETE("/:id", packageHandler.Delete)
		}

		// Clients
		clients := api.Group("/clients")
		{
			clients.GET("/", clientHandler.GetAll)
			clients.GET("/:id", clientHandler.GetByID)
			clients.GET("/contact", clientHandler.GetByContactNumber)
			clients.POST("/", clientHandler.Create)
			clients.PUT("/:id", clientHandler.Update)
			clients.DELETE("/:id", clientHandler.Delete)
		}

		// Labs
		labs := api.Group("/labs")
		{
			labs.GET("/", labHandler.GetAll)
			labs.GET("/:id", labHandler.GetByID)
			labs.GET("/contact", labHandler.GetByContactNumber)
			labs.POST("/", labHandler.Create)
			labs.PUT("/:id", labHandler.Update)
			labs.DELETE("/:id", labHandler.Delete)
		}

		// Leads
		leads := api.Group("/leads")
		{
			leads.GET("/", leadHandler.GetAll)
			leads.GET("/:id", leadHandler.GetByID)
			leads.POST("/", leadHandler.Create)
			leads.PUT("/:id", leadHandler.Update)
			leads.DELETE("/:id", leadHandler.Delete)
			leads.POST("/bulk-status", leadHandler.BulkUpdateStatus)
		}
	}

	port := cfg.Port
	if port == 0 {
		port = 5000
	}

	log.Printf("🚀 Server is running on port %d", port)
	r.Run(fmt.Sprintf(":%d", port))
}
