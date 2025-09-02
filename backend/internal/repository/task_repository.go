package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/rudraprasaaad/task-scheduler/internal/database"
	"github.com/rudraprasaaad/task-scheduler/internal/models"
)

type TaskRepository struct {
	db *database.DB
}

func NewTaskRepository(db *database.DB) *TaskRepository {
	return &TaskRepository{db: db}
}

func (r *TaskRepository) Create(ctx context.Context, task *models.Task) error {
	payloadJSON, err := json.Marshal(task.Payload)

	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	query := `INSERT into tasks (id, name, type, payload, priority, status, retries, max_retries, created_at, updated_at, scheduled_at, error, worker_id) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`

	_, err = r.db.ExecContext(ctx, query, task.ID, task.Name, task.Type, payloadJSON, task.Priority, task.Status, task.Retries, task.MaxRetries, task.CreatedAt, task.UpdatedAt, task.ScheduledAt, task.Error, task.WorkerID)

	if err != nil {
		return fmt.Errorf("failed to create task: %w", err)
	}

	return nil
}

func (r *TaskRepository) GetByID(ctx context.Context, id string) (*models.Task, error) {
	query := `SELECT id, name, type, payload, priority, status, retries, max_retries, created_at, updated_at, scheduled_at, started_at, completed_at, error, worker_id FROM tasks WHERE id = $1`

	var task models.Task
	var payloadJSON []byte
	var startedAt, completedAt sql.NullTime
	var workerID, errorMsg sql.NullString

	err := r.db.QueryRowContext(ctx, query, id).Scan(&task.ID, &task.Name, &task.Type, &payloadJSON, &task.Priority, &task.Status, &task.Retries, &task.MaxRetries, &task.CreatedAt, &task.UpdatedAt, &task.ScheduledAt, &startedAt, &completedAt, &errorMsg, &workerID)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("task not found")
		}

		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	if err := json.Unmarshal(payloadJSON, &task.Payload); err != nil {
		return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	if startedAt.Valid {
		task.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		task.CompletedAt = &completedAt.Time
	}
	if errorMsg.Valid {
		task.Error = errorMsg.String
	}
	if workerID.Valid {
		task.WorkerID = workerID.String
	}

	return &task, nil
}

func (r *TaskRepository) Update(ctx context.Context, task *models.Task) error {
	payloadJSON, err := json.Marshal(task.Payload)

	if err != nil {
		return fmt.Errorf("failed to marshal payload : %w", err)
	}

	query := `UPDATE tasks SET name = $2, type = $2,  payload = $4, priority = $5, status = $6, retries = $7, max_retries = $8, updated_at = $9, scheduled_at = $10, started_at = $11, completed_at = $12, error = $13, worker_id = $14 WHERE id = $1`

	_, err = r.db.ExecContext(ctx, query, task.ID, task.Name, task.Type, payloadJSON, task.Priority, task.Status, task.Retries, task.MaxRetries, task.UpdatedAt, task.ScheduledAt, task.StartedAt, task.CompletedAt, task.Error, task.WorkerID)

	if err != nil {
		return fmt.Errorf("failed to update task :%w", err)
	}

	return nil
}

func (r *TaskRepository) List(ctx context.Context, limit, offset int) ([]*models.Task, error) {
	query := `SELECT id, name, type, payload, priority, status, retries, max_retries, created_at, updated_at, scheduled_at, started_at, completed_at, error, worker_id FROM tasks ORDER BY created_at DESC LIMIT $1 OFFSET $2`

	rows, err := r.db.QueryContext(ctx, query, limit, offset)

	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}

	defer rows.Close()

	var tasks []*models.Task

	for rows.Next() {
		var task models.Task
		var payloadJSON []byte
		var startedAt, completedAt sql.NullTime
		var workerID, errorMsg sql.NullString

		err := rows.Scan(
			&task.ID, &task.Name, &task.Type, &payloadJSON, &task.Priority, &task.Status, &task.Retries, &task.MaxRetries, &task.CreatedAt, &task.UpdatedAt, &task.ScheduledAt, &startedAt, &completedAt, &errorMsg, &workerID,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan task: %w", err)
		}

		if err := json.Unmarshal(payloadJSON, &task.Payload); err != nil {
			return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
		}

		if startedAt.Valid {
			task.StartedAt = &startedAt.Time
		}

		if completedAt.Valid {
			task.CompletedAt = &completedAt.Time
		}

		if errorMsg.Valid {
			task.Error = errorMsg.String
		}

		if workerID.Valid {
			task.WorkerID = workerID.String
		}

		tasks = append(tasks, &task)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return tasks, nil
}

