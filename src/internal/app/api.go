package app

import (
	"fmt"
	"log"
	"os"

	"b2b-diagnostic-aggregator/apis/internal/config"
	"b2b-diagnostic-aggregator/apis/internal/handlers"
	"b2b-diagnostic-aggregator/apis/internal/logging"
	"b2b-diagnostic-aggregator/apis/internal/repository"
	"b2b-diagnostic-aggregator/apis/internal/service"

	"github.com/gin-gonic/gin"
)

func Run() error {
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
	dbReady := true
	if err != nil {
		dbReady = false
		log.Printf("Failed to connect to database: %v", err)
	}

	// Initialize Repositories
	packageRepo := repository.NewPackageRepository(db)
	loginRepo := repository.NewLoginRepository(db)
	forgotPasswordRepo := repository.NewForgotPasswordRepository(db)
	clientRepo := repository.NewClientRepository(db)
	labRepo := repository.NewLabRepository(db)
	leadRepo := repository.NewLeadRepository(db)
	leadUow := repository.NewLeadUnitOfWork(db)

	// Initialize Services
	packageSvc := service.NewPackageService(packageRepo)
	loginSvc := service.NewLoginService(loginRepo, forgotPasswordRepo, clientRepo, labRepo, cfg.JWT)
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
	registerMiddleware(r, dbReady)

	registerRoutes(r, cfg.JWT.Secret, routeDeps{
		packageHandler: packageHandler,
		loginHandler:   loginHandler,
		clientHandler:  clientHandler,
		labHandler:     labHandler,
		leadHandler:    leadHandler,
	})

	port := cfg.Port
	if port == 0 {
		port = 5000
	}

	log.Printf("🚀 Server is running on port %d", port)
	return r.Run(fmt.Sprintf(":%d", port))
}
