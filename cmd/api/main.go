package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/therealutkarshpriyadarshi/transcode/internal/config"
	"github.com/therealutkarshpriyadarshi/transcode/internal/database"
	"github.com/therealutkarshpriyadarshi/transcode/internal/middleware"
	"github.com/therealutkarshpriyadarshi/transcode/internal/queue"
	"github.com/therealutkarshpriyadarshi/transcode/internal/storage"
	"github.com/therealutkarshpriyadarshi/transcode/internal/transcoder"
	"github.com/therealutkarshpriyadarshi/transcode/pkg/models"
)

type API struct {
	repo    *database.Repository
	storage *storage.Storage
	queue   *queue.Queue
	ffmpeg  *transcoder.FFmpeg
}

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

	// Initialize JWT secret from config
	middleware.SetJWTSecret(cfg.Auth.JWTSecret)
	log.Printf("JWT authentication configured")

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

	// Initialize FFmpeg
	ffmpeg := transcoder.NewFFmpeg(cfg.Transcoder.FFmpegPath, cfg.Transcoder.FFprobePath)

	// Create API instance
	api := &API{
		repo:    repo,
		storage: stor,
		queue:   q,
		ffmpeg:  ffmpeg,
	}

	// Setup router
	router := setupRouter(api)

	// Create HTTP server
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Starting API server on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server stopped")
}

func setupRouter(api *API) *gin.Engine {
	router := gin.Default()

	// Health check
	router.GET("/health", api.healthCheck)

	// API routes
	v1 := router.Group("/api/v1")
	{
		// Videos
		v1.POST("/videos/upload", api.uploadVideo)
		v1.GET("/videos/:id", api.getVideo)
		v1.GET("/videos", api.listVideos)
		v1.DELETE("/videos/:id", api.deleteVideo)

		// Jobs
		v1.POST("/videos/:id/transcode", api.createTranscodeJob)
		v1.GET("/jobs/:id", api.getJob)
		v1.GET("/videos/:id/jobs", api.getVideoJobs)
		v1.POST("/jobs/:id/cancel", api.cancelJob)

		// Outputs
		v1.GET("/videos/:id/outputs", api.getVideoOutputs)
	}

	return router
}

// Health check endpoint
func (api *API) healthCheck(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// Check database health
	if err := api.repo.Health(ctx); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "unhealthy",
			"error":  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
	})
}

// Upload video endpoint
func (api *API) uploadVideo(c *gin.Context) {
	file, err := c.FormFile("video")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No video file provided"})
		return
	}

	// Save to temporary location
	tempPath := fmt.Sprintf("/tmp/%s", uuid.New().String())
	if err := c.SaveUploadedFile(file, tempPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
		return
	}
	defer os.Remove(tempPath)

	// Extract video metadata
	videoInfo, err := api.ffmpeg.ExtractVideoInfo(c.Request.Context(), tempPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to extract metadata: %v", err)})
		return
	}

	// Create video record
	video := &models.Video{
		ID:       uuid.New().String(),
		Filename: file.Filename,
		Size:     file.Size,
		Status:   models.VideoStatusPending,
	}

	// Copy metadata
	video.Duration = videoInfo.Duration
	video.Width = videoInfo.Width
	video.Height = videoInfo.Height
	video.Codec = videoInfo.Codec
	video.Bitrate = videoInfo.Bitrate
	video.FrameRate = videoInfo.FrameRate
	video.Metadata = videoInfo.Metadata

	// Upload to storage
	storageKey := fmt.Sprintf("videos/%s/original/%s", video.ID, file.Filename)
	if err := api.storage.UploadFile(c.Request.Context(), storageKey, tempPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to upload: %v", err)})
		return
	}

	video.OriginalURL = storageKey

	// Save to database
	if err := api.repo.CreateVideo(c.Request.Context(), video); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to create video: %v", err)})
		return
	}

	c.JSON(http.StatusCreated, video)
}

// Get video endpoint
func (api *API) getVideo(c *gin.Context) {
	videoID := c.Param("id")

	video, err := api.repo.GetVideo(c.Request.Context(), videoID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Video not found"})
		return
	}

	c.JSON(http.StatusOK, video)
}

