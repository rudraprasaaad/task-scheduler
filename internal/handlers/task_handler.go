package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/rudraprasaaad/task-scheduler/internal/models"
	"github.com/rudraprasaaad/task-scheduler/internal/queue"
)

type TaskHandler struct {
	queue *queue.PriorityQueue
	tasks map[string]*models.Task
}

func NewTaskHandler() *TaskHandler {
	return &TaskHandler{
		queue: queue.NewPriorityQueue(),
		tasks: make(map[string]*models.Task),
	}
}

type CreateTaskRequest struct {
	Name       string                 `json:"name"`
	Type       string                 `json:"type"`
	Payload    map[string]interface{} `json:"payload"`
	Priority   models.TaskPriority    `json:"priority,omitempty"`
	ScheduleAt *time.Time             `json:"schedule_at,omitempty"`
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

	h.tasks[task.ID] = task

	h.queue.Enqueue(task)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(task)
}

func (h *TaskHandler) GetTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["id"]

	task, exists := h.tasks[taskID]

	if !exists {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Typ", "application/json")
	json.NewEncoder(w).Encode(task)
}

func (h *TaskHandler) ListTasks(w http.ResponseWriter, r *http.Request) {
	tasks := make([]*models.Task, 0, len(h.tasks))
	for _, task := range h.tasks {
		tasks = append(tasks, task)
	}

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
