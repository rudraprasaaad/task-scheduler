package redis

import "time"

type Config struct {
	Host            string
	Port            int
	Password        string
	DB              int
	MaxRetries      int
	PoolSize        int
	MinIdleConns    int
	ConnMaxLifetime time.Duration
}
