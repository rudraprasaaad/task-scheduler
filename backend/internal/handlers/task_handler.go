package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/rudraprasaaad/task-scheduler/internal/models"
	"github.com/rudraprasaaad/task-scheduler/internal/queue"
	"github.com/rudraprasaaad/task-scheduler/internal/repository"
	"github.com/rudraprasaaad/task-scheduler/internal/worker"
)

type TaskHandler struct {
	taskRepo   *repository.TaskRepository
	workerRepo *repository.WorkerRepository
	queue      *queue.DatabaseQueue
	pool       *worker.Pool
}

func NewTaskHandler(taskRepo *repository.TaskRepository, workerRepo *repository.WorkerRepository, workerCount int) *TaskHandler {
	queue := queue.NewDatabaseQueue(taskRepo)

	handler := &TaskHandler{
		taskRepo:   taskRepo,
		workerRepo: workerRepo,
		queue:      queue,
	}

	handler.pool = worker.NewPool(workerCount, queue, workerRepo)

	return handler
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

	if err := h.queue.Enqueue(task); err != nil {
		http.Error(w, "Failed to create task", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(task)
}

func (h *TaskHandler) GetTaskByID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["id"]

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	task, err := h.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

func (h *TaskHandler) ListTasks(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := 50
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	offset := 0
	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tasks, err := h.taskRepo.List(ctx, limit, offset)
	if err != nil {
		http.Error(w, "Failed to retrieve tasks", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"tasks":  tasks,
		"count":  len(tasks),
		"limit":  limit,
		"offset": offset,
	})
}

func (h *TaskHandler) GetQueueStatus(w http.ResponseWriter, r *http.Request) {
	queueSize, err := h.queue.Size()
	if err != nil {
		http.Error(w, "Failed to get queue status", http.StatusInternalServerError)
		return
	}

	status := map[string]interface{}{
		"queue_size": queueSize,
		"timestamp":  time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func (h *TaskHandler) GetWorkerStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.pool.GetWorkerStats()
	if err != nil {
		http.Error(w, "Failed to get worker stats", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (h *TaskHandler) GetTaskStats(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stats, err := h.taskRepo.GetStats(ctx)
	if err != nil {
		http.Error(w, "Failed to get task stats", http.StatusInternalServerError)
		return
	}

	queueSize, _ := h.queue.Size()

	responseStats := make(map[string]interface{})
	total := 0

	for status, count := range stats {
		responseStats[status] = count
		total += count
	}

	responseStats["queue_size"] = queueSize
	responseStats["total_tasks"] = total

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (h *TaskHandler) DeleteTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["id"]

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := h.taskRepo.Delete(ctx, taskID); err != nil {
		http.Error(w, "Failed to delete task", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
