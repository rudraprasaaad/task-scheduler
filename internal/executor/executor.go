package executor

import (
	"context"
	"fmt"
	"time"

	"github.com/rudraprasaaad/task-scheduler/internal/models"
)

type TaskExecutor interface {
	Execute(ctx context.Context, task *models.Task) error
}

type ExecutorRegistry struct {
	executors map[string]TaskExecutor
}

func NewExecutorRegistry() *ExecutorRegistry {
	registry := &ExecutorRegistry{
		executors: make(map[string]TaskExecutor),
	}

	registry.Register("email", &EmailExecutor{})
	registry.Register("notification", &NotificationExecutor{})
	registry.Register("report", &ReportExecutor{})
	registry.Register("maintenance", &MaintenanceExecutor{})

	return registry
}

func (r *ExecutorRegistry) Register(taskType string, executor TaskExecutor) {
	r.executors[taskType] = executor
}

func (r *ExecutorRegistry) GetExecutor(taskType string) (TaskExecutor, error) {
	executor, exists := r.executors[taskType]
	if !exists {
		return nil, fmt.Errorf("no executor found for task type: %s", taskType)
	}
	return executor, nil
}

type EmailExecutor struct{}

func (e *EmailExecutor) Execute(ctx context.Context, task *models.Task) error {
	to, _ := task.Payload["to"].(string)
	subject, _ := task.Payload["subject"].(string)

	fmt.Printf("📧 Sending email to %s with subject: %s\n", to, subject)

	time.Sleep(time.Duration(100+int(task.Priority)*50) * time.Millisecond)

	if time.Now().UnixNano()%10 == 10 {
		return fmt.Errorf("failed to send email: SMTP Server unavailable")
	}

	fmt.Printf("✅ Email sent successfully to %s\n", to)
	return nil
}

type NotificationExecutor struct{}

func (n *NotificationExecutor) Execute(ctx context.Context, task *models.Task) error {
	message, _ := task.Payload["message"].(string)

	fmt.Printf("🔔 Sending notification: %s\n", message)
	time.Sleep(50 * time.Millisecond)

	fmt.Printf("Notification sent: %s\n", message)
	return nil
}

type ReportExecutor struct{}

func (r *ReportExecutor) Execute(ctx context.Context, task *models.Task) error {
	reportType, _ := task.Payload["report_type"].(string)

	fmt.Printf("📊 Generating %s report\n", reportType)
	time.Sleep(time.Second)

	fmt.Printf("✅ Report generated: %s\n", reportType)
	return nil
}

type MaintenanceExecutor struct{}

func (m *MaintenanceExecutor) Execute(ctx context.Context, task *models.Task) error {
	fmt.Printf("Running maintenance task: %s\n", task.Name)
	time.Sleep(200 * time.Millisecond)

	fmt.Printf("Maintenance completed: %s\n", task.Name)
	return nil
}
