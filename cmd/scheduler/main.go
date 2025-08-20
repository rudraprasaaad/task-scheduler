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

	taskHandler := handlers.NewTaskHandler()

	router := mux.NewRouter()

	router.Use(middleware.LoggingMiddleware)

	api := router.PathPrefix("/api/v1").Subrouter()
	api.HandleFunc("/tasks", taskHandler.CreateTask).Methods("POST")
	api.HandleFunc("/tasks", taskHandler.ListTasks).Methods("GET")
	api.HandleFunc("/tasks/{id}", taskHandler.GetTask).Methods("GET")
	api.HandleFunc("/queue/status", taskHandler.GetQueueStatus).Methods("GET")

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
		log.Printf("Server starting on port %s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}
