package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Port               string
	AppEnv             string
	DBHost             string
	DBPort             string
	DBUser             string
	DBPassword         string
	DBName             string
	JWTSecret          string
	Timezone           string
	FrontendURL        string
	CORSAllowedOrigins string
}

func LoadConfig() (*Config, error) {
	_ = godotenv.Load() // Ignore error if .env is missing in production

	cfg := &Config{
		Port:               getEnv("PORT", "8080"),
		AppEnv:             getEnv("APP_ENV", "development"),
		DBHost:             getEnv("DB_HOST", "127.0.0.1"),
		DBPort:             getEnv("DB_PORT", "3306"),
		DBUser:             getEnv("DB_USER", "root"),
		DBPassword:         getEnv("DB_PASSWORD", ""),
		DBName:             getEnv("DB_NAME", "trust_management_db"),
		JWTSecret:          getEnv("JWT_SECRET", ""),
		Timezone:           getEnv("TIMEZONE", "Asia/Kolkata"),
		FrontendURL:        getEnv("FRONTEND_URL", "http://localhost:5173"),
		CORSAllowedOrigins: getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:5173,http://127.0.0.1:5173,http://localhost:3000"),
	}

	// Secrets must never have a hardcoded fallback in source — fail fast instead.
	if cfg.DBPassword == "" {
		return nil, fmt.Errorf("DB_PASSWORD must be set via environment/.env")
	}
	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET must be set via environment/.env")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val
	}
	return fallback
}

func (c *Config) GetDSN() string {
	return c.DBUser + ":" + c.DBPassword + "@tcp(" + c.DBHost + ":" + c.DBPort + ")/" + c.DBName + "?charset=utf8mb4&parseTime=True&loc=Local"
}

func (c *Config) GetPortInt() int {
	p, err := strconv.Atoi(c.Port)
	if err != nil {
		return 8080
	}
	return p
}
