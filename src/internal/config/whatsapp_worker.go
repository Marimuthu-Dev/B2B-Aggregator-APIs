package config

import (
	"errors"
	"strings"
	"time"
)

// WhatsAppWorkerConfig drives the WhatsApp background worker.
// Database connectivity uses the same DB_* settings as the API and other workers (see LoadConfig / ConnectDatabase).
type WhatsAppWorkerConfig struct {
	// SingleBatch runs one batch (SelectPendingBatch + sends) and exits — for App Service WebJobs Triggered + Scheduled.
	// When false, RunLoop polls until SIGINT/SIGTERM (Continuous WebJob or local dev).
	SingleBatch bool
	// APIKey is the cpaaslink.com API key for authentication.
	APIKey string
	// APIEndpoint is the WhatsApp API endpoint URL (e.g., https://cpaaslink.com/api/whatsapp/public/apikey)
	APIEndpoint string
	// UseMMLite determines whether to use the MM Lite endpoint instead of the regular endpoint.
	UseMMLite bool
	// TemplateName is the default WhatsApp template name to use for messages.
	TemplateName string
	// CampaignName is the default campaign name for WhatsApp messages.
	CampaignName string
	BatchSize    int
	PollInterval time.Duration
	IdleWait     time.Duration
	SendTimeout  time.Duration
}

// LoadWhatsAppWorkerConfig reads worker settings from the environment.
// Call after godotenv.Load (same pattern as LoadEmailWorkerConfig and LoadFitnessCertWorkerConfig).
func LoadWhatsAppWorkerConfig() (WhatsAppWorkerConfig, error) {
	c := WhatsAppWorkerConfig{
		SingleBatch:   getEnvAsBool("WHATSAPP_WORKER_SINGLE_BATCH", false),
		APIKey:        strings.TrimSpace(getEnv("WHATSAPP_API_KEY", "")),
		APIEndpoint:   strings.TrimSpace(getEnv("WHATSAPP_API_ENDPOINT", "https://cpaaslink.com/api/whatsapp/public/apikey")),
		UseMMLite:     getEnvAsBool("WHATSAPP_USE_MM_LITE", false),
		TemplateName:  strings.TrimSpace(getEnv("WHATSAPP_TEMPLATE_NAME", "")),
		CampaignName:  strings.TrimSpace(getEnv("WHATSAPP_CAMPAIGN_NAME", "default_campaign")),
		BatchSize:     getEnvAsInt("WHATSAPP_BATCH_SIZE", 25),
		PollInterval:  time.Duration(getEnvAsInt("WHATSAPP_POLL_INTERVAL_SECONDS", 120)) * time.Second,
		IdleWait:      time.Duration(getEnvAsInt("WHATSAPP_IDLE_WAIT_SECONDS", 60)) * time.Second,
		SendTimeout:   time.Duration(getEnvAsInt("WHATSAPP_SEND_TIMEOUT_SECONDS", 60)) * time.Second,
	}
	if c.APIKey == "" {
		return c, errors.New("WHATSAPP_API_KEY is required")
	}
	if c.APIEndpoint == "" {
		return c, errors.New("WHATSAPP_API_ENDPOINT is required")
	}
	if c.TemplateName == "" {
		return c, errors.New("WHATSAPP_TEMPLATE_NAME is required")
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
	
	// Adjust endpoint if using MM Lite
	if c.UseMMLite {
		c.APIEndpoint = strings.Replace(c.APIEndpoint, "/apikey", "/mm-lite", 1)
	}
	
	return c, nil
}
