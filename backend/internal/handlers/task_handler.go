package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/rudraprasaaad/task-scheduler/internal/models"
	"github.com/rudraprasaaad/task-scheduler/internal/queue"
	"github.com/rudraprasaaad/task-scheduler/internal/worker"
)

type TaskHandler struct {
	queue *queue.PriorityQueue
	tasks map[string]*models.Task
	pool  *worker.Pool
	mutex sync.RWMutex
}

func NewTaskHandler(workerCount int) *TaskHandler {
	queue := queue.NewPriorityQueue()

	handler := &TaskHandler{
		queue: queue,
		tasks: make(map[string]*models.Task),
	}

	handler.pool = worker.NewPool(workerCount, queue, handler)

	return handler
}

func (h *TaskHandler) UpdateTask(task *models.Task) error {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	h.tasks[task.ID] = task
	return nil
}

func (h *TaskHandler) GetTask(id string) (*models.Task, error) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	task, exists := h.tasks[id]
	if !exists {
		return nil, fmt.Errorf("task not found")
	}

	return task, nil
}

func (h *TaskHandler) StartWorkers(ctx context.Context) {
	h.pool.Start(ctx)
}

func (h *TaskHandler) StopWorkers() {
	h.pool.Stop()
}

type CreateTaskRequest struct {
	Name       string                 `json:"name"`
	Type       string                 `json:"type"`
	Payload    map[string]interface{} `json:"payload"`
	Priority   models.TaskPriority    `json:"priority,omitempty"`
	ScheduleAt *time.Time             `json:"schedule_at,omitempty"`
	MaxRetries int                    `json:"max_retries,omitempty"`
}

func (h *TaskHandler) CreateTask(w http.ResponseWriter, r *http.Request) {
	var req CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Name == "" || req.Type == "" {
		http.Error(w, "Name and type are required", http.StatusBadRequest)
		return
	}

	task := models.NewTask(req.Name, req.Type, req.Payload)

	if req.Priority > 0 {
		task.Priority = req.Priority
	}

	if req.ScheduleAt != nil {
		task.ScheduledAt = *req.ScheduleAt
	}

	if req.MaxRetries > 0 {
		task.MaxRetries = req.MaxRetries
	}

	h.tasks[task.ID] = task

	h.queue.Enqueue(task)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(task)
}

func (h *TaskHandler) GetTaskByID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["id"]

	h.mutex.RLock()
	task, exists := h.tasks[taskID]
	h.mutex.RUnlock()

	if !exists {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

func (h *TaskHandler) ListTasks(w http.ResponseWriter, r *http.Request) {
	h.mutex.RLock()
	tasks := make([]*models.Task, 0, len(h.tasks))
	for _, task := range h.tasks {
		tasks = append(tasks, task)
	}
	h.mutex.RUnlock()

	w.Header().Set("Content-Type", "applicaton/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"tasks": tasks,
		"count": len(tasks),
	})
}

func (h *TaskHandler) GetQueueStatus(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"queue_size": h.queue.Size(),
		"timestamp":  time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func (h *TaskHandler) GetWorkerStats(w http.ResponseWriter, r *http.Request) {
	stats := h.pool.GetWorkerStats()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (h *TaskHandler) GetTaskStats(w http.ResponseWriter, r *http.Request) {
	h.mutex.RLock()
	var pending, running, completed, failed int

	for _, task := range h.tasks {
		switch task.Status {
		case models.TaskStatusPending:
			pending++
		case models.TaskStatusRunning:
			running++
		case models.TaskStatusCompleted:
			completed++
		case models.TaskStatusFailed:
			failed++
		}
	}
	h.mutex.RUnlock()

	stats := map[string]interface{}{
		"total_tasks": len(h.tasks),
		"pending":     pending,
		"running":     running,
		"completed":   completed,
		"failed":      failed,
		"queue_size":  h.queue.Size(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}
