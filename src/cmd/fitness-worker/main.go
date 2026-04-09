package main

import (
	"context"
	"io"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"b2b-diagnostic-aggregator/apis/internal/config"
	"b2b-diagnostic-aggregator/apis/internal/logging"
	"b2b-diagnostic-aggregator/apis/internal/repository"
	"b2b-diagnostic-aggregator/apis/internal/storage"
	"b2b-diagnostic-aggregator/apis/internal/worker/fitness"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	_ = godotenv.Load(".env")
	_ = godotenv.Load("src/.env")
	_ = godotenv.Load("../.env")

	appCfg := config.LoadConfig()
	wc, err := config.LoadFitnessCertWorkerConfig()
	if err != nil {
		log.Fatalf("fitness worker config: %v", err)
	}

	var logOut io.Writer = os.Stderr
	if logWriter, lwErr := logging.NewHourlyFileWriter(logging.Config{
		Dir:            appCfg.Log.Dir,
		RetentionHours: appCfg.Log.RetentionHours,
		Prefix:         "fitness-worker",
	}); lwErr != nil {
		log.Printf("fitness worker: file logging disabled (%v); using stderr only", lwErr)
		log.SetOutput(os.Stderr)
	} else {
		logOut = io.MultiWriter(os.Stderr, logWriter)
		log.SetOutput(logOut)
	}
	log.SetFlags(log.LstdFlags)

	logger := slog.New(slog.NewTextHandler(logOut, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)
	logger.Info("fitness worker starting",
		slog.String("logDir", appCfg.Log.Dir),
		slog.Int("logRetentionHours", appCfg.Log.RetentionHours),
		slog.String("templateDir", wc.TemplateDir),
		slog.Bool("runOnce", wc.RunOnce),
		slog.String("pollInterval", wc.PollInterval.String()),
	)

	db, err := config.ConnectDatabase(appCfg.DB)
	if err != nil {
		log.Fatalf("database: %v", err)
	}

	blobSvc, err := storage.NewAzureMoUBlobService(appCfg.AzureBlob, logger)
	if err != nil {
		log.Fatalf("azure blob: %v", err)
	}

	deps := fitness.Deps{
		DB:         db,
		Blob:       blobSvc,
		LeadRepo:   repository.NewLeadRepository(db),
		ClientRepo: repository.NewClientRepository(db),
		Config:     wc,
		Log:        logger,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if wc.RunOnce {
		if err := fitness.RunOnce(ctx, deps); err != nil {
			log.Fatalf("fitness worker: %v", err)
		}
		return
	}

	if err := fitness.RunLoop(ctx, deps); err != nil && err != context.Canceled {
		log.Fatalf("fitness worker: %v", err)
	}
	log.Println("fitness worker stopped")
}
