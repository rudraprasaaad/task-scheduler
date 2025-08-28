package server

import (
	"context"
	"log"
	"sync"
	"time"

	workerpb "github.com/rudraprasaaad/task-scheduler/internal/grpc/generated/worker"
	"github.com/rudraprasaaad/task-scheduler/internal/models"
	"github.com/rudraprasaaad/task-scheduler/internal/repository"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type WorkerServer struct {
	workerpb.UnimplementedWorkerServiceServer
	workerRepo   *repository.WorkerRepository
	activeStream map[string]chan *workerpb.HeartbeatResponse
	streamsMutex sync.RWMutex
}

func NewWorkerServer(workerRepo *repository.WorkerRepository) *WorkerServer {
	return &WorkerServer{
		workerRepo:   workerRepo,
		activeStream: make(map[string]chan *workerpb.HeartbeatResponse),
	}
}

func (s *WorkerServer) RegisterWorker(ctx context.Context, req *workerpb.RegisterWorkerRequest) (*workerpb.RegisterWorkerResponse, error) {
	if req.Worker == nil {
		return &workerpb.RegisterWorkerResponse{
			Success: false,
			Message: "Worker information is required",
		}, nil
	}

	worker := s.protoToModel(req.Worker)
	worker.Status = models.WorkerStatusIdle
	worker.LastSeen = time.Now()

	if err := s.workerRepo.Register(ctx, worker); err != nil {
		log.Printf("Failed to register worker %s: %v", worker.ID, err)

		return &workerpb.RegisterWorkerResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	log.Printf("Worker %s registered successfully", worker.ID)

	return &workerpb.RegisterWorkerResponse{
		Success:    true,
		Message:    "worker registered successfully",
		AssignedId: worker.ID,
	}, nil
}

func (s *WorkerServer) Heartbeat(ctx context.Context, req *workerpb.HeartbeatRequest) (*workerpb.HeartbeatResponse, error) {
	if req.WorkerId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "worker_id is required")
	}

	if err := s.workerRepo.UpdateStatus(ctx, req.WorkerId, models.WorkerStatus(req.Status)); err != nil {
		log.Printf("Failed to update worker %s status: %v", req.WorkerId, err)

		return &workerpb.HeartbeatResponse{
			Acknowledged: false,
			Instructions: map[string]string{
				"error": err.Error(),
			},
		}, nil
	}

	response := &workerpb.HeartbeatResponse{
		Acknowledged: true,
		Instructions: map[string]string{
			"status": "healthty",
		},
	}

	s.streamsMutex.RLock()
	if stream, exists := s.activeStream[req.WorkerId]; exists {
		select {
		case stream <- response:
		default:
		}
	}
	s.streamsMutex.RUnlock()

	return response, nil
}

func (s *WorkerServer) HealthCheck(ctx context.Context, req *workerpb.HealthRequest) (*workerpb.HealthResponse, error) {
	return &workerpb.HealthResponse{
		Status:    "healthy",
		Timestamp: timestamppb.New(time.Now()),
		Details: map[string]string{
			"service": "worker-service",
			"version": "1.0.0",
		},
	}, nil
}

func (s *WorkerServer) GetWorkers(ctx context.Context, req *workerpb.GetWorkersRequest) (*workerpb.GetWorkersResponse, error) {
	workers, err := s.workerRepo.GetAll(ctx)

	if err != nil {
		log.Printf("Failed to get workers: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to get workers: %v", err)
	}

	pbWorkers := make([]*workerpb.Worker, 0, len(workers))
	for _, worker := range workers {
		pbWorkers = append(pbWorkers, s.modelToProto(worker))
	}

	return &workerpb.GetWorkersResponse{
		Workers: pbWorkers,
	}, nil
}

func (s *WorkerServer) StreamHeartbeats(req *workerpb.HeartbeatRequest, stream workerpb.WorkerService_StreamHeartbeatsServer) error {
	workerID := req.WorkerId
	log.Printf("Starting heartbeat stream for worker %s", workerID)

	respChan := make(chan *workerpb.HeartbeatResponse, 10)

	s.streamsMutex.Lock()
	s.activeStream[workerID] = respChan
	s.streamsMutex.Unlock()

	defer func() {
		s.streamsMutex.Lock()
		delete(s.activeStream, workerID)
		close(respChan)
		s.streamsMutex.Unlock()
		log.Printf("Heartbeat stream ended for worker %s", workerID)
	}()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-stream.Context().Done():
			return nil
		case response := <-respChan:
			if err := stream.Send(response); err != nil {
				log.Printf("Failed to send heartbeat response: %v", err)

				return err
			}
		case <-ticker.C:
			response := &workerpb.HeartbeatResponse{
				Acknowledged: true,
				Instructions: map[string]string{
					"message": "keep-alive",
				},
			}

			if err := stream.Send(response); err != nil {
				log.Printf("Failed to send keep-alive: %v", err)
				return err
			}
		}
	}
}

func (s *WorkerServer) protoToModel(pbWorker *workerpb.Worker) *models.Worker {
	return &models.Worker{
		ID:       pbWorker.Id,
		Status:   models.WorkerStatus(pbWorker.Status),
		LastSeen: pbWorker.LastSeen.AsTime(),
		TasksRun: int(pbWorker.TasksRun),
	}
}

func (s *WorkerServer) modelToProto(worker *models.Worker) *workerpb.Worker {
	return &workerpb.Worker{
		Id:       worker.ID,
		Status:   string(worker.Status),
		LastSeen: timestamppb.New(worker.LastSeen),
		TasksRun: int32(worker.TasksRun),
	}
}
