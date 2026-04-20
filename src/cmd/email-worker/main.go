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

	"b2b-diagnostic-aggregator/apis/internal/acsemail"
	"b2b-diagnostic-aggregator/apis/internal/config"
	"b2b-diagnostic-aggregator/apis/internal/logging"
	"b2b-diagnostic-aggregator/apis/internal/repository"
	emailworker "b2b-diagnostic-aggregator/apis/internal/worker/email"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	_ = godotenv.Load(".env")
	_ = godotenv.Load("src/.env")
	_ = godotenv.Load("../.env")

	appCfg := config.LoadConfig()
	wc, err := config.LoadEmailWorkerConfig()
	if err != nil {
		log.Fatalf("email worker config: %v", err)
	}

	var logOut io.Writer = os.Stderr
	if logWriter, lwErr := logging.NewHourlyFileWriter(logging.Config{
		Dir:            appCfg.Log.Dir,
		RetentionHours: appCfg.Log.RetentionHours,
		Prefix:         "email-worker",
	}); lwErr != nil {
		log.Printf("email worker: file logging disabled (%v); using stderr only", lwErr)
		log.SetOutput(os.Stderr)
	} else {
		logOut = io.MultiWriter(os.Stderr, logWriter)
		log.SetOutput(logOut)
	}
	log.SetFlags(log.LstdFlags)

	logger := slog.New(slog.NewTextHandler(logOut, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)
	logger.Info("email worker starting",
		slog.String("logDir", appCfg.Log.Dir),
		slog.Int("logRetentionHours", appCfg.Log.RetentionHours),
		slog.Bool("singleBatch", wc.SingleBatch),
		slog.String("senderDisplayName", wc.ACSSenderDisplayName),
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

	repo, err := repository.NewEmailOutboxRepository(context.Background(), sqlDB)
	if err != nil {
		log.Fatalf("email outbox repository: %v", err)
	}

	httpClient := &http.Client{Timeout: wc.SendTimeout + 5*time.Second}
	sender, err := acsemail.NewService(acsemail.Config{
		ConnectionString:  wc.ACSConnectionString,
		APIVersion:        wc.ACSAPIVersion,
		HTTPClient:        httpClient,
		SenderDisplayName: wc.ACSSenderDisplayName,
	})
	if err != nil {
		log.Fatalf("acs email: %v", err)
	}

	deps := emailworker.Deps{
		Repo:   repo,
		Sender: sender,
		Config: wc,
		Log:    logger,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if wc.SingleBatch {
		if _, err := emailworker.RunOnce(ctx, deps); err != nil {
			log.Fatalf("email worker: %v", err)
		}
		return
	}

	if err := emailworker.RunLoop(ctx, deps); err != nil && err != context.Canceled {
		log.Fatalf("email worker: %v", err)
	}
	log.Println("email worker stopped")
}
