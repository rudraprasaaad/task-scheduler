package repository

import (
	"context"
	"fmt"

	"github.com/rudraprasaaad/task-scheduler/internal/database"
	"github.com/rudraprasaaad/task-scheduler/internal/models"
)

type TaskExecutionRepository struct {
	db *database.DB
}

func NewTaskExecutionRepository(db *database.DB) *TaskExecutionRepository {
	return &TaskExecutionRepository{db: db}
}

func (r *TaskExecutionRepository) Create(ctx context.Context, execution *models.TaskExecution) error {
	query := `
        INSERT INTO task_executions (
            id, 
            task_id, 
            worker_id, 
            started_at, 
            completed_at, 
            status, 
            error, 
            execution_time_ms
        )
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
    `

	_, err := r.db.ExecContext(
		ctx,
		query,
		execution.ID,
		execution.TaskID,
		execution.WorkerID,
		execution.StartedAt,
		execution.CompletedAt,
		execution.Status,
		execution.Error,
		execution.ExecutionTimeMs,
	)

	if err != nil {
		return fmt.Errorf("failed to create task execution record in database: %w", err)
	}

	return nil
}
