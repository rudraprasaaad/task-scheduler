package server

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/rudraprasaaad/task-scheduler/internal/auth"
	taskpb "github.com/rudraprasaaad/task-scheduler/internal/grpc/generated/task"
	"github.com/rudraprasaaad/task-scheduler/internal/grpc/interceptor"
	"github.com/rudraprasaaad/task-scheduler/internal/models"
	"github.com/rudraprasaaad/task-scheduler/internal/repository"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type TaskServer struct {
	taskpb.UnimplementedTaskServiceServer
	taskRepo   *repository.TaskRepository
	workerRepo *repository.WorkerRepository
}

func NewTaskServer(taskRepo *repository.TaskRepository, workerRepo *repository.WorkerRepository) *TaskServer {
	return &TaskServer{
		taskRepo:   taskRepo,
		workerRepo: workerRepo,
	}
}

func (s *TaskServer) CreateTask(ctx context.Context, req *taskpb.CreateTaskRequest) (*taskpb.CreateTaskResponse, error) {
	claims, ok := ctx.Value(interceptor.GrpcUserContextKey).(*auth.Claims)
	if !ok {
		return nil, status.Error(codes.Internal, "user claims not found in context")
	}

	log.Printf("CreateTask request received from UserID: %s", claims.UserID)

	if len(req.Tasks) == 0 {
		return nil, status.Error(codes.InvalidArgument, "No tasks provided in request")
	}

	createdTasks := make([]*taskpb.Task, 0, len(req.Tasks))

	for _, payload := range req.Tasks {
		var pld map[string]interface{}
		if err := json.Unmarshal([]byte(payload.PayloadJson), &pld); err != nil {
			log.Printf("Failed to unmarshal payload for task '%s': %v", payload.Name, err)
			continue
		}

		task := models.NewTask(payload.Name, payload.Type, pld)
		task.Priority = models.TaskPriority(payload.Priority)
		task.MaxRetries = int(payload.MaxRetries)
		if payload.ScheduledAt.IsValid() {
			task.ScheduledAt = payload.ScheduledAt.AsTime()
		}

		if err := s.taskRepo.Create(ctx, task); err != nil {
			log.Printf("Failed to create task '%s' in database: %v", task.Name, err)

			continue
		}

		pbTask, err := s.modelToProto(task)
		if err != nil {
			log.Printf("Failed to convert created task back to proto: %v", err)
			continue
		}
		createdTasks = append(createdTasks, pbTask)
	}

	return &taskpb.CreateTaskResponse{
		Tasks: createdTasks,
	}, nil
}

func (s *TaskServer) GetAvailableTask(ctx context.Context, req *taskpb.GetTaskRequest) (*taskpb.GetTaskResponse, error) {
	log.Printf("Worker %s requesting %d tasks", req.WorkerId, req.Limit)

	limit := int(req.Limit)

	if limit == 0 {
		limit = 10
	}

	tasks, err := s.taskRepo.GetReadyTasks(ctx, limit)
	if err != nil {
		log.Printf("Failed to get ready tasks: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to get tasks: %v", err)
	}

	pbTasks := make([]*taskpb.Task, 0, len(tasks))
	for _, task := range tasks {
		pbTask, err := s.modelToProto(task)
		if err != nil {
			log.Printf("failed to convert task %s: %v", task.ID, err)
			continue
		}
		pbTasks = append(pbTasks, pbTask)
	}

	return &taskpb.GetTaskResponse{
		Tasks: pbTasks,
	}, nil
}

func (s *TaskServer) UpdateTask(ctx context.Context, req *taskpb.UpdateTaskRequest) (*taskpb.UpdateTaskResponse, error) {
	if req.Task == nil {
		return &taskpb.UpdateTaskResponse{
			Success: false,
			Message: "task is required",
		}, nil
	}

	task, err := s.protoToModel(req.Task)

	if err != nil {
		return &taskpb.UpdateTaskResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	if err := s.taskRepo.Update(ctx, task); err != nil {
		log.Printf("Failed to update task %s: %v", task.ID, err)
		return &taskpb.UpdateTaskResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	log.Printf("Task %s updated successfully by worker %s", task.ID, task.WorkerID)

	return &taskpb.UpdateTaskResponse{
		Success: true,
		Message: "task updated sucessfully",
	}, nil
}

func (s *TaskServer) StreamTasks(req *taskpb.GetTaskRequest, stream taskpb.TaskService_StreamTasksServer) error {
	log.Printf("Starting task stream for worker %s", req.WorkerId)

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-stream.Context().Done():
			log.Printf("Task stream ended for worker %s", req.WorkerId)
			return nil
		case <-ticker.C:
			tasks, err := s.taskRepo.GetReadyTasks(stream.Context(), int(req.Limit))

			if err != nil {
				log.Printf("Failed to get ready tasks for stream: %v", err)
				continue
			}

			for _, task := range tasks {
				pbTask, err := s.modelToProto(task)
				if err != nil {
					log.Printf("Failed to convert task for stream: %v", err)
					continue
				}

				if err := stream.Send(pbTask); err != nil {
					log.Printf("Failed to send task via stream: %v", err)
					return err
				}
			}
		}
	}
}

func (s *TaskServer) modelToProto(task *models.Task) (*taskpb.Task, error) {
	payloadJSON, err := json.Marshal(task.Payload)
	if err != nil {
		return nil, err
	}

	pbTask := &taskpb.Task{
		Id:          task.ID,
		Name:        task.Name,
		Type:        task.Type,
		PayloadJson: string(payloadJSON),
		Priority:    int32(task.Priority),
		Status:      string(task.Status),
		Retries:     int32(task.Retries),
		MaxRetries:  int32(task.MaxRetries),
		CreatedAt:   timestamppb.New(task.CreatedAt),
		UpdatedAt:   timestamppb.New(task.UpdatedAt),
		ScheduledAt: timestamppb.New(task.ScheduledAt),
		Error:       task.Error,
		WorkerId:    task.WorkerID,
	}

	if task.StartedAt != nil {
		pbTask.StartedAt = timestamppb.New(*task.StartedAt)
	}

	if task.CompletedAt != nil {
		pbTask.CompletedAt = timestamppb.New(*task.CompletedAt)
	}

	return pbTask, nil
}

func (s *TaskServer) protoToModel(pbTask *taskpb.Task) (*models.Task, error) {
	var payload map[string]interface{}

	if err := json.Unmarshal([]byte(pbTask.PayloadJson), &payload); err != nil {
		return nil, err
	}

	task := &models.Task{
		ID:        pbTask.Id,
		Name:      pbTask.Name,
		Type:      pbTask.Type,
		Payload:   payload,
		Priority:  models.TaskPriority(pbTask.Priority),
		Status:    models.TaskStatus(pbTask.Status),
		Retries:   int(pbTask.Retries),
		CreatedAt: pbTask.CreatedAt.AsTime(),
		UpdatedAt: pbTask.UpdatedAt.AsTime(),
		Error:     pbTask.Error,
		WorkerID:  pbTask.WorkerId,
	}

	if pbTask.StartedAt != nil {
		startedAt := pbTask.StartedAt.AsTime()
		task.StartedAt = &startedAt
	}

	if pbTask.CompletedAt != nil {
		completedAt := pbTask.CompletedAt.AsTime()
		task.CompletedAt = &completedAt
	}

	return task, nil
}
