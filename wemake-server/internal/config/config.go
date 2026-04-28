package config

import (
	"fmt"
	"os"
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
}

func LoadConfig() (*Config, error) {
	return &Config{
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
	}, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func (c *Config) GetDSN() string {
	// Always pin the session timezone to Bangkok so that TIMESTAMP WITHOUT
	// TIME ZONE columns and PostgreSQL NOW() / CURRENT_TIMESTAMP produce
	// Bangkok wall-clock values on every server environment (local or cloud).
	const bangkokTZ = "Asia/Bangkok"
	if c.DatabaseURL != "" {
		// Append as query parameter for URL-style DSNs (e.g. Render DATABASE_URL).
		sep := "?"
		if strings.Contains(c.DatabaseURL, "?") {
			sep = "&"
		}
		return c.DatabaseURL + sep + "TimeZone=" + bangkokTZ
	}
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s TimeZone=%s",
		c.DBHost,
		c.DBPort,
		c.DBUser,
		c.DBPassword,
		c.DBName,
		c.DBSSLMode,
		bangkokTZ,
	)
}
