package app

import (
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
	packageClientMapRepo := repository.NewPackageClientMappingRepository(db)
	packageLabMapRepo := repository.NewPackageLabMappingRepository(db)
	loginRepo := repository.NewLoginRepository(db)
	forgotPasswordRepo := repository.NewForgotPasswordRepository(db)
	clientRepo := repository.NewClientRepository(db)
	clientLocationRepo := repository.NewClientLocationRepository(db)
	employeeRepo := repository.NewEmployeeRepository(db)
	labRepo := repository.NewLabRepository(db)
	leadRepo := repository.NewLeadRepository(db)
	leadUow := repository.NewLeadUnitOfWork(db)
	testRepo := repository.NewTestRepository(db)

	// Initialize Services
	packageSvc := service.NewPackageService(packageRepo, testRepo, packageClientMapRepo, packageLabMapRepo, clientRepo, labRepo)
	loginSvc := service.NewLoginService(loginRepo, forgotPasswordRepo, clientRepo, employeeRepo, labRepo, cfg.JWT)
	clientSvc := service.NewClientService(clientRepo)
	clientLocationSvc := service.NewClientLocationService(clientLocationRepo)
	employeeSvc := service.NewEmployeeService(employeeRepo)
	labSvc := service.NewLabService(labRepo)
	leadSvc := service.NewLeadService(leadRepo, leadUow, clientRepo, packageRepo)
	testSvc := service.NewTestService(testRepo)

	// Initialize Handlers
	packageHandler := handlers.NewPackageHandler(packageSvc)
	loginHandler := handlers.NewLoginHandler(loginSvc)
	clientHandler := handlers.NewClientHandler(clientSvc)
	clientLocationHandler := handlers.NewClientLocationHandler(clientLocationSvc)
	employeeHandler := handlers.NewEmployeeHandler(employeeSvc)
	labHandler := handlers.NewLabHandler(labSvc)
	leadHandler := handlers.NewLeadHandler(leadSvc)
	testHandler := handlers.NewTestHandler(testSvc)

	// Initialize Gin
	r := gin.Default()
	registerMiddleware(r, dbReady)

	registerRoutes(r, cfg.JWT.Secret, routeDeps{
		packageHandler: packageHandler,
		loginHandler:   loginHandler,
		clientHandler:         clientHandler,
		clientLocationHandler: clientLocationHandler,
		employeeHandler:       employeeHandler,
		labHandler:            labHandler,
		leadHandler:    leadHandler,
		testHandler:   testHandler,
	})

	// Azure App Service and cloud platforms set PORT env; default 8080
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server running on port %s", port)
	return r.Run(":" + port)
}
