package config

import (
	"fmt"
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
	Email       OutboundEmailConfig
}

// OutboundEmailConfig is used when the API inserts rows into {DB_SCHEMA}.tbl_Emails.
type OutboundEmailConfig struct {
	FromAddress  string
	CCAddress    string
	BCCAddress   string
	SupportPhone string
	SupportEmail string
	LogoURL      string
}

const defaultMoUMaxBytes = 5 * 1024 * 1024

// AzureBlobConfig holds MoU and diagnostic-report PDF blob settings (from environment).
//
// MoU / legal documents — storage auth (use either connection string or account + key):
//   - AZURE_STORAGE_CONNECTION_STRING — full connection string (optional)
//   - AZURE_STORAGE_ACCOUNT + AZURE_STORAGE_KEY — built into a connection string if the full string is unset
//   - AZURE_STORAGE_ENDPOINT_SUFFIX — optional (default: core.windows.net)
//   - AZURE_LEGAL_CONTAINER_NAME — MoU PDF container (default: legal-documents)
//   - MOU_MAX_UPLOAD_BYTES — max MoU PDF size in bytes (default: 5242880 = 5 MiB)
//   - MOU_UPLOAD_TIMEOUT_SECONDS — MoU upload timeout toward Azure (default: 60)
//   - MOU_SAS_TTL_MINUTES — read-only SAS lifetime for MoU download (default: 15, max: 1440)
//
// Blood test / diagnostic reports (fallback to MoU settings above when unset or invalid):
//   - AZURE_DIAGNOSTIC_REPORTS_CONTAINER (default: diagnostic-reports)
//   - DIAGNOSTIC_REPORTS_MAX_UPLOAD_BYTES
//   - DIAGNOSTIC_REPORTS_UPLOAD_TIMEOUT_SECONDS
//   - DIAGNOSTIC_REPORTS_SAS_TTL_MINUTES (max: 1440)
type AzureBlobConfig struct {
	ConnectionString string
	// StorageAccountName and StorageAccountKey are the preferred way to authenticate (see AZURE_STORAGE_*).
	// Used with NewClientWithSharedKeyCredential and for SAS signing; never pass the raw connection string into SAS APIs.
	StorageAccountName             string
	StorageAccountKey              string
	BlobEndpointSuffix             string // e.g. core.windows.net (no https://)
	ContainerName                  string
	DiagnosticReportsContainer     string
	MoUMaxBytes                    int64
	UploadTimeout                  time.Duration
	MoUSASTTL                      time.Duration
	DiagnosticReportsMaxBytes      int64
	DiagnosticReportsUploadTimeout time.Duration
	DiagnosticReportsSASTTL        time.Duration
}

