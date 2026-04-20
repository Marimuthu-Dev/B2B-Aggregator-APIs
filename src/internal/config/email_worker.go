package config

import (
	"errors"
	"strings"
	"time"
)

// EmailWorkerConfig drives the ACS email outbox background worker.
// Database connectivity uses the same DB_* settings as the API and fitness-worker (see LoadConfig / ConnectDatabase).
type EmailWorkerConfig struct {
	// SingleBatch runs one batch (SelectPendingBatch + sends) and exits — for App Service WebJobs Triggered + Scheduled.
	// When false, RunLoop polls until SIGINT/SIGTERM (Continuous WebJob or local dev).
	SingleBatch bool
	// ACSConnectionString is the Azure Communication Services resource connection string (endpoint + access key).
	ACSConnectionString string
	// ACSSenderDisplayName is the From "friendly name" (e.g. UrMediConnect). May also need Portal Mail From config.
	ACSSenderDisplayName string
	BatchSize            int
	PollInterval        time.Duration
	IdleWait            time.Duration
	SendTimeout         time.Duration
	ACSAPIVersion       string
}

// LoadEmailWorkerConfig reads worker settings from the environment.
// Call after godotenv.Load (same pattern as LoadFitnessCertWorkerConfig).
func LoadEmailWorkerConfig() (EmailWorkerConfig, error) {
	c := EmailWorkerConfig{
		SingleBatch:          getEnvAsBool("EMAIL_WORKER_SINGLE_BATCH", false),
		ACSConnectionString:  strings.TrimSpace(getEnv("ACS_CONNECTION_STRING", "")),
		ACSSenderDisplayName: strings.TrimSpace(getEnv("ACS_SENDER_DISPLAY_NAME", "")),
		BatchSize:            getEnvAsInt("EMAIL_BATCH_SIZE", 25),
		PollInterval:        time.Duration(getEnvAsInt("EMAIL_POLL_INTERVAL_SECONDS", 120)) * time.Second,
		IdleWait:            time.Duration(getEnvAsInt("EMAIL_IDLE_WAIT_SECONDS", 60)) * time.Second,
		SendTimeout:         time.Duration(getEnvAsInt("EMAIL_SEND_TIMEOUT_SECONDS", 60)) * time.Second,
		ACSAPIVersion:       strings.TrimSpace(getEnv("ACS_EMAIL_API_VERSION", "")),
	}
	if c.ACSConnectionString == "" {
		return c, errors.New("ACS_CONNECTION_STRING is required")
	}
	if c.BatchSize < 1 {
		c.BatchSize = 1
	}
	if c.PollInterval < time.Second {
		c.PollInterval = time.Second
	}
	if c.IdleWait < time.Second {
		c.IdleWait = time.Second
	}
	if c.SendTimeout < time.Second {
		c.SendTimeout = time.Second
	}
	return c, nil
}
