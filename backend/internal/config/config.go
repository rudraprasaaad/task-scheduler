package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port        string
	DatabaseURL string
	RedisURL    string
	MaxWorkers  int
	LogLevel    string
	Environment string
	Auth        AuthConfig
}

type AuthConfig struct {
	JWTSecret       string
	TokenExpiration time.Duration
}

func Load() *Config {
	tokenExp, err := time.ParseDuration(getEnv("JWT_EXPIRATION", "24h"))
	if err != nil {
		tokenExp = 24 * time.Hour
	}

	return &Config{
		Port:        getEnv("PORT", "8080"),
		DatabaseURL: getEnv("DATABASE_URL", ""),
		RedisURL:    getEnv("REDIS_URL", ""),
		MaxWorkers:  getEnvAsInt("MAX_WORKERS", 10),
		LogLevel:    getEnv("LOG_LEVEL", "info"),
		Environment: getEnv("ENVIRONMENT", "development"),
		Auth: AuthConfig{
			JWTSecret:       getEnv("JWT_SECRET_KEY", ""),
			TokenExpiration: tokenExp,
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
