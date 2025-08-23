package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/rudraprasaaad/task-scheduler/internal/config"
	"github.com/rudraprasaaad/task-scheduler/internal/handlers"
	"github.com/rudraprasaaad/task-scheduler/internal/middleware"
)

func main() {
	cfg := config.Load()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	taskHandler := handlers.NewTaskHandler(cfg.MaxWorkers)

	taskHandler.StartWorkers(ctx)
	defer taskHandler.StopWorkers()

	router := mux.NewRouter()

	router.Use(middleware.LoggingMiddleware)

	api := router.PathPrefix("/api/v1").Subrouter()
	api.HandleFunc("/tasks", taskHandler.CreateTask).Methods("POST")
	api.HandleFunc("/tasks", taskHandler.ListTasks).Methods("GET")
	api.HandleFunc("/tasks/{id}", taskHandler.GetTaskByID).Methods("GET")
	api.HandleFunc("/queue/status", taskHandler.GetQueueStatus).Methods("GET")
	api.HandleFunc("/workers/stats", taskHandler.GetWorkerStats).Methods("GET")
	api.HandleFunc("/task/stats", taskHandler.GetTaskStats).Methods("GET")

	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
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
	cancel()

	shutdownCtx, shutDownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutDownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}
