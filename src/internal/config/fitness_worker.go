package config

import (
	"errors"
	"strconv"
	"strings"
	"time"
)

// FitnessCertWorkerConfig drives the fitness-certificate background worker (HTML → PDF, merge, blob, DB).
type FitnessCertWorkerConfig struct {
	// TemplateDir holds certificate_<ClientTypeID>.html files.
	TemplateDir string
	// ActorUserID is written to tbl_Leads.LastUpdatedBy and tbl_LeadsHistory.CreatedBy.
	ActorUserID int64
	// PendingLeadStatusID e.g. 9 — must match LeadStatusID for rows awaiting certificate generation.
	PendingLeadStatusID int8
	// DoneLeadStatusID e.g. 10 — applied after successful merge and upload.
	DoneLeadStatusID int8
	PollInterval     time.Duration
	BatchSize        int
	// ChromiumPath optional path to chromium/chrome binary (empty = search PATH).
	ChromiumPath string
	// RunOnce exits after one batch (for cron/systemd oneshot).
	RunOnce bool
}

// LoadFitnessCertWorkerConfig reads worker settings from the environment.
// Call after godotenv.Load (e.g. same as API).
func LoadFitnessCertWorkerConfig() (FitnessCertWorkerConfig, error) {
	c := FitnessCertWorkerConfig{
		TemplateDir:         strings.TrimSpace(getEnv("FITNESS_CERT_TEMPLATE_DIR", "templates")),
		PendingLeadStatusID: int8(getEnvAsInt("FITNESS_CERT_PENDING_LEAD_STATUS_ID", 9)),
		DoneLeadStatusID:    int8(getEnvAsInt("FITNESS_CERT_DONE_LEAD_STATUS_ID", 10)),
		PollInterval:        time.Duration(getEnvAsInt("FITNESS_CERT_POLL_INTERVAL_SECONDS", 300)) * time.Second,
		BatchSize:           getEnvAsInt("FITNESS_CERT_BATCH_SIZE", 10),
		ChromiumPath:        strings.TrimSpace(getEnv("CHROMIUM_PATH", "")),
		RunOnce:             getEnvAsBool("FITNESS_CERT_WORKER_RUN_ONCE", false),
	}
	if c.TemplateDir == "" {
		c.TemplateDir = "templates"
	}
	if c.BatchSize <= 0 {
		c.BatchSize = 10
	}
	if c.PollInterval <= 0 {
		c.PollInterval = 5 * time.Minute
	}
	idStr := strings.TrimSpace(getEnv("FITNESS_CERT_WORKER_USER_ID", ""))
	if idStr == "" {
		return c, errors.New("FITNESS_CERT_WORKER_USER_ID is required (positive int64 system user for audit columns)")
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		return c, errors.New("FITNESS_CERT_WORKER_USER_ID must be a positive integer")
	}
	c.ActorUserID = id
	return c, nil
}
