package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Environment string
	Port        int
	Domain      string
	DB          DBConfig
	JWT         JWTConfig
	Domains     DomainURLs
}

type DBConfig struct {
	Server                 string
	User                   string
	Password               string
	Database               string
	PoolMax                int
	PoolMin                int
	IdleTimeout            int
	Encrypt                bool
	TrustServerCertificate bool
}

type JWTConfig struct {
	Secret           string
	ExpiresIn        string
	RefreshExpiresIn string
}

type DomainURLs struct {
	Client   string
	Employee string
	Lab      string
}

var AppConfig *Config

func LoadConfig() {
	// Try common .env locations (cwd, repo root, or parent).
	if err := godotenv.Load(); err != nil {
		_ = godotenv.Load("../.env")
		_ = godotenv.Load("../../.env")
	}

	AppConfig = &Config{
		Environment: getEnv("ENVIRONMENT", "development"),
		Port:        getEnvAsInt("PORT", 5000),
		Domain:      getEnv("DOMAIN", ""),
		DB: DBConfig{
			Server:                 getEnv("DB_SERVER", ""),
			User:                   getEnv("DB_USER", ""),
			Password:               getEnv("DB_PASSWORD", ""),
			Database:               getEnv("DB_DATABASE_NAME", ""),
			PoolMax:                getEnvAsInt("DB_POOL_MAX", 10),
			PoolMin:                getEnvAsInt("DB_POOL_MIN", 0),
			IdleTimeout:            getEnvAsInt("DB_IDLE_TIMEOUT", 30000),
			Encrypt:                getEnvAsBool("DB_ENCRYPT", false),
			TrustServerCertificate: getEnvAsBool("DB_TRUST_SERVER_CERT", true),
		},
		JWT: JWTConfig{
			Secret:           getEnv("JWT_SECRET", "aggreator@123456@"),
			ExpiresIn:        getEnv("JWT_EXPIRES_IN", "24h"),
			RefreshExpiresIn: getEnv("JWT_REFRESH_EXPIRES_IN", "7d"),
		},
		Domains: DomainURLs{
			Client:   getEnv("CLIENT_DOMAIN_URL", ""),
			Employee: getEnv("EMPLOYEE_DOMAIN_URL", ""),
			Lab:      getEnv("LAB_DOMAIN_URL", ""),
		},
	}
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
