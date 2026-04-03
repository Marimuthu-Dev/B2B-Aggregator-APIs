package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// FitnessCertWorkerConfig drives the fitness-certificate background worker (HTML → PDF, merge, blob, DB).
type FitnessCertWorkerConfig struct {
	// TemplateDir holds certificate_<ClientTypeID>.html files.
	TemplateDir string
	// ActorUserID is written to tbl_Leads.LastUpdatedBy and tbl_LeadsHistory.CreatedBy.
	// Zero is allowed as the default system actor when no explicit user ID is configured.
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
		TemplateDir:         strings.TrimSpace(getEnv("FITNESS_CERT_TEMPLATE_DIR", "./templates")),
		PendingLeadStatusID: int8(getEnvAsInt("FITNESS_CERT_PENDING_LEAD_STATUS_ID", 9)),
		DoneLeadStatusID:    int8(getEnvAsInt("FITNESS_CERT_DONE_LEAD_STATUS_ID", 10)),
		PollInterval:        time.Duration(getEnvAsInt("FITNESS_CERT_POLL_INTERVAL_SECONDS", 300)) * time.Second,
		BatchSize:           getEnvAsInt("FITNESS_CERT_BATCH_SIZE", 10),
		ChromiumPath:        strings.TrimSpace(getEnv("CHROMIUM_PATH", "")),
		RunOnce:             getEnvAsBool("FITNESS_CERT_WORKER_RUN_ONCE", false),
	}
	if c.TemplateDir == "" {
		c.TemplateDir = "./templates"
	}
	resolvedTemplateDir, err := resolveFitnessWorkerPath(c.TemplateDir)
	if err != nil {
		return c, fmt.Errorf("resolve fitness worker template dir: %w", err)
	}
	c.TemplateDir = resolvedTemplateDir
	if c.ChromiumPath != "" {
		resolvedChromium, err := resolveAgainstExecutableDir(c.ChromiumPath)
		if err != nil {
			return c, fmt.Errorf("resolve CHROMIUM_PATH: %w", err)
		}
		c.ChromiumPath = resolvedChromium
	}
	if c.BatchSize <= 0 {
		c.BatchSize = 10
	}
	if c.PollInterval <= 0 {
		c.PollInterval = 5 * time.Minute
	}
	idStr := strings.TrimSpace(getEnv("FITNESS_CERT_WORKER_USER_ID", "0"))
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id < 0 {
		return c, errors.New("FITNESS_CERT_WORKER_USER_ID must be zero or a positive integer")
	}
	c.ActorUserID = id
	return c, nil
}

func resolveFitnessWorkerPath(path string) (string, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		trimmed = "./templates"
	}
	return resolveAgainstExecutableDir(trimmed)
}

// resolveAgainstExecutableDir joins non-absolute paths with the directory of os.Executable()
// so relative CHROMIUM_PATH and template dirs work when the process CWD is not the WebJob folder.
func resolveAgainstExecutableDir(trimmed string) (string, error) {
	if filepath.IsAbs(trimmed) {
		return filepath.Clean(trimmed), nil
	}
	execPath, err := os.Executable()
	if err != nil {
		return "", err
	}
	execDir := filepath.Dir(execPath)
	return filepath.Clean(filepath.Join(execDir, trimmed)), nil
}
