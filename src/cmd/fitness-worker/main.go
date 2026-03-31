package main

import (
	"context"
	"log"
	"log/slog"
	"os/signal"
	"syscall"

	"b2b-diagnostic-aggregator/apis/internal/config"
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

	db, err := config.ConnectDatabase(appCfg.DB)
	if err != nil {
		log.Fatalf("database: %v", err)
	}

	blobSvc, err := storage.NewAzureMoUBlobService(appCfg.AzureBlob, slog.Default())
	if err != nil {
		log.Fatalf("azure blob: %v", err)
	}

	deps := fitness.Deps{
		DB:         db,
		Blob:       blobSvc,
		LeadRepo:   repository.NewLeadRepository(db),
		ClientRepo: repository.NewClientRepository(db),
		Config:     wc,
		Log:        slog.Default(),
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
