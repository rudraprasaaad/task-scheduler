package database

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Host            string
	Port            int
	User            string
	Password        string
	DBname          string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

func NewConfig() *Config {
	return &Config{
		Host:            getEnv("DB_HOST", "localhost"),
		Port:            getEnvAsInt("DB_PORT", 5432),
		User:            getEnv("DB_USER", "postgres"),
		Password:        getEnv("DB_PASSWORD", "password"),
		DBname:          getEnv("DB_NAME", "task_scheduler"),
		SSLMode:         getEnv("DB_SSL_MODE", "disable"),
		MaxOpenConns:    getEnvAsInt("DB_MAX_OPENS_CONNS", 25),
		MaxIdleConns:    getEnvAsInt("DB_MAX_IDLE_CONNS", 5),
		ConnMaxLifetime: getEnvAsDuration("DB_CONN_MAX_LIFETIME", 5*time.Minute),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return defaultValue
}

func (c *Config) ConnectionString() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s", c.Host, c.Port, c.User, c.Password, c.DBname, c.SSLMode)
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}

	return defaultValue
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}

	return defaultValue
}
