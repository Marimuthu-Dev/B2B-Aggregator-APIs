package config

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Environment string
	Port        int
	Domain      string
	DB          DBConfig
	JWT         JWTConfig
	Log         LogConfig
	Domains     DomainURLs
	AzureBlob   AzureBlobConfig
}

const defaultMoUMaxBytes = 5 * 1024 * 1024

// AzureBlobConfig holds MoU PDF blob settings (from environment).
//
// Environment variables:
//   - AZURE_STORAGE_CONNECTION_STRING — full Azure Storage connection string (optional; empty disables uploads)
//   - AZURE_CONTAINER_NAME — blob container name (default: legal-documents)
//   - MOU_MAX_UPLOAD_BYTES — max PDF size in bytes (default: 5242880 = 5 MiB)
//   - MOU_UPLOAD_TIMEOUT_SECONDS — per-upload timeout toward Azure (default: 60)
//   - MOU_SAS_TTL_MINUTES — lifetime of read-only SAS URLs for MoU download (default: 15, max: 1440)
type AzureBlobConfig struct {
	ConnectionString string
	ContainerName    string
	MoUMaxBytes      int64
	UploadTimeout    time.Duration
	MoUSASTTL        time.Duration
}

type DBConfig struct {
	Server                 string
	User                   string
	Password               string
	Database               string
	PoolMax                int // max open connections to the database
	PoolMin                int // max idle connections kept in the pool (reuse)
	IdleTimeout            int // max time a connection can be idle before closed (ms)
	ConnMaxLifetime        int // max time a connection may be reused (ms); 0 = no limit
	Encrypt                bool
	TrustServerCertificate bool
}

type JWTConfig struct {
	Secret           string
	ExpiresIn        string
	RefreshExpiresIn string
}

type LogConfig struct {
	Dir            string
	RetentionHours int
}

type DomainURLs struct {
	Client   string
	Employee string
	Lab      string
}

func LoadConfig() *Config {
	// Load .env from typical locations: cwd, src/.env when cwd is repo root, parents.
	_ = godotenv.Load()
	_ = godotenv.Load(".env")
	_ = godotenv.Load("src/.env")
	_ = godotenv.Load("../.env")
	_ = godotenv.Load("../../.env")

	return &Config{
		Environment: getEnv("ENVIRONMENT", "development"),
		Port:        getEnvAsInt("PORT", 8080),
		Domain:      getEnv("DOMAIN", ""),
		DB: DBConfig{
			Server:                 getEnv("DB_SERVER", ""),
			User:                   getEnv("DB_USER", ""),
			Password:               getEnv("DB_PASSWORD", ""),
			Database:               getEnv("DB_DATABASE_NAME", ""),
			PoolMax:                getEnvAsInt("DB_POOL_MAX", 25),
			PoolMin:                getEnvAsInt("DB_POOL_MIN", 5),
			IdleTimeout:            getEnvAsInt("DB_IDLE_TIMEOUT", 30000),
			ConnMaxLifetime:        getEnvAsInt("DB_CONN_MAX_LIFETIME_MS", 3600000),
			Encrypt:                getEnvAsBool("DB_ENCRYPT", true),
			TrustServerCertificate: getEnvAsBool("DB_TRUST_SERVER_CERT", false),
		},
		JWT: JWTConfig{
			Secret:           getEnv("JWT_SECRET", ""),
			ExpiresIn:        getEnv("JWT_EXPIRES_IN", "24h"),
			RefreshExpiresIn: getEnv("JWT_REFRESH_EXPIRES_IN", "7d"),
		},
		Log: LogConfig{
			Dir:            getEnv("LOG_DIR", "logs"),
			RetentionHours: getEnvAsInt("LOG_RETENTION_HOURS", 24),
		},
		Domains: DomainURLs{
			Client:   getEnv("CLIENT_DOMAIN_URL", ""),
			Employee: getEnv("EMPLOYEE_DOMAIN_URL", ""),
			Lab:      getEnv("LAB_DOMAIN_URL", ""),
		},
		AzureBlob: loadAzureBlobConfig(),
	}
}

func loadAzureBlobConfig() AzureBlobConfig {
	maxBytes := getEnvAsInt64("MOU_MAX_UPLOAD_BYTES", defaultMoUMaxBytes)
	if maxBytes <= 0 {
		maxBytes = defaultMoUMaxBytes
	}
	timeoutSec := getEnvAsInt("MOU_UPLOAD_TIMEOUT_SECONDS", 60)
	if timeoutSec <= 0 {
		timeoutSec = 60
	}
	sasMin := getEnvAsInt("MOU_SAS_TTL_MINUTES", 15)
	if sasMin <= 0 {
		sasMin = 15
	}
	if sasMin > 1440 {
		sasMin = 1440
	}
	container := strings.TrimSpace(getEnv("AZURE_CONTAINER_NAME", "legal-documents"))
	if container == "" {
		container = "legal-documents"
	}
	return AzureBlobConfig{
		ConnectionString: trimSurroundingQuotes(getEnv("AZURE_STORAGE_CONNECTION_STRING", "")),
		ContainerName:    container,
		MoUMaxBytes:      maxBytes,
		UploadTimeout:    time.Duration(timeoutSec) * time.Second,
		MoUSASTTL:        time.Duration(sasMin) * time.Minute,
	}
}

// trimSurroundingQuotes removes one pair of ASCII single or double quotes (common in .env files).
func trimSurroundingQuotes(s string) string {
	s = strings.TrimSpace(s)
	if len(s) < 2 {
		return s
	}
	if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
		return s[1 : len(s)-1]
	}
	return s
}

func getEnvAsInt64(key string, defaultValue int64) int64 {
	valueStr := getEnv(key, "")
	if valueStr == "" {
		return defaultValue
	}
	if value, err := strconv.ParseInt(valueStr, 10, 64); err == nil {
		return value
	}
	return defaultValue
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := getEnv(key, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	valueStr := getEnv(key, "")
	if value, err := strconv.ParseBool(valueStr); err == nil {
		return value
	}
	return defaultValue
}
