package queue

import (
	"context"
	"sync"
	"time"

	"github.com/rudraprasaaad/task-scheduler/internal/models"
	"github.com/rudraprasaaad/task-scheduler/internal/repository"
)

type DatabaseQueue struct {
	taskRepo *repository.TaskRepository
	mutex    sync.RWMutex
}

func NewDatabaseQueue(taskRepo *repository.TaskRepository) *DatabaseQueue {
	return &DatabaseQueue{
		taskRepo: taskRepo,
	}
}

func (dq *DatabaseQueue) Enqueue(task *models.Task) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return dq.taskRepo.Create(ctx, task)
}

func (dq *DatabaseQueue) Dequeue(limit int) ([]*models.Task, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return dq.taskRepo.GetReadyTasks(ctx, limit)
}

func (dq *DatabaseQueue) Size() (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stats, err := dq.taskRepo.GetStats(ctx)
	if err != nil {
		return 0, err
	}

	return stats["pending"], nil
}

func (dq *DatabaseQueue) UpdateTask(task *models.Task) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	task.UpdatedAt = time.Now()
	return dq.taskRepo.Update(ctx, task)
}
