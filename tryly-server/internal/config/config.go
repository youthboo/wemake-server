package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port                string
	PublicBaseURL       string
	CloudinaryURL       string
	CloudinaryCloudName string
	CloudinaryAPIKey    string
	CloudinaryAPISecret string
	DatabaseURL         string
	DBHost              string
	DBPort              string
	DBUser              string
	DBPassword          string
	DBName              string
	DBSSLMode           string
	Environment         string
	JWTSecret           string
	CORSOrigins         string
	SlipOKBranchID      string
	SlipOKAPIKey        string
	SlipOKAutoApproveCap float64
}

func LoadConfig() (*Config, error) {
	cfg := &Config{
		Port:                getEnv("PORT", "3000"),
		PublicBaseURL:       strings.TrimRight(getEnv("PUBLIC_BASE_URL", ""), "/"),
		CloudinaryURL:       getEnv("CLOUDINARY_URL", ""),
		CloudinaryCloudName: getEnv("CLOUDINARY_CLOUD_NAME", ""),
		CloudinaryAPIKey:    getEnv("CLOUDINARY_API_KEY", ""),
		CloudinaryAPISecret: getEnv("CLOUDINARY_API_SECRET", ""),
		DatabaseURL:         getEnv("DATABASE_URL", ""),
		DBHost:              getEnv("DB_HOST", "localhost"),
		DBPort:              getEnv("DB_PORT", "5432"),
		DBUser:              getEnv("DB_USER", "postgres"),
		DBPassword:          getEnv("DB_PASSWORD", ""),
		DBName:              getEnv("DB_NAME", "postgres"),
		DBSSLMode:           getEnv("DB_SSLMODE", "disable"),
		Environment:         getEnv("ENV", "development"),
		JWTSecret:           getEnv("JWT_SECRET", "your-secret-key"),
		CORSOrigins:         getEnv("CORS_ORIGINS", "*"),
		SlipOKBranchID:      getEnv("SLIPOK_BRANCH_ID", ""),
		SlipOKAPIKey:        getEnv("SLIPOK_API_KEY", ""),
		SlipOKAutoApproveCap: getEnvFloat("SLIPOK_AUTO_APPROVE_CAP", 50000),
	}

	if err := cfg.validateForProduction(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// validateForProduction fails fast when required secrets are missing or left at
// their insecure defaults, but ONLY when ENV=production. Development/demo runs
// (the default) are unaffected, so this does not change local behavior.
func (c *Config) validateForProduction() error {
	if strings.ToLower(c.Environment) != "production" {
		return nil
	}

	var problems []string
	if c.JWTSecret == "" || c.JWTSecret == "your-secret-key" {
		problems = append(problems, "JWT_SECRET must be set to a strong non-default value")
	}
	if c.DatabaseURL == "" && c.DBPassword == "" {
		problems = append(problems, "DATABASE_URL (or DB_PASSWORD) must be set")
	}
	if c.CORSOrigins == "" || c.CORSOrigins == "*" {
		problems = append(problems, "CORS_ORIGINS must be an explicit allowlist, not \"*\"")
	}

	if len(problems) > 0 {
		return fmt.Errorf("insecure production config: %s", strings.Join(problems, "; "))
	}
	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if f, err := strconv.ParseFloat(value, 64); err == nil {
			return f
		}
	}
	return defaultValue
}

func (c *Config) GetDSN() string {
	// Always pin the session timezone to Bangkok so that TIMESTAMP WITHOUT
	// TIME ZONE columns and PostgreSQL NOW() / CURRENT_TIMESTAMP produce
	// Bangkok wall-clock values on every server environment (local or cloud).
	//
	// IMPORTANT: lib/pq SILENTLY IGNORES a `TimeZone=` / `timezone=` DSN param —
	// the only reliable way to set the session GUC is via libpq `options`
	// (`-c timezone=...`). Using the wrong form leaves the session at UTC and
	// stores UTC wall-clock, which the FE then renders 7h off.
	const bangkokTZ = "Asia/Bangkok"
	if c.DatabaseURL != "" {
		// URL-style DSN (e.g. Render DATABASE_URL): pass options url-encoded.
		sep := "?"
		if strings.Contains(c.DatabaseURL, "?") {
			sep = "&"
		}
		return c.DatabaseURL + sep + "options=-c%20timezone%3DAsia%2FBangkok"
	}
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s options='-c timezone=%s'",
		c.DBHost,
		c.DBPort,
		c.DBUser,
		c.DBPassword,
		c.DBName,
		c.DBSSLMode,
		bangkokTZ,
	)
}
