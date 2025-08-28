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
	"github.com/rudraprasaaad/task-scheduler/internal/config"
	"github.com/rudraprasaaad/task-scheduler/internal/database"
	"github.com/rudraprasaaad/task-scheduler/internal/handlers"
	"github.com/rudraprasaaad/task-scheduler/internal/middleware"
	"github.com/rudraprasaaad/task-scheduler/internal/repository"

	taskpb "github.com/rudraprasaaad/task-scheduler/internal/grpc/generated/task"
	workerpb "github.com/rudraprasaaad/task-scheduler/internal/grpc/generated/worker"
	"github.com/rudraprasaaad/task-scheduler/internal/grpc/server"
	"google.golang.org/grpc"
)

func main() {
	cfg := config.Load()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db, err := database.New(&cfg.Database)

	if err != nil {
		log.Fatalf("Failed to connect in database: %v", err)
	}
	defer db.Close()

	migrator := database.NewMigrator(db)
	if err := migrator.RunMigrations("migrations"); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	taskRepo := repository.NewTaskRepository(db)
	workerRepo := repository.NewWorkerRepository(db)

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

	taskHandler := handlers.NewTaskHandler(taskRepo, workerRepo, cfg.MaxWorkers)

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
		if err := db.Health(); err != nil {
			http.Error(w, "Database unhealthy", http.StatusServiceUnavailable)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}).Methods("GET")

	router.HandleFunc("/db/stats", func(w http.ResponseWriter, r *http.Request) {
		stats := db.Stats()
		json.NewEncoder(w).Encode(map[string]interface{}{
			"open_connections":    stats.OpenConnections,
			"in_use":              stats.InUse,
			"idle":                stats.Idle,
			"wait_count":          stats.WaitCount,
			"wait_duration":       stats.WaitDuration,
			"max_idle_closed":     stats.MaxIdleClosed,
			"max_lifetime_closed": stats.MaxLifetimeClosed,
		})
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
