package app

import (
	"log"
	"log/slog"
	"os"
	"strings"

	"b2b-diagnostic-aggregator/apis/internal/config"
	"b2b-diagnostic-aggregator/apis/internal/dto"
	"b2b-diagnostic-aggregator/apis/internal/handlers"
	"b2b-diagnostic-aggregator/apis/internal/logging"
	"b2b-diagnostic-aggregator/apis/internal/repository"
	"b2b-diagnostic-aggregator/apis/internal/service"
	"b2b-diagnostic-aggregator/apis/internal/storage"

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
	storeRepo := repository.NewStoreRepository(db)
	var blobSvc service.BlobService
	ab := cfg.AzureBlob
	blobConfigured := strings.TrimSpace(ab.ConnectionString) != "" ||
		(strings.TrimSpace(ab.StorageAccountName) != "" && strings.TrimSpace(ab.StorageAccountKey) != "")
	if blobConfigured {
		bs, err := storage.NewAzureMoUBlobService(ab, slog.Default())
		if err != nil {
			log.Printf("MoU Azure Blob disabled (invalid config): %v", err)
		} else {
			blobSvc = bs
		}
	}
	var emailOutbox *repository.EmailOutboxRepository
	if db != nil {
		if sqlDB, sqlErr := db.DB(); sqlErr != nil {
			log.Printf("Email outbox disabled (sql.DB): %v", sqlErr)
		} else {
			emailOutbox = repository.NewEmailOutboxRepositoryFromSQL(sqlDB)
		}
	}
	loginSvc := service.NewLoginService(loginRepo, forgotPasswordRepo, clientRepo, employeeRepo, labRepo, storeRepo, cfg.JWT, emailOutbox, cfg.Email, cfg.Domains)
	clientSvc := service.NewClientService(clientRepo, blobSvc, storeRepo, emailOutbox, forgotPasswordRepo, cfg.Email, cfg.Domains.Client)
	clientLocationSvc := service.NewClientLocationService(clientLocationRepo)
	employeeSvc := service.NewEmployeeService(employeeRepo, emailOutbox, forgotPasswordRepo, cfg.Email, cfg.Domains.Employee)
	labSvc := service.NewLabService(labRepo, blobSvc, emailOutbox, forgotPasswordRepo, cfg.Email, cfg.Domains.Lab)
	storeSvc := service.NewStoreService(storeRepo, clientRepo, emailOutbox, forgotPasswordRepo, cfg.Email, cfg.Domains.Client)
	leadSvc := service.NewLeadService(leadRepo, leadUow, clientRepo, packageRepo, labRepo, storeRepo, blobSvc)
	testSvc := service.NewTestService(testRepo)

	// Initialize Handlers
	packageHandler := handlers.NewPackageHandler(packageSvc, storeSvc)
	loginHandler := handlers.NewLoginHandler(loginSvc)
	clientHandler := handlers.NewClientHandler(clientSvc)
	clientLocationHandler := handlers.NewClientLocationHandler(clientLocationSvc)
	employeeHandler := handlers.NewEmployeeHandler(employeeSvc)
	labHandler := handlers.NewLabHandler(labSvc)
	storeHandler := handlers.NewStoreHandler(storeSvc)
	leadHandler := handlers.NewLeadHandler(leadSvc, storeSvc)
	testHandler := handlers.NewTestHandler(testSvc)

	// Initialize Gin
	r := gin.Default()
	r.MaxMultipartMemory = dto.MultipartFormMaxMemory
	r.RedirectTrailingSlash = false // allow both /path and /path/ without 301 redirect (e.g. for FE clients that use trailing slash)
	registerMiddleware(r, dbReady)

	registerRoutes(r, cfg.JWT.Secret, routeDeps{
		packageHandler:        packageHandler,
		loginHandler:          loginHandler,
		clientHandler:         clientHandler,
		clientLocationHandler: clientLocationHandler,
		employeeHandler:       employeeHandler,
		labHandler:            labHandler,
		storeHandler:          storeHandler,
		leadHandler:           leadHandler,
		testHandler:           testHandler,
	})

	// Azure App Service and cloud platforms set PORT env; default 8080
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server running on port %s", port)
	return r.Run(":" + port)
}
