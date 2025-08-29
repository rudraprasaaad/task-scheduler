package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rudraprasaaad/task-scheduler/internal/models"
	redisClient "github.com/rudraprasaaad/task-scheduler/internal/redis"
)

const (
	TaskQueueKey      = "task_scheduler:queue"
	TaskDataKeyPrefix = "task_scheduler:task:"
	TaskLockKeyPrefix = "task_scheduler:lock:"
	TaskUpdateChannel = "task_scheduler:updates"
)

type RedisQueue struct {
	client *redisClient.Client
	pubsub *redis.PubSub
}

func NewRedisQueue(client *redisClient.Client) *RedisQueue {
	return &RedisQueue{
		client: client,
	}
}

func (rq *RedisQueue) Enqueue(task *models.Task) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	taskData, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}

	pipe := rq.client.TxPipeline()

	taskKey := TaskDataKeyPrefix + task.ID
	pipe.Set(ctx, taskKey, taskData, 24*time.Hour)

	priority := float64(task.Priority)
	scheduledAtUnix := float64(task.ScheduledAt.Unix())

	score := priority*100000 + scheduledAtUnix
	pipe.ZAdd(ctx, TaskQueueKey, redis.Z{
		Score:  score,
		Member: task.ID,
	})

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to enqueue task: %w", err)
	}

	rq.publishTaskUpdate(ctx, task.ID, "created")

	log.Printf("Task %s enqueued with priority %d", task.ID, task.Priority)
	return nil
}

func (rq *RedisQueue) Dequeue(limit int) ([]*models.Task, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := rq.client.ZRangeByScore(ctx, TaskQueueKey, &redis.ZRangeBy{
		Min:   "-inf",
		Max:   "+inf",
		Count: int64(limit),
	}).Result()

	if err != nil {
		return nil, fmt.Errorf("failed to get tasks from queue:%w", err)
	}

	if len(result) == 0 {
		return []*models.Task{}, nil
	}

	var tasks []*models.Task

	for _, taskID := range result {
		lockKey := TaskLockKeyPrefix + taskID
		locked, err := rq.client.SetNX(ctx, lockKey, "locked", 5*time.Minute).Result()

		if err != nil || !locked {
			continue
		}

		taskKey := TaskDataKeyPrefix + taskID
		taskData, err := rq.client.Get(ctx, taskKey).Result()
		if err != nil {
			rq.client.Del(ctx, lockKey)
			continue
		}

		var task models.Task
		if err := json.Unmarshal([]byte(taskData), &task); err != nil {
			rq.client.Del(ctx, lockKey)
			continue
		}

		if task.ScheduledAt.After(time.Now()) {
			rq.client.Del(ctx, lockKey)
			continue
		}

		rq.client.ZRem(ctx, TaskQueueKey, taskID)

		tasks = append(tasks, &task)

		log.Printf("Task %s dequeueud by worker", taskID)
	}

	return tasks, nil
}

func (rq *RedisQueue) UpdateTask(task *models.Task) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	taskData, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}

	taskKey := TaskDataKeyPrefix + task.ID
	err = rq.client.Set(ctx, taskKey, taskData, 24*time.Hour).Err()

	if err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	if task.Status == models.TaskStatusPending && task.Retries > 0 {
		priority := float64(task.Priority)
		scheduledAtUnix := float64(task.ScheduledAt.Unix())
		score := priority*1000000 + scheduledAtUnix

		err = rq.client.ZAdd(ctx, TaskQueueKey, redis.Z{
			Score:  score,
			Member: task.ID,
		}).Err()

		if err != nil {
			return fmt.Errorf("failed to re-enqueue task:%w", err)
		}

		log.Printf("Task %s re-enqueued for retry %d%d", task.ID, task.Retries, task.MaxRetries)
	}

	if task.Status == models.TaskStatusCompleted || task.Status == models.TaskStatusFailed {
		lockKey := TaskLockKeyPrefix + task.ID
		rq.client.Del(ctx, lockKey)
	}

	rq.publishTaskUpdate(ctx, task.ID, string(task.Status))

	return nil
}

func (rq *RedisQueue) Size() (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	count, err := rq.client.ZCard(ctx, TaskQueueKey).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get queue size: %w", err)
	}

	return int(count), nil
}

func (rq *RedisQueue) publishTaskUpdate(ctx context.Context, taskID, status string) {
	message := map[string]interface{}{
		"task_id":   taskID,
		"timestamp": time.Now().Unix(),
	}

	messageJSON, err := json.Marshal(message)
	if err != nil {
		log.Printf("Failed to marshal task update message: %v", err)
		return
	}

	err = rq.client.Publish(ctx, TaskUpdateChannel, messageJSON).Err()

	if err != nil {
		log.Printf("Failed to publish task update %v", err)
	}
}

func (rq *RedisQueue) CleanupExpiredTasks(ctx context.Context) error {
	pattern := TaskDataKeyPrefix + "*"
	iter := rq.client.Scan(ctx, 0, pattern, 0).Iterator()

	for iter.Next(ctx) {
		key := iter.Val()
		ttl := rq.client.TTL(ctx, key).Val()

		if ttl < 0 {
			rq.client.Del(ctx, key)
		}
	}

	lockPattern := TaskLockKeyPrefix + "*"
	lockIter := rq.client.Scan(ctx, 0, lockPattern, 0).Iterator()

	for lockIter.Next(ctx) {
		key := lockIter.Val()
		ttl := rq.client.TTL(ctx, key).Val()
		if ttl < time.Minute {
			taskID := key[len(TaskLockKeyPrefix):]
			log.Printf("Cleaning up expired lock for task %s", taskID)
		}
	}

	return iter.Err()
}
