package main

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/rudraprasaaad/task-scheduler/internal/cache"
	"github.com/rudraprasaaad/task-scheduler/internal/config"
	"github.com/rudraprasaaad/task-scheduler/internal/database"
	"github.com/rudraprasaaad/task-scheduler/internal/handlers"
	"github.com/rudraprasaaad/task-scheduler/internal/middleware"
	"github.com/rudraprasaaad/task-scheduler/internal/redis"
	"github.com/rudraprasaaad/task-scheduler/internal/repository"

	taskpb "github.com/rudraprasaaad/task-scheduler/internal/grpc/generated/task"
	workerpb "github.com/rudraprasaaad/task-scheduler/internal/grpc/generated/worker"
	"github.com/rudraprasaaad/task-scheduler/internal/grpc/server"
	"google.golang.org/grpc"
)

func main() {
	cfg := config.Load()
	log.Printf("Starting application in %s environment", cfg.Environment)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if cfg.DatabaseURL == "" {
		log.Fatal("DATABASE_URL environment variable is not set")
	}

	db, err := database.New(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect in database: %v", err)
	}
	defer db.Close()
	log.Println("Database connection successful.")

	if cfg.RedisURL == "" {
		log.Fatalf("REDIS_URL environment variable is not set")
	}
	redisClient, err := redis.NewClient(cfg.RedisURL)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisClient.Close()

	migrator := database.NewMigrator(db)
	if err := migrator.RunMigrations("migrations"); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}
	log.Println("Database migrations completed successfully.")

	taskRepo := repository.NewTaskRepository(db)
	workerRepo := repository.NewWorkerRepository(db)
	cache := cache.NewRedisCache(redisClient, "task_scheduler:")

	grpcServer := grpc.NewServer()
	taskServer := server.NewTaskServer(taskRepo, workerRepo)
	workerServer := server.NewWorkerServer(workerRepo)
	taskpb.RegisterTaskServiceServer(grpcServer, taskServer)
	workerpb.RegisterWorkerServiceServer(grpcServer, workerServer)

	grpcListener, err := net.Listen("tcp", ":9090")
	if err != nil {
		log.Fatalf("Failed to listen for gRPC: %v", err)
	}

	go func() {
		log.Println("Starting gRPC server on :9090")
		if err := grpcServer.Serve(grpcListener); err != nil {
			log.Fatalf("Failed to server gRPC: %v", err)
		}
	}()

	taskHandler := handlers.NewTaskHandler(taskRepo, workerRepo, redisClient, cache, cfg.MaxWorkers)

	taskHandler.StartWorkers(ctx)
	defer taskHandler.StopWorkers()

	router := mux.NewRouter()

	router.Use(middleware.LoggingMiddleware)

	api := router.PathPrefix("/api/v1").Subrouter()
	api.HandleFunc("/tasks", taskHandler.CreateTask).Methods("POST")
	api.HandleFunc("/tasks", taskHandler.ListTasks).Methods("GET")
	api.HandleFunc("/tasks/{id}", taskHandler.GetTaskByID).Methods("GET")
	api.HandleFunc("/tasks/{id}", taskHandler.DeleteTask).Methods("DELETE")
	api.HandleFunc("/queue/status", taskHandler.GetQueueStatus).Methods("GET")
	api.HandleFunc("/workers/stats", taskHandler.GetWorkerStats).Methods("GET")
	api.HandleFunc("/tasks/stats", taskHandler.GetTaskStats).Methods("GET")

	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if err := db.Health(); err != nil {
			http.Error(w, `{"status": "error", "message": "database unhealthy"}`, http.StatusServiceUnavailable)
			return
		}

		if err := redisClient.Health(r.Context()); err != nil {
			http.Error(w, `{"status":"error", "message": "redis unhealthy"}`, http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}).Methods("GET")

	router.HandleFunc("/db/stats", func(w http.ResponseWriter, r *http.Request) {
		stats := db.Stats()
		json.NewEncoder(w).Encode(stats)
	}).Methods("GET")

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("Server starting on port %s with %d workers", cfg.Port, cfg.MaxWorkers)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	grpcServer.GracefulStop()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	cancel()

	log.Println("Server exited")
}
