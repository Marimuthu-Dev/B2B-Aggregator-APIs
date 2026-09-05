package main

import (
	"context"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"b2b-diagnostic-aggregator/apis/internal/config"
	"b2b-diagnostic-aggregator/apis/internal/logging"
	"b2b-diagnostic-aggregator/apis/internal/repository"
	"b2b-diagnostic-aggregator/apis/internal/whatsapp"
	whatsappworker "b2b-diagnostic-aggregator/apis/internal/worker/whatsapp"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	_ = godotenv.Load(".env")
	_ = godotenv.Load("src/.env")
	_ = godotenv.Load("../.env")

	appCfg := config.LoadConfig()
	wc, err := config.LoadWhatsAppWorkerConfig()
	if err != nil {
		log.Fatalf("whatsapp worker config: %v", err)
	}

	var logOut io.Writer = os.Stderr
	if logWriter, lwErr := logging.NewHourlyFileWriter(logging.Config{
		Dir:            appCfg.Log.Dir,
		RetentionHours: appCfg.Log.RetentionHours,
		Prefix:         "whatsapp-worker",
	}); lwErr != nil {
		log.Printf("whatsapp worker: file logging disabled (%v); using stderr only", lwErr)
		log.SetOutput(os.Stderr)
	} else {
		logOut = io.MultiWriter(os.Stderr, logWriter)
		log.SetOutput(logOut)
	}
	log.SetFlags(log.LstdFlags)

	logger := slog.New(slog.NewTextHandler(logOut, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)
	logger.Info("whatsapp worker starting",
		slog.String("logDir", appCfg.Log.Dir),
		slog.Int("logRetentionHours", appCfg.Log.RetentionHours),
		slog.Bool("singleBatch", wc.SingleBatch),
		slog.String("apiEndpoint", wc.APIEndpoint),
		slog.String("templateName", wc.TemplateName),
		slog.String("campaignName", wc.CampaignName),
		slog.Int("batchSize", wc.BatchSize),
		slog.Duration("pollIntervalAfterWork", wc.PollInterval),
		slog.Duration("idleWaitWhenEmpty", wc.IdleWait),
		slog.Duration("sendTimeout", wc.SendTimeout),
	)

	dbGorm, err := config.ConnectDatabase(appCfg.DB)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	sqlDB, err := dbGorm.DB()
	if err != nil {
		log.Fatalf("database sql: %v", err)
	}
	defer func() {
		if cerr := sqlDB.Close(); cerr != nil {
			logger.Warn("db close", slog.String("error", cerr.Error()))
		}
	}()

	repo, err := repository.NewWhatsAppRepository(context.Background(), sqlDB)
	if err != nil {
		log.Fatalf("whatsapp repository: %v", err)
	}

	httpClient := &http.Client{Timeout: wc.SendTimeout + 5*time.Second}
	sender, err := whatsapp.NewService(whatsapp.Config{
		APIKey:       wc.APIKey,
		APIEndpoint:  wc.APIEndpoint,
		TemplateName: wc.TemplateName,
		CampaignName: wc.CampaignName,
		HTTPClient:   httpClient,
		SendTimeout:  wc.SendTimeout,
	})
	if err != nil {
		log.Fatalf("whatsapp service: %v", err)
	}

	deps := whatsappworker.Deps{
		Repo:   repo,
		Sender: sender,
		Config: wc,
		Log:    logger,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if wc.SingleBatch {
		if _, err := whatsappworker.RunOnce(ctx, deps); err != nil {
			log.Fatalf("whatsapp worker: %v", err)
		}
		return
	}

	if err := whatsappworker.RunLoop(ctx, deps); err != nil && err != context.Canceled {
		log.Fatalf("whatsapp worker: %v", err)
	}
	log.Println("whatsapp worker stopped")
}
