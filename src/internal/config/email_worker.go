package config

import (
	"errors"
	"strings"
	"time"
)

// EmailWorkerConfig drives the ACS email outbox background worker.
type EmailWorkerConfig struct {
	DBConnString        string
	ACSConnectionString string
	BatchSize           int
	PollInterval        time.Duration
	IdleWait            time.Duration
	SendTimeout         time.Duration
	ACSAPIVersion       string
}

// LoadEmailWorkerConfig reads worker settings from the environment.
// Call after godotenv.Load (same pattern as LoadFitnessCertWorkerConfig).
func LoadEmailWorkerConfig() (EmailWorkerConfig, error) {
	c := EmailWorkerConfig{
		DBConnString:        strings.TrimSpace(getEnv("DB_CONN_STRING", "")),
		ACSConnectionString: strings.TrimSpace(getEnv("ACS_CONNECTION_STRING", "")),
		BatchSize:           getEnvAsInt("EMAIL_BATCH_SIZE", 25),
		PollInterval:        time.Duration(getEnvAsInt("EMAIL_POLL_INTERVAL_SECONDS", 120)) * time.Second,
		IdleWait:            time.Duration(getEnvAsInt("EMAIL_IDLE_WAIT_SECONDS", 60)) * time.Second,
		SendTimeout:         time.Duration(getEnvAsInt("EMAIL_SEND_TIMEOUT_SECONDS", 60)) * time.Second,
		ACSAPIVersion:       strings.TrimSpace(getEnv("ACS_EMAIL_API_VERSION", "")),
	}
	if c.DBConnString == "" {
		return c, errors.New("DB_CONN_STRING is required")
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
