package redis

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

type Client struct {
	*redis.Client
	config *Config
}

func NewClient(cfg *Config) (*Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:            fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password:        cfg.Password,
		DB:              cfg.DB,
		MaxRetries:      cfg.MaxRetries,
		PoolSize:        cfg.PoolSize,
		MinIdleConns:    cfg.MinIdleConns,
		ConnMaxLifetime: cfg.ConnMaxLifetime,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	log.Printf("Connected to Redis at %s:%d", cfg.Host, cfg.Port)

	return &Client{
		Client: rdb,
		config: cfg,
	}, nil
}

func (c *Client) Close() error {
	return c.Client.Close()
}

func (c *Client) Health(ctx context.Context) error {
	return c.Client.Ping(ctx).Err()
}

func (c *Client) Stats() map[string]interface{} {
	poolStats := c.Client.PoolStats()
	return map[string]interface{}{
		"hits":        poolStats.Hits,
		"misses":      poolStats.Misses,
		"timeouts":    poolStats.Timeouts,
		"total_conns": poolStats.TotalConns,
		"idle_conns":  poolStats.IdleConns,
		"stale_conns": poolStats.StaleConns,
	}
}
