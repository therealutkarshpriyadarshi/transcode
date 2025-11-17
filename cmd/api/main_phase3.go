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
	"github.com/therealutkarshpriyadarshi/transcode/internal/monitoring"
	"github.com/therealutkarshpriyadarshi/transcode/internal/queue"
	"github.com/therealutkarshpriyadarshi/transcode/internal/scheduler"
	"github.com/therealutkarshpriyadarshi/transcode/internal/storage"
	"github.com/therealutkarshpriyadarshi/transcode/internal/transcoder"
	"github.com/therealutkarshpriyadarshi/transcode/internal/upload"
	"github.com/therealutkarshpriyadarshi/transcode/internal/webhook"
	"github.com/therealutkarshpriyadarshi/transcode/pkg/models"
)

type API struct {
	repo           *database.Repository
	storage        *storage.Storage
	queue          *queue.Queue
	ffmpeg         *transcoder.FFmpeg
	uploadService  *upload.MultipartUploadService
	webhookService *webhook.Service
	scheduler      *scheduler.JobScheduler
	monitor        *monitoring.Monitor
	rateLimiter    *middleware.RateLimiter
}

func mainPhase3() {
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

	// Setup dead letter queue
	if err := q.SetupDeadLetterQueue(); err != nil {
		log.Printf("Warning: Failed to setup DLQ: %v", err)
	}

	// Initialize FFmpeg
	ffmpeg := transcoder.NewFFmpeg(cfg.Transcoder.FFmpegPath, cfg.Transcoder.FFprobePath)

	// Initialize multipart upload service
	uploadService := upload.NewMultipartUploadService(cfg.Transcoder.TempDir, cfg.Transcoder.ChunkSize)

	// Start cleanup goroutine for expired uploads
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go uploadService.CleanupExpired(ctx)

	// Initialize webhook service
	webhookService := webhook.NewService(repo)

	// Start webhook retry worker
	go webhookService.RetryWorker(ctx)

	// Initialize job scheduler
	jobScheduler := scheduler.NewScheduler(repo, q, cfg.Transcoder.MaxConcurrent)
	if err := jobScheduler.Start(); err != nil {
		log.Fatalf("Failed to start scheduler: %v", err)
	}
	defer jobScheduler.Stop()

	// Initialize monitoring
	monitor := monitoring.NewMonitor(repo, q)
	monitor.Start(ctx)

	// Initialize rate limiter (10 requests per second, burst of 20)
	rateLimiter := middleware.NewRateLimiter(10, 20)
	go rateLimiter.Cleanup()

	// Create API instance
	api := &API{
		repo:           repo,
		storage:        stor,
		queue:          q,
		ffmpeg:         ffmpeg,
		uploadService:  uploadService,
		webhookService: webhookService,
		scheduler:      jobScheduler,
		monitor:        monitor,
		rateLimiter:    rateLimiter,
	}

	// Setup router
	router := setupRouterPhase3(api)

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

	// Cancel context for background workers
	cancel()

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server stopped")
}

func setupRouterPhase3(api *API) *gin.Engine {
	router := gin.Default()

	// Apply global middleware
	router.Use(middleware.CORS())
	router.Use(middleware.Logger())
	router.Use(middleware.RateLimit(api.rateLimiter))

	// Health check
	router.GET("/health", api.healthCheck)

	// Public routes
	public := router.Group("/api/v1")
	{
		// Auth
		public.POST("/auth/register", api.register)
		public.POST("/auth/login", api.login)
	}

	// Protected routes (require authentication)
	protected := router.Group("/api/v1")
	protected.Use(middleware.OptionalAuth(api.repo))
	{
		// Videos
		protected.POST("/videos/upload", middleware.QuotaLimit(api.repo), api.uploadVideo)
		protected.GET("/videos/:id", api.getVideo)
		protected.GET("/videos", api.listVideos)
		protected.DELETE("/videos/:id", api.deleteVideo)

		// Multipart uploads
		protected.POST("/uploads/initiate", api.initiateUpload)
		protected.PUT("/uploads/:upload_id/parts/:part_number", api.uploadPart)
		protected.POST("/uploads/:upload_id/complete", middleware.QuotaLimit(api.repo), api.completeUpload)
		protected.DELETE("/uploads/:upload_id", api.abortUpload)
		protected.GET("/uploads/:upload_id", api.getUploadStatus)

		// Jobs
		protected.POST("/videos/:id/transcode", api.createTranscodeJob)
		protected.GET("/jobs/:id", api.getJob)
		protected.GET("/videos/:id/jobs", api.getVideoJobs)
		protected.POST("/jobs/:id/cancel", api.cancelJob)
		protected.POST("/jobs/:id/pause", api.pauseJob)
		protected.POST("/jobs/:id/resume", api.resumeJob)

		// Outputs
		protected.GET("/videos/:id/outputs", api.getVideoOutputs)

		// Webhooks
		protected.POST("/webhooks", api.createWebhook)
		protected.GET("/webhooks", api.listWebhooks)

		// Monitoring
		protected.GET("/metrics", api.getMetrics)
		protected.GET("/workers/health", api.getWorkerHealth)
		protected.GET("/system/health", api.getSystemHealth)
		protected.GET("/queue/stats", api.getQueueStats)
	}

	return router
}