func (r *TaskRepository) GetReadyTasks(ctx context.Context, limit int) ([]*models.Task, error) {
	query := `
    SELECT id, name, type, payload, priority, status, retries, max_retries,
           created_at, updated_at, scheduled_at, started_at, completed_at,
           error, worker_id
    FROM tasks 
    WHERE status = 'pending' AND scheduled_at <= NOW()
    ORDER BY priority DESC, scheduled_at ASC
    LIMIT $1`

	rows, err := r.db.QueryContext(ctx, query, limit)

	if err != nil {
		return nil, fmt.Errorf("failed to get ready tasks: %w", err)
	}

	defer rows.Close()

	var tasks []*models.Task

	for rows.Next() {
		var task models.Task
		var payloadJSON []byte
		var startedAt, completedAt sql.NullTime
		var workerID, errorMsg sql.NullString

		err := rows.Scan(
			&task.ID, &task.Name, &task.Type, &payloadJSON, &task.Priority, &task.Status, &task.Retries, &task.MaxRetries, &task.CreatedAt, &task.UpdatedAt, &task.ScheduledAt, &startedAt, &completedAt, &errorMsg, &workerID,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan ready taskL %w", err)
		}

		if err := json.Unmarshal(payloadJSON, &task.Payload); err != nil {
			return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
		}

		if startedAt.Valid {
			task.StartedAt = &startedAt.Time
		}

		if completedAt.Valid {
			task.CompletedAt = &completedAt.Time
		}

		if errorMsg.Valid {
			task.Error = errorMsg.String
		}

		if workerID.Valid {
			task.WorkerID = workerID.String
		}

		tasks = append(tasks, &task)
	}

	return tasks, nil
}

func (r *TaskRepository) GetStats(ctx context.Context) (map[string]int, error) {
	query := `
    SELECT status, COUNT(*) as count
    FROM tasks
    GROUP BY status`

	rows, err := r.db.QueryContext(ctx, query)

	if err != nil {
		return nil, fmt.Errorf("failed to get task stats: %w", err)
	}

	defer rows.Close()

	stats := make(map[string]int)

	for rows.Next() {
		var status string
		var count int

		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("failed to scan stats: %w", err)
		}

		stats[status] = count

	}

	return stats, nil
}

func (r *TaskRepository) Delete(ctx context.Context, id string) error {
	query := "DELETE FROM tasks WHERE id = $1"

	result, err := r.db.ExecContext(ctx, query, id)

	if err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
	}

	rowssAffected, err := result.RowsAffected()

	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowssAffected == 0 {
		return fmt.Errorf("task not found")
	}

	return nil
}

func (r *TaskRepository) GetStatus(ctx context.Context, taskID string) (models.TaskStatus, error) {
	var status models.TaskStatus
	query := "SELECT status FROM tasks WHERE id = $1"
	err := r.db.QueryRowContext(ctx, query, taskID).Scan(&status)
	return status, err
}

func (r *TaskRepository) UpdateStatus(ctx context.Context, taskID string, status models.TaskStatus) error {
	query := `UPDATE tasks SET tasks = $1, updated_at = NOW() WHERE id = $2`

	result, err := r.db.ExecContext(ctx, query, status, taskID)
	if err != nil {
		return fmt.Errorf("Failed to execute update task status query: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("Failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no task found with ID %s to update", taskID)
	}

	return nil
}
