package config

import (
	"os"
	"strconv"
	"time"

	"github.com/rudraprasaaad/task-scheduler/internal/database"
	"github.com/rudraprasaaad/task-scheduler/internal/redis"
)

type Config struct {
	Port        string
	Environment string
	LogLevel    string
	MaxWorkers  int

	Database database.Config

	Redis redis.Config
}

func Load() *Config {
	return &Config{
		Port:        getEnv("PORT", "8080"),
		Environment: getEnv("ENVIRONMENT", "development"),
		LogLevel:    getEnv("LOG_LEVEL", "info"),
		MaxWorkers:  getEnvAsInt("MAX_WORKERS", 10),

		Database: database.Config{
			Host:            getEnv("DB_HOST", "localhost"),
			Port:            getEnvAsInt("DB_PORT", 5432),
			User:            getEnv("DB_USER", "postgres"),
			Password:        getEnv("DB_PASSWORD", "password"),
			DBName:          getEnv("DB_NAME", "task_scheduler"),
			SSLMode:         getEnv("DB_SSL_MODE", "disable"),
			MaxOpenConns:    getEnvAsInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getEnvAsInt("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: getEnvAsDuration("DB_CONN_MAX_LIFETIME", "5m"),
		},

		Redis: redis.Config{
			Host:            getEnv("REDIS_HOST", "localhost"),
			Port:            getEnvAsInt("REDIS_PORT", 6379),
			Password:        getEnv("REDIS_PASSWORD", ""),
			DB:              getEnvAsInt("REDIS_DB", 0),
			MaxRetries:      getEnvAsInt("REDIS_MAX_RETRIES", 3),
			PoolSize:        getEnvAsInt("REDIS_POOL_SIZE", 20),
			MinIdleConns:    getEnvAsInt("REDIS_MIN_IDLE_CONNS", 5),
			ConnMaxLifetime: getEnvAsDuration("REDIS_CONN_MAX_LIFETIME", "10m"),
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvAsDuration(key string, defaultValue string) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	duration, _ := time.ParseDuration(defaultValue)
	return duration
}