// Enhanced upload with webhook notification
func (api *API) uploadVideoPhase3(c *gin.Context) {
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

	// Get user ID from context
	if userID, exists := middleware.GetUserID(c); exists {
		video.UserID = &userID
	}

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

	// Send webhook notification
	if api.webhookService != nil {
		_ = api.webhookService.NotifyVideoUploaded(c.Request.Context(), video)
	}

	c.JSON(http.StatusCreated, video)
}

// Enhanced create job with scheduler
func (api *API) createTranscodeJobPhase3(c *gin.Context) {
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
	video, err := api.repo.GetVideo(c.Request.Context(), videoID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Video not found"})
		return
	}

	// Check user ownership if authenticated
	if userID, exists := middleware.GetUserID(c); exists {
		if video.UserID == nil || *video.UserID != userID {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}
	}

	// Create job
	job := &models.Job{
		VideoID:  videoID,
		Status:   models.JobStatusPending,
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

	// Get user ID from context
	if userID, exists := middleware.GetUserID(c); exists {
		job.UserID = &userID
	}

	// Save to database
	if err := api.repo.CreateJob(c.Request.Context(), job); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to create job: %v", err)})
		return
	}

	// Schedule job using scheduler
	if api.scheduler != nil {
		if err := api.scheduler.ScheduleJob(job); err != nil {
			log.Printf("Failed to schedule job: %v", err)
			// Fallback to direct queue publish
			if err := api.queue.PublishJob(c.Request.Context(), job); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to queue job: %v", err)})
				return
			}
		}
	} else {
		// Fallback to direct queue publish
		if err := api.queue.PublishJob(c.Request.Context(), job); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to queue job: %v", err)})
			return
		}
	}

	c.JSON(http.StatusCreated, job)
}

// Enhanced cancel job
func (api *API) cancelJobPhase3(c *gin.Context) {
	jobID := c.Param("id")

	// Get job to check ownership
	job, err := api.repo.GetJob(c.Request.Context(), jobID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
		return
	}

	// Check user ownership if authenticated
	if userID, exists := middleware.GetUserID(c); exists {
		if job.UserID == nil || *job.UserID != userID {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}
	}

	if err := api.repo.CancelJob(c.Request.Context(), jobID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Notify scheduler
	if api.scheduler != nil {
		api.scheduler.JobCompleted(jobID)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Job cancelled"})
}

// Enhanced delete video
func (api *API) deleteVideoPhase3(c *gin.Context) {
	videoID := c.Param("id")

	// Get video to check ownership
	video, err := api.repo.GetVideo(c.Request.Context(), videoID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Video not found"})
		return
	}

	// Check user ownership if authenticated
	if userID, exists := middleware.GetUserID(c); exists {
		if video.UserID == nil || *video.UserID != userID {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}
	}

	// Delete from storage
	if err := api.storage.DeleteFile(c.Request.Context(), video.OriginalURL); err != nil {
		log.Printf("Failed to delete file from storage: %v", err)
	}

	// Delete from database (cascade will delete jobs and outputs)
	if err := api.repo.DeleteVideo(c.Request.Context(), videoID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete video"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Video deleted"})
}
