package worker

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/rudraprasaaad/task-scheduler/internal/executor"
	"github.com/rudraprasaaad/task-scheduler/internal/models"
	"github.com/rudraprasaaad/task-scheduler/internal/queue"
	"github.com/rudraprasaaad/task-scheduler/internal/repository"
)

type Worker struct {
	ID       string
	Status   WorkerStatus
	LastSeen time.Time
	TasksRun int
}

type WorkerStatus string

const (
	WorkerStatusIdle    WorkerStatus = "idle"
	WorkerStatusRunning WorkerStatus = "running"
	WorkerStatusStopped WorkerStatus = "stopped"
)

type Pool struct {
	workers    map[string]*models.Worker
	queue      *queue.DatabaseQueue
	executor   *executor.ExecutorRegistry
	workerRepo *repository.WorkerRepository

	workerCount int
	stopChan    chan struct{}
	wg          sync.WaitGroup
	mutex       sync.RWMutex
}

func NewPool(workerCount int, queue *queue.DatabaseQueue, workerRepo *repository.WorkerRepository) *Pool {
	return &Pool{
		workers:     make(map[string]*models.Worker),
		queue:       queue,
		executor:    executor.NewExecutorRegistry(),
		workerRepo:  workerRepo,
		workerCount: workerCount,
		stopChan:    make(chan struct{}),
	}
}

func (p *Pool) Start(ctx context.Context) {
	log.Printf("Starting worker pool with %d workers", p.workerCount)

	for i := 0; i < p.workerCount; i++ {
		workerID := fmt.Sprintf("worker-%d", i+1)
		worker := &models.Worker{
			ID:       workerID,
			Status:   models.WorkerStatusIdle,
			LastSeen: time.Now(),
		}

		if err := p.registerWorker(ctx, worker); err != nil {
			log.Printf("Failed to register worker %s: %v", workerID, err)
			continue
		}

		p.mutex.Lock()
		p.workers[workerID] = worker
		p.mutex.Unlock()

		p.wg.Add(1)
		go p.runWorker(ctx, worker)
	}

	go p.monitorWorkers(ctx)
}

func (p *Pool) Stop() {
	log.Println("Stopping worker pool...")
	close(p.stopChan)
	p.wg.Wait()

	ctx := context.Background()
	p.mutex.RLock()
	for _, worker := range p.workers {
		p.workerRepo.UpdateStatus(ctx, worker.ID, models.WorkerStatusStopped)
	}
	p.mutex.RUnlock()

	log.Println("Worker pool stopped")
}

func (p *Pool) registerWorker(ctx context.Context, worker *models.Worker) error {
	return p.workerRepo.Register(ctx, worker)
}

func (p *Pool) runWorker(ctx context.Context, worker *models.Worker) {
	defer p.wg.Done()

	log.Printf("Worker %s started", worker.ID)

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("Worker %s stopping due to context cancellation", worker.ID)
			p.updateWorkerStatus(ctx, worker, models.WorkerStatusStopped)
			return
		case <-p.stopChan:
			log.Printf("Worker %s stopping", worker.ID)
			p.updateWorkerStatus(ctx, worker, models.WorkerStatusStopped)
			return
		case <-ticker.C:
			tasks, err := p.queue.Dequeue(1)
			if err != nil {
				log.Printf("Worker %s failed to dequeue tasks: %v", worker.ID, err)
				continue
			}

			if len(tasks) == 0 {
				continue
			}

			task := tasks[0]
			p.processTask(ctx, worker, task)
		}
	}
}

func (p *Pool) processTask(ctx context.Context, worker *models.Worker, task *models.Task) {
	p.updateWorkerStatus(ctx, worker, models.WorkerStatusRunning)

	task.Status = models.TaskStatusRunning
	task.WorkerID = worker.ID
	now := time.Now()
	task.StartedAt = &now

	if err := p.queue.UpdateTask(task); err != nil {
		log.Printf("Failed to update task status: %v", err)
	}

	log.Printf("Worker %s processing task %s (%s)", worker.ID, task.ID, task.Type)

	err := p.executeTask(ctx, task)

	if err != nil {
		p.handleTaskFailure(task, err)
	} else {
		p.handleTaskSuccess(task)
	}

	worker.TasksRun++
	p.workerRepo.IncrementTaskCount(ctx, worker.ID)
	p.updateWorkerStatus(ctx, worker, models.WorkerStatusIdle)
}