// List videos endpoint
func (api *API) listVideos(c *gin.Context) {
	limit := 20
	offset := 0

	videos, err := api.repo.ListVideos(c.Request.Context(), limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"videos": videos,
		"limit":  limit,
		"offset": offset,
	})
}

// Delete video endpoint
func (api *API) deleteVideo(c *gin.Context) {
	videoID := c.Param("id")

	// Get video to ensure it exists and get the original URL
	video, err := api.repo.GetVideo(c.Request.Context(), videoID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Video not found"})
		return
	}

	// Get all outputs to delete their files from storage
	outputs, err := api.repo.GetOutputsByVideoID(c.Request.Context(), videoID)
	if err != nil {
		log.Printf("Warning: Failed to get outputs for video %s: %v", videoID, err)
		// Continue with deletion even if we can't get outputs
	}

	// Delete output files from storage
	for _, output := range outputs {
		if output.Path != "" {
			if err := api.storage.Delete(c.Request.Context(), output.Path); err != nil {
				log.Printf("Warning: Failed to delete output file %s: %v", output.Path, err)
				// Continue with deletion even if storage deletion fails
			}
		}
	}

	// Delete original video file from storage
	if video.OriginalURL != "" {
		// Extract object name from URL (assuming it's the last part)
		objectName := video.OriginalURL
		if err := api.storage.Delete(c.Request.Context(), objectName); err != nil {
			log.Printf("Warning: Failed to delete original video file %s: %v", objectName, err)
			// Continue with deletion even if storage deletion fails
		}
	}

	// Delete video and all associated records from database
	if err := api.repo.DeleteVideo(c.Request.Context(), videoID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to delete video: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Video deleted successfully", "video_id": videoID})
}

// Create transcode job endpoint
func (api *API) createTranscodeJob(c *gin.Context) {
	videoID := c.Param("id")

	var req struct {
		Resolution   string `json:"resolution" binding:"required"`
		OutputFormat string `json:"output_format"`
		Codec        string `json:"codec"`
		Bitrate      int64  `json:"bitrate"`
		Preset       string `json:"preset"`
		Priority     int    `json:"priority"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check video exists
	_, err := api.repo.GetVideo(c.Request.Context(), videoID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Video not found"})
		return
	}

	// Create job
	job := &models.Job{
		VideoID:  videoID,
		Status:   models.JobStatusQueued,
		Priority: req.Priority,
		Config: models.TranscodeConfig{
			OutputFormat: req.OutputFormat,
			Resolution:   req.Resolution,
			Codec:        req.Codec,
			Bitrate:      req.Bitrate,
			Preset:       req.Preset,
			AudioCodec:   "aac",
			AudioBitrate: 128,
		},
	}

	if job.Priority == 0 {
		job.Priority = models.JobPriorityNormal
	}

	// Save to database
	if err := api.repo.CreateJob(c.Request.Context(), job); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to create job: %v", err)})
		return
	}

	// Publish to queue
	if err := api.queue.PublishJob(c.Request.Context(), job); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to queue job: %v", err)})
		return
	}

	c.JSON(http.StatusCreated, job)
}

// Get job endpoint
func (api *API) getJob(c *gin.Context) {
	jobID := c.Param("id")

	job, err := api.repo.GetJob(c.Request.Context(), jobID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
		return
	}

	c.JSON(http.StatusOK, job)
}

// Get video jobs endpoint
func (api *API) getVideoJobs(c *gin.Context) {
	videoID := c.Param("id")

	jobs, err := api.repo.GetJobsByVideoID(c.Request.Context(), videoID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"jobs": jobs})
}

// Cancel job endpoint
func (api *API) cancelJob(c *gin.Context) {
	jobID := c.Param("id")

	// Cancel the job in the database
	if err := api.repo.CancelJob(c.Request.Context(), jobID); err != nil {
		if err.Error() == "job not found or cannot be cancelled" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to cancel job: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Job cancelled successfully", "job_id": jobID})
}

// Get video outputs endpoint
func (api *API) getVideoOutputs(c *gin.Context) {
	videoID := c.Param("id")

	outputs, err := api.repo.GetOutputsByVideoID(c.Request.Context(), videoID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"outputs": outputs})
}

// Add Health method to repository
func (r *database.Repository) Health(ctx context.Context) error {
	return r.db.Health(ctx)
}
