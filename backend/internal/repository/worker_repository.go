package repository

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/rudraprasaaad/task-scheduler/internal/database"
	"github.com/rudraprasaaad/task-scheduler/internal/models"
)

type WorkerRepository struct {
	db *database.DB
}

func NewWorkerRepository(db *database.DB) *WorkerRepository {
	return &WorkerRepository{db: db}
}

func (r *WorkerRepository) Register(ctx context.Context, worker *models.Worker) error {
	query := `
    INSERT INTO workers (id, status, last_seen, tasks_run, created_at)
    VALUES ($1, $2, $3, $4, $5)
    ON CONFLICT (id) DO UPDATE SET
        status = EXCLUDED.status,
        last_seen = EXCLUDED.last_seen`

	_, err := r.db.ExecContext(ctx, query, worker.ID, worker.Status, worker.LastSeen, worker.TasksRun, time.Now())

	if err != nil {
		return fmt.Errorf("failed to register worker: %w", err)
	}

	return nil
}

func (r *WorkerRepository) UpdateStatus(ctx context.Context, workerID string, status models.WorkerStatus) error {
	query := `
    UPDATE workers 
    SET status = $2, last_seen = $3
    WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query, workerID, status, time.Now())

	if err != nil {
		return fmt.Errorf("failed to update worker status: %w", err)
	}

	return nil
}

func (r *WorkerRepository) IncrementTaskCount(ctx context.Context, workerID string) error {
	query := `UDPATE workers SET tasks_run = tasks_run + 1, last_seen = $2 WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query, workerID, time.Now())

	if err != nil {
		return fmt.Errorf("failed to increment task count: %w", err)
	}

	return nil
}

func (r *WorkerRepository) GetByID(ctx context.Context, id string) (*models.Worker, error) {
	query := `SELECT id, status, last_seen, tasks_run FROM workers WHERE id = $1`

	row := r.db.QueryRowContext(ctx, query, id)

	var worker models.Worker

	err := row.Scan(
		&worker.ID,
		&worker.Status,
		&worker.LastSeen,
		&worker.LastSeen,
		&worker.TasksRun,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("worker with ID %s not found", id)
		}

		return nil, fmt.Errorf("error scanning worker data %w", err)
	}

	return &worker, nil
}

func (r *WorkerRepository) GetAll(ctx context.Context) ([]*models.Worker, error) {
	query := `
    SELECT id, status, last_seen, tasks_run
    FROM workers
    ORDER BY created_at`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get workers: %w", err)
	}
	defer rows.Close()

	var workers []*models.Worker

	for rows.Next() {
		var w models.Worker

		err := rows.Scan(&w.ID, &w.Status, &w.LastSeen, &w.TasksRun)
		if err != nil {
			return nil, fmt.Errorf("failed to scan worker: %w", err)
		}

		workers = append(workers, &w)
	}

	return workers, nil
}

func (r *WorkerRepository) CleanupStaleWorkers(ctx context.Context, timeout time.Duration) error {
	query := `
    UPDATE workers 
    SET status = 'stopped'
    WHERE last_seen < $1 AND status != 'stopped'`

	cutoff := time.Now().Add(-timeout)

	result, err := r.db.ExecContext(ctx, query, cutoff)
	if err != nil {
		return fmt.Errorf("failed to cleanup stale workers: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected > 0 {
		log.Printf("Marked %d stale workers as stopped", rowsAffected)
	}

	return nil
}
