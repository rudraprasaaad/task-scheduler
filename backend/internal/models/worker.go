package models

import "time"

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
