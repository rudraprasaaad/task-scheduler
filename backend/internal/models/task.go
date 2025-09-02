package models

import (
	"time"

	"github.com/google/uuid"
)

type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
	TaskStatusCancelled TaskStatus = "cancelled"
)

type TaskPriority int

const (
	PriorityLow    TaskPriority = 1
	PriorityMedium TaskPriority = 5
	PriorityHigh   TaskPriority = 10
)

type Task struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	Payload     map[string]interface{} `json:"payload"`
	Priority    TaskPriority           `json:"priority"`
	Status      TaskStatus             `json:"status"`
	Retries     int                    `json:"retries"`
	MaxRetries  int                    `json:"max_retries"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	ScheduledAt time.Time              `json:"scheduled_at"`
	StartedAt   *time.Time             `json:"started_at,omitempty"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	Error       string                 `json:"error,omitempty"`
	WorkerID    string                 `json:"worker_id,omitempty"`
}

func NewTask(name, taskType string, payload map[string]interface{}) *Task {
	now := time.Now()

	return &Task{
		ID:          generateID(),
		Name:        name,
		Type:        taskType,
		Payload:     payload,
		Priority:    PriorityMedium,
		Status:      TaskStatusPending,
		MaxRetries:  3,
		CreatedAt:   now,
		UpdatedAt:   now,
		ScheduledAt: now,
	}
}

func generateID() string {
	id := uuid.New()
	return id.String()
}
