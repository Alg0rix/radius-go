package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	DBDSN                string
	HTTPAddr             string
	InternalSecret       string
	RADIUSAuthAddr       string
	RADIUSAcctAddr       string
	RADIUSCoAAddr        string
	EnableCoA            bool
	LogFormat            string
	StaleSessionTimeout  int
	DBRefreshInterval    int
	SessionCleanupPeriod int

	serviceName string
}

func Load(serviceName ...string) Config {
	_ = godotenv.Load()

	name := ""
	if len(serviceName) > 0 {
		name = serviceName[0]
	}

	logFormat := getEnv("LOG_FORMAT", "console")

	enableCoA := false
	if v := os.Getenv("ENABLE_COA"); v != "" {
		enableCoA, _ = strconv.ParseBool(v)
	}

	return Config{
		DBDSN:                getEnv("DB_DSN", ""),
		HTTPAddr:             fmt.Sprintf(":%d", getEnvInt("HTTP_PORT", 8083)),
		InternalSecret:       getEnv("INTERNAL_SECRET", ""),
		RADIUSAuthAddr:       fmt.Sprintf(":%d", getEnvInt("RADIUS_AUTH_PORT", 1812)),
		RADIUSAcctAddr:       fmt.Sprintf(":%d", getEnvInt("RADIUS_ACCT_PORT", 1813)),
		RADIUSCoAAddr:        fmt.Sprintf(":%d", getEnvInt("RADIUS_COA_PORT", 3799)),
		EnableCoA:            enableCoA,
		LogFormat:            logFormat,
		StaleSessionTimeout:  getEnvInt("STALE_SESSION_TIMEOUT", 86400),
		DBRefreshInterval:    getEnvInt("DB_REFRESH_INTERVAL", 60),
		SessionCleanupPeriod: getEnvInt("SESSION_CLEANUP_PERIOD", 300),
		serviceName:          name,
	}
}

func (c Config) Validate() error {
	if c.DBDSN == "" {
		return fmt.Errorf("DB_DSN is required")
	}
	if c.InternalSecret == "" {
		return fmt.Errorf("INTERNAL_SECRET is required")
	}
	return nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}