package repository

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/rudraprasaaad/task-scheduler/internal/database"
	"github.com/rudraprasaaad/task-scheduler/internal/worker"
)

type WorkerRepository struct {
	db *database.DB
}

func NewWorkerRepository(db *database.DB) *WorkerRepository {
	return &WorkerRepository{db: db}
}

func (r *WorkerRepository) Register(ctx context.Context, worker *worker.Worker) error {
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

func (r *WorkerRepository) UpdateStatus(ctx context.Context, workerID string, status worker.WorkerStatus) error {
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

func (r *WorkerRepository) GetAll(ctx context.Context) ([]*worker.Worker, error) {
	query := `
    SELECT id, status, last_seen, tasks_run
    FROM workers
    ORDER BY created_at`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get workers: %w", err)
	}
	defer rows.Close()

	var workers []*worker.Worker

	for rows.Next() {
		var w worker.Worker

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
