package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	redisClient "github.com/rudraprasaaad/task-scheduler/internal/redis"
)

type RedisCache struct {
	client *redisClient.Client
	prefix string
}

func NewRedisCache(client *redisClient.Client, prefix string) *RedisCache {
	return &RedisCache{
		client: client,
		prefix: prefix,
	}
}

func (rc *RedisCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal valueL %w", err)
	}

	fullKey := rc.prefix + key
	return rc.client.Set(ctx, fullKey, data, expiration).Err()
}

func (rc *RedisCache) Get(ctx context.Context, key string, dest interface{}) error {
	fullKey := rc.prefix + key
	data, err := rc.client.Get(ctx, fullKey).Result()
	if err != nil {
		return err
	}

	return json.Unmarshal([]byte(data), dest)
}

func (rc *RedisCache) Delete(ctx context.Context, key string) error {
	fullKey := rc.prefix + key
	return rc.client.Del(ctx, fullKey).Err()
}

func (rc *RedisCache) Exists(ctx context.Context, key string) (bool, error) {
	fullKey := rc.prefix + key
	count, err := rc.client.Exists(ctx, fullKey).Result()
	return count > 0, err
}

func (rc *RedisCache) SetWorkerStats(ctx context.Context, workerID string, stats map[string]interface{}) error {
	key := fmt.Sprintf("worker:stats:%s", workerID)
	return rc.Set(ctx, key, stats, 5*time.Minute)
}

func (rc *RedisCache) GetWorkerStats(ctx context.Context, workerID string) (map[string]interface{}, error) {
	key := fmt.Sprintf("worker:stats:%s", workerID)
	var stats map[string]interface{}
	err := rc.Get(ctx, key, stats)
	return stats, err
}

func (rc *RedisCache) CacheTaskStats(ctx context.Context, stats map[string]int) error {
	return rc.Set(ctx, "task:stats", stats, 1*time.Minute)
}

func (rc *RedisCache) GetCachedTaskStats(ctx context.Context) (map[string]int, error) {
	var stats map[string]int
	err := rc.Get(ctx, "task:stats", &stats)
	return stats, err
}
