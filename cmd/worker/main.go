package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/therealutkarshpriyadarshi/transcode/internal/config"
	"github.com/therealutkarshpriyadarshi/transcode/internal/database"
	"github.com/therealutkarshpriyadarshi/transcode/internal/queue"
	"github.com/therealutkarshpriyadarshi/transcode/internal/storage"
	"github.com/therealutkarshpriyadarshi/transcode/internal/transcoder"
	"github.com/therealutkarshpriyadarshi/transcode/pkg/models"
)

func main() {
	// Load configuration
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "config.yaml"
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize database
	db, err := database.New(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	repo := database.NewRepository(db)

	// Initialize storage
	stor, err := storage.New(cfg.Storage)
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}

	// Initialize queue
	q, err := queue.New(cfg.Queue)
	if err != nil {
		log.Fatalf("Failed to connect to queue: %v", err)
	}
	defer q.Close()

	// Initialize transcoder service
	transcoderService := transcoder.NewService(cfg.Transcoder, stor, repo)

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown gracefully
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutting down worker gracefully...")
		cancel()
	}()

	// Job handler
	jobHandler := func(job *models.Job) error {
		log.Printf("Processing job %s for video %s", job.ID, job.VideoID)

		if err := transcoderService.ProcessJob(ctx, job); err != nil {
			log.Printf("Failed to process job %s: %v", job.ID, err)
			return err
		}

		log.Printf("Successfully processed job %s", job.ID)
		return nil
	}

	// Start consuming jobs
	log.Println("Worker started, waiting for jobs...")
	if err := q.ConsumeJobs(ctx, jobHandler); err != nil {
		log.Fatalf("Failed to consume jobs: %v", err)
	}

	// Wait for shutdown
	<-ctx.Done()
	log.Println("Worker stopped")
}