func (p *Pool) executeTask(ctx context.Context, task *models.Task) error {
	exec, err := p.executor.GetExecutor(task.Type)
	if err != nil {
		return fmt.Errorf("executor not found: %v", err)
	}

	taskCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	return exec.Execute(taskCtx, task)
}

func (p *Pool) handleTaskSuccess(task *models.Task) {
	task.Status = models.TaskStatusCompleted
	now := time.Now()
	task.CompletedAt = &now
	task.Error = ""

	if err := p.queue.UpdateTask(task); err != nil {
		log.Printf("Failed to update task after success: %v", err)
	}

	log.Printf("✅ Task %s completed successfully", task.ID)
}

func (p *Pool) handleTaskFailure(task *models.Task, execErr error) {
	task.Retries++
	task.Error = execErr.Error()

	if task.Retries < task.MaxRetries {
		backoff := time.Duration(task.Retries*task.Retries) * time.Second
		task.ScheduledAt = time.Now().Add(backoff)
		task.Status = models.TaskStatusPending
		task.WorkerID = ""
		task.StartedAt = nil

		log.Printf("⚠️ Task %s failed (attempt %d/%d), retrying in %v: %v",
			task.ID, task.Retries, task.MaxRetries, backoff, execErr)
	} else {
		task.Status = models.TaskStatusFailed
		now := time.Now()
		task.CompletedAt = &now

		log.Printf("❌ Task %s failed permanently after %d attempts: %v",
			task.ID, task.Retries, execErr)
	}

	if err := p.queue.UpdateTask(task); err != nil {
		log.Printf("Failed to update task after failure: %v", err)
	}
}

func (p *Pool) updateWorkerStatus(ctx context.Context, worker *models.Worker, status models.WorkerStatus) {
	p.mutex.Lock()
	worker.Status = status
	worker.LastSeen = time.Now()
	p.mutex.Unlock()

	if err := p.workerRepo.UpdateStatus(ctx, worker.ID, status); err != nil {
		log.Printf("Failed to update worker status in database: %v", err)
	}
}

func (p *Pool) monitorWorkers(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	cleanupTicker := time.NewTicker(1 * time.Minute)
	defer cleanupTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-p.stopChan:
			return
		case <-ticker.C:
			p.logWorkerStats()
		case <-cleanupTicker.C:
			if err := p.workerRepo.CleanupStaleWorkers(ctx, 2*time.Minute); err != nil {
				log.Printf("Failed to cleanup stale workers: %v", err)
			}
		}
	}
}

func (p *Pool) logWorkerStats() {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	var idle, running, stopped int
	totalTasks := 0

	for _, worker := range p.workers {
		switch worker.Status {
		case models.WorkerStatusIdle:
			idle++
		case models.WorkerStatusRunning:
			running++
		case models.WorkerStatusStopped:
			stopped++
		}
		totalTasks += worker.TasksRun
	}

	queueSize, _ := p.queue.Size()

	log.Printf("Worker Stats - Idle: %d, Running: %d, Stopped: %d, Total Tasks: %d, Queue Size: %d",
		idle, running, stopped, totalTasks, queueSize)
}

func (p *Pool) GetWorkerStats() (map[string]interface{}, error) {
	ctx := context.Background()

	dbWorkers, err := p.workerRepo.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	workers := make([]map[string]interface{}, 0, len(dbWorkers))
	for _, worker := range dbWorkers {
		workers = append(workers, map[string]interface{}{
			"id":        worker.ID,
			"status":    worker.Status,
			"last_seen": worker.LastSeen,
			"tasks_run": worker.TasksRun,
		})
	}

	queueSize, _ := p.queue.Size()

	return map[string]interface{}{
		"workers":       workers,
		"total_workers": len(dbWorkers),
		"queue_size":    queueSize,
	}, nil
}
