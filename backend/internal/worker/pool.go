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

type TaskStorage interface {
	UpdateTask(task *models.Task) error
	GetTask(id string) (*models.Task, error)
}

type Pool struct {
	workers  map[string]*Worker
	queue    *queue.PriorityQueue
	executor *executor.ExecutorRegistry
	storage  TaskStorage

	workerCount int
	stopChan    chan struct{}
	wg          sync.WaitGroup
	mutex       sync.RWMutex
}

func NewPool(workerCount int, queue *queue.PriorityQueue, storage TaskStorage) *Pool {
	return &Pool{
		workers:     make(map[string]*Worker),
		queue:       queue,
		executor:    executor.NewExecutorRegistry(),
		storage:     storage,
		workerCount: workerCount,
		stopChan:    make(chan struct{}),
	}
}

func (p *Pool) Start(ctx context.Context) {
	log.Printf("Starting worker pool with %d workers", p.workerCount)

	for i := 0; i < p.workerCount; i++ {
		workerID := fmt.Sprintf("worker-%d", i+1)
		worker := &Worker{
			ID:       workerID,
			Status:   WorkerStatusIdle,
			LastSeen: time.Now(),
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
	log.Println("Stopping worker pool....")
	close(p.stopChan)
	p.wg.Wait()
	log.Println("Worker pool stopped")
}

func (p *Pool) runWorker(ctx context.Context, worker *Worker) {
	defer p.wg.Done()

	log.Printf("Worker %s started", worker.ID)

	for {
		select {
		case <-ctx.Done():
			log.Printf("Worker %s stopping due to context cancellation", worker.ID)
			p.updateWorkerStatus(worker, WorkerStatusStopped)
			return
		case <-p.stopChan:
			log.Printf("Worker %s stopping", worker.ID)
			p.updateWorkerStatus(worker, WorkerStatusStopped)
			return
		default:
			task := p.queue.Dequeue()
			if task == nil {
				time.Sleep(100 * time.Millisecond)
				continue
			}

			p.processTask(ctx, worker, task)
		}
	}
}

func (p *Pool) processTask(ctx context.Context, worker *Worker, task *models.Task) {
	p.updateWorkerStatus(worker, WorkerStatusRunning)

	task.Status = models.TaskStatusRunning
	task.WorkerID = worker.ID
	now := time.Now()
	task.StartedAt = &now
	task.UpdatedAt = now

	if err := p.storage.UpdateTask(task); err != nil {
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
	p.updateWorkerStatus(worker, WorkerStatusIdle)
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
	task.UpdatedAt = now
	task.Error = ""

	if err := p.storage.UpdateTask(task); err != nil {
		log.Printf("Failed to update task after success : %v", err)
	}

	log.Printf("✅ Task %s completed successfully", task.ID)
}

func (p *Pool) handleTaskFailure(task *models.Task, execErr error) {
	task.Retries++
	task.Error = execErr.Error()
	task.UpdatedAt = time.Now()

	if task.Retries < task.MaxRetries {
		backoff := time.Duration(task.Retries*task.Retries) * time.Second
		task.ScheduledAt = time.Now().Add(backoff)
		task.Status = models.TaskStatusPending
		task.WorkerID = ""
		task.StartedAt = nil

		p.queue.Enqueue(task)

		log.Printf("⚠️ Task %s failed (attempt %d%d), retrying in %v: %v", task.ID, task.Retries, task.MaxRetries, backoff, execErr)
	} else {
		task.Status = models.TaskStatusFailed
		now := time.Now()
		task.CompletedAt = &now

		log.Printf("❌ Task %s failed permanently after %d attempts:L %v", task.ID, task.Retries, execErr)
	}

	if err := p.storage.UpdateTask(task); err != nil {
		log.Printf("Failed to update task after failure: %v", err)
	}
}

func (p *Pool) updateWorkerStatus(worker *Worker, status WorkerStatus) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	worker.Status = status
	worker.LastSeen = time.Now()
}

func (p *Pool) monitorWorkers(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-p.stopChan:
			return
		case <-ticker.C:
			p.logWorkerStats()
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
		case WorkerStatusIdle:
			idle++
		case WorkerStatusRunning:
			running++
		case WorkerStatusStopped:
			stopped++
		}
		totalTasks += worker.TasksRun
	}

	log.Printf("Worker Stats - Idle: %d, Running: %d, Stopped: %d, Total Tasks: %d, Queue Size: %d", idle, running, stopped, totalTasks, p.queue.Size())
}

func (p *Pool) GetWorkerStats() map[string]interface{} {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	stats := make(map[string]interface{})
	workers := make([]map[string]interface{}, 0, len(p.workers))

	for _, worker := range p.workers {
		workers = append(workers, map[string]interface{}{
			"id":        worker.ID,
			"status":    worker.Status,
			"last_seen": worker.LastSeen,
			"tasks_run": worker.TasksRun,
		})
	}

	stats["workers"] = workers
	stats["total_workers"] = len(p.workers)
	stats["queue_size"] = p.queue.Size()

	return stats
}
