package cron

import (
	"log"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/rudraprasaaad/task-scheduler/internal/models"
	"github.com/rudraprasaaad/task-scheduler/internal/queue"
)

type Scheduler struct {
	scheduler gocron.Scheduler
	queue     *queue.RedisQueue
}

func NewScheduler(queue *queue.RedisQueue) (*Scheduler, error) {
	s, err := gocron.NewScheduler()
	if err != nil {
		return nil, err
	}

	return &Scheduler{
		scheduler: s,
		queue:     queue,
	}, nil
}

func (s *Scheduler) RegisterJobs() {
	_, err := s.scheduler.NewJob(
		gocron.CronJob("0 0 * * *", false),
		gocron.NewTask(s.enqueueDailyReportTask),
	)

	if err != nil {
		log.Printf("ERROR: Failed to register daily report cron job : %v", err)
	} else {
		log.Println("Successfully registered daily report cron job.")
	}
}

func (s *Scheduler) Start() {
	s.scheduler.Start()
	log.Println("Cron scheduler started.")
}

func (s *Scheduler) Stop() {
	if err := s.scheduler.Shutdown(); err != nil {
		log.Printf("ERROR: Failed to shutdown cron scheduler: %v", err)
	}

	log.Println("Cron scheduler stopped.")
}

func (s *Scheduler) enqueueDailyReportTask() {
	log.Println("CRON: Triggered 'enqueueDailyReportTask'.")
	payload := map[string]interface{}{
		"report_date": time.Now().UTC().Format("2006-01-02"),
		"send_to":     "",
	}

	task := models.NewTask(
		"Generate Daily Summary Report",
		"generate-daily-report",
		payload,
	)
	task.Priority = models.PriorityHigh

	if err := s.queue.Enqueue(task); err != nil {
		log.Printf("CRON ERROR: Failed to enqueue daily report task: %v", err)
	} else {
		log.Printf("CRON SUCCESS: Enqueued task %s for daily report generation", task.ID)
	}
}