type DBConfig struct {
	Server                 string
	User                   string
	Password               string
	Database               string
	Schema                 string // SQL Server schema prefix (DB_SCHEMA); e.g. MediAdmin
	PoolMax                int    // max open connections to the database
	PoolMin                int    // max idle connections kept in the pool (reuse)
	IdleTimeout            int    // max time a connection can be idle before closed (ms)
	ConnMaxLifetime        int    // max time a connection may be reused (ms); 0 = no limit
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

	dbSchema := strings.TrimSpace(getEnv("DB_SCHEMA", "MediAdmin"))
	if dbSchema == "" {
		dbSchema = "MediAdmin"
	}

	return &Config{
		Environment: getEnv("ENVIRONMENT", "development"),
		Port:        getEnvAsInt("PORT", 8080),
		Domain:      getEnv("DOMAIN", ""),
		DB: DBConfig{
			Server:                 getEnv("DB_SERVER", ""),
			User:                   getEnv("DB_USER", ""),
			Password:               getEnv("DB_PASSWORD", ""),
			Database:               getEnv("DB_DATABASE_NAME", ""),
			Schema:                 dbSchema,
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
		Email:     loadOutboundEmailConfig(),
	}
}

func loadOutboundEmailConfig() OutboundEmailConfig {
	from := strings.TrimSpace(getEnv("EMAIL_FROM_ADDRESS", "support@urmediconnect.com"))
	if from == "" {
		from = "support@urmediconnect.com"
	}
	logo := strings.TrimSpace(getEnv("EMAIL_LOGO_URL", "https://urmediconnect.com/img/logo.jpeg"))
	if logo == "" {
		logo = "https://urmediconnect.com/img/logo.jpeg"
	}
	supportEmail := strings.TrimSpace(getEnv("EMAIL_SUPPORT_EMAIL", "support@urmediconnect.com"))
	if supportEmail == "" {
		supportEmail = "support@urmediconnect.com"
	}
	supportPhone := strings.TrimSpace(getEnv("EMAIL_SUPPORT_PHONE", "+91 9036302806"))
	if supportPhone == "" {
		supportPhone = "+91 9036302806"
	}
	return OutboundEmailConfig{
		FromAddress:  from,
		CCAddress:    strings.TrimSpace(getEnv("EMAIL_CC_ADDRESS", "")),
		BCCAddress:   strings.TrimSpace(getEnv("EMAIL_BCC_ADDRESS", "")),
		SupportPhone: supportPhone,
		SupportEmail: supportEmail,
		LogoURL:      logo,
	}
}

// resolveAzureStorageConnectionString prefers AZURE_STORAGE_CONNECTION_STRING;
// otherwise builds one from AZURE_STORAGE_ACCOUNT, AZURE_STORAGE_KEY, and optional AZURE_STORAGE_ENDPOINT_SUFFIX.
func resolveAzureStorageConnectionString() string {
	conn := trimSurroundingQuotes(getEnv("AZURE_STORAGE_CONNECTION_STRING", ""))
	if strings.TrimSpace(conn) != "" {
		return conn
	}
	acc := trimSurroundingQuotes(getEnv("AZURE_STORAGE_ACCOUNT", ""))
	key := trimSurroundingQuotes(getEnv("AZURE_STORAGE_KEY", ""))
	if strings.TrimSpace(acc) == "" || strings.TrimSpace(key) == "" {
		return ""
	}
	suffix := trimSurroundingQuotes(getEnv("AZURE_STORAGE_ENDPOINT_SUFFIX", "core.windows.net"))
	if suffix == "" {
		suffix = "core.windows.net"
	}
	return fmt.Sprintf(
		"DefaultEndpointsProtocol=https;AccountName=%s;AccountKey=%s;EndpointSuffix=%s",
		acc, key, suffix,
	)
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
	container := strings.TrimSpace(getEnv("AZURE_LEGAL_CONTAINER_NAME", "legal-documents"))
	if container == "" {
		container = "legal-documents"
	}
	reportsContainer := strings.TrimSpace(getEnv("AZURE_DIAGNOSTIC_REPORTS_CONTAINER", "diagnostic-reports"))
	if reportsContainer == "" {
		reportsContainer = "diagnostic-reports"
	}

	reportMax := getEnvAsInt64("DIAGNOSTIC_REPORTS_MAX_UPLOAD_BYTES", maxBytes)
	if reportMax <= 0 {
		reportMax = maxBytes
	}
	reportTimeoutSec := getEnvAsInt("DIAGNOSTIC_REPORTS_UPLOAD_TIMEOUT_SECONDS", timeoutSec)
	if reportTimeoutSec <= 0 {
		reportTimeoutSec = timeoutSec
	}
	reportSASMin := getEnvAsInt("DIAGNOSTIC_REPORTS_SAS_TTL_MINUTES", sasMin)
	if reportSASMin <= 0 {
		reportSASMin = sasMin
	}
	if reportSASMin > 1440 {
		reportSASMin = 1440
	}

	conn := resolveAzureStorageConnectionString()
	suffix := trimSurroundingQuotes(getEnv("AZURE_STORAGE_ENDPOINT_SUFFIX", "core.windows.net"))
	if strings.TrimSpace(suffix) == "" {
		suffix = "core.windows.net"
	}
	return AzureBlobConfig{
		ConnectionString:               conn,
		StorageAccountName:             trimSurroundingQuotes(getEnv("AZURE_STORAGE_ACCOUNT", "")),
		StorageAccountKey:              trimSurroundingQuotes(getEnv("AZURE_STORAGE_KEY", "")),
		BlobEndpointSuffix:             suffix,
		ContainerName:                  container,
		DiagnosticReportsContainer:     reportsContainer,
		MoUMaxBytes:                    maxBytes,
		UploadTimeout:                  time.Duration(timeoutSec) * time.Second,
		MoUSASTTL:                      time.Duration(sasMin) * time.Minute,
		DiagnosticReportsMaxBytes:      reportMax,
		DiagnosticReportsUploadTimeout: time.Duration(reportTimeoutSec) * time.Second,
		DiagnosticReportsSASTTL:        time.Duration(reportSASMin) * time.Minute,
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
