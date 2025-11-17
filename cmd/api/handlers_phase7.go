package main

import (
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/therealutkarshpriyadarshi/transcode/internal/analytics"
	"github.com/therealutkarshpriyadarshi/transcode/internal/transcoder"
	"github.com/therealutkarshpriyadarshi/transcode/pkg/models"
)

// Phase 7 Handlers

// Scene Detection Handlers

type sceneDetectionRequest struct {
	Threshold        float64 `json:"threshold"`
	MinSceneDuration float64 `json:"min_scene_duration"`
	MaxScenes        int     `json:"max_scenes"`
}

func (s *Server) handleSceneDetection(c *gin.Context) {
	videoID := c.Param("id")

	var req sceneDetectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get video
	video, err := s.repo.GetVideo(c.Request.Context(), videoID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "video not found"})
		return
	}

	// Create temp directory for scene frames
	tempDir := filepath.Join(s.cfg.Transcoder.TempDir, "scenes", videoID)

	// Download video from storage
	inputPath := filepath.Join(tempDir, "input"+filepath.Ext(video.Filename))
	if err := s.storage.DownloadFile(c.Request.Context(), video.OriginalURL, inputPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to download video"})
		return
	}

	// Perform scene detection
	ffmpeg := transcoder.NewFFmpeg(s.cfg.Transcoder.FFmpegPath, s.cfg.Transcoder.FFprobePath)
	opts := transcoder.SceneDetectionOptions{
		InputPath:        inputPath,
		OutputDir:        tempDir,
		Threshold:        req.Threshold,
		MinSceneDuration: req.MinSceneDuration,
		MaxScenes:        req.MaxScenes,
	}

	result, err := ffmpeg.DetectScenes(c.Request.Context(), opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "scene detection failed", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// Watermark Handlers

type watermarkRequest struct {
	WatermarkText  string  `json:"watermark_text,omitempty"`
	WatermarkImage string  `json:"watermark_image,omitempty"` // URL or path
	Position       string  `json:"position"`
	Opacity        float64 `json:"opacity"`
	Scale          float64 `json:"scale"`
	FontSize       int     `json:"font_size"`
	FontColor      string  `json:"font_color"`
	Padding        int     `json:"padding"`
	OutputFormat   string  `json:"output_format"`
}

func (s *Server) handleApplyWatermark(c *gin.Context) {
	videoID := c.Param("id")

	var req watermarkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get video
	video, err := s.repo.GetVideo(c.Request.Context(), videoID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "video not found"})
		return
	}

	// Create temp directory
	tempDir := filepath.Join(s.cfg.Transcoder.TempDir, "watermark", uuid.New().String())

	// Download video
	inputPath := filepath.Join(tempDir, "input"+filepath.Ext(video.Filename))
	if err := s.storage.DownloadFile(c.Request.Context(), video.OriginalURL, inputPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to download video"})
		return
	}

	// Prepare output
	outputFormat := req.OutputFormat
	if outputFormat == "" {
		outputFormat = "mp4"
	}
	outputPath := filepath.Join(tempDir, "watermarked."+outputFormat)

	// Apply watermark
	ffmpeg := transcoder.NewFFmpeg(s.cfg.Transcoder.FFmpegPath, s.cfg.Transcoder.FFprobePath)
	opts := transcoder.WatermarkOptions{
		InputPath:     inputPath,
		OutputPath:    outputPath,
		WatermarkText: req.WatermarkText,
		Position:      req.Position,
		Opacity:       req.Opacity,
		Scale:         req.Scale,
		FontSize:      req.FontSize,
		FontColor:     req.FontColor,
		Padding:       req.Padding,
	}

	if req.WatermarkImage != "" {
		// Download watermark image if URL
		watermarkPath := filepath.Join(tempDir, "watermark.png")
		if err := s.storage.DownloadFile(c.Request.Context(), req.WatermarkImage, watermarkPath); err == nil {
			opts.WatermarkPath = watermarkPath
		} else {
			opts.WatermarkPath = req.WatermarkImage // Assume it's a local path
		}
	}

	if err := ffmpeg.ApplyWatermark(c.Request.Context(), opts); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "watermarking failed", "details": err.Error()})
		return
	}

	// Upload watermarked video
	storageKey := filepath.Join("videos", videoID, "watermarked", filepath.Base(outputPath))
	if err := s.storage.UploadFile(c.Request.Context(), storageKey, outputPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to upload watermarked video"})
		return
	}

	url, _ := s.storage.GetURL(c.Request.Context(), storageKey)

	c.JSON(http.StatusOK, gin.H{
		"message": "watermark applied successfully",
		"url":     url,
		"path":    storageKey,
	})
}

// Concatenation Handlers

type concatenationRequest struct {
	VideoIDs           []string `json:"video_ids" binding:"required"`
	Method             string   `json:"method"`
	TransitionType     string   `json:"transition_type"`
	TransitionDuration float64  `json:"transition_duration"`
	ReEncode           bool     `json:"re_encode"`
}

func (s *Server) handleConcatenateVideos(c *gin.Context) {
	var req concatenationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(req.VideoIDs) < 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "at least 2 videos required"})
		return
	}

	// Create temp directory
	tempDir := filepath.Join(s.cfg.Transcoder.TempDir, "concat", uuid.New().String())

	// Download all videos
	inputPaths := make([]string, 0, len(req.VideoIDs))
	for i, videoID := range req.VideoIDs {
		video, err := s.repo.GetVideo(c.Request.Context(), videoID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "video not found", "video_id": videoID})
			return
		}

		inputPath := filepath.Join(tempDir, filepath.Sprintf("input_%d%s", i, filepath.Ext(video.Filename)))
		if err := s.storage.DownloadFile(c.Request.Context(), video.OriginalURL, inputPath); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to download video", "video_id": videoID})
			return
		}

		inputPaths = append(inputPaths, inputPath)
	}

	// Concatenate videos
	outputPath := filepath.Join(tempDir, "concatenated.mp4")
	ffmpeg := transcoder.NewFFmpeg(s.cfg.Transcoder.FFmpegPath, s.cfg.Transcoder.FFprobePath)
	opts := transcoder.ConcatenationOptions{
		InputPaths:         inputPaths,
		OutputPath:         outputPath,
		Method:             req.Method,
		TransitionType:     req.TransitionType,
		TransitionDuration: req.TransitionDuration,
		ReEncode:           req.ReEncode,
	}

	if err := ffmpeg.ConcatVideo(c.Request.Context(), opts); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "concatenation failed", "details": err.Error()})
		return
	}

	// Upload concatenated video
	storageKey := filepath.Join("videos", "concatenated", uuid.New().String()+".mp4")
	if err := s.storage.UploadFile(c.Request.Context(), storageKey, outputPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to upload concatenated video"})
		return
	}

	url, _ := s.storage.GetURL(c.Request.Context(), storageKey)

	c.JSON(http.StatusOK, gin.H{
		"message": "videos concatenated successfully",
		"url":     url,
		"path":    storageKey,
	})
}

// Analytics Handlers

func (s *Server) handleTrackPlaybackEvent(c *gin.Context) {
	var event models.PlaybackEvent
	if err := c.ShouldBindJSON(&event); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	analyticsService := analytics.NewService(s.repo)
	if err := analyticsService.TrackEvent(c.Request.Context(), &event); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to track event"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "event tracked successfully"})
}

func (s *Server) handleStartPlaybackSession(c *gin.Context) {
	videoID := c.Param("id")

	var req struct {
		UserID     string            `json:"user_id"`
		DeviceInfo map[string]string `json:"device_info"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	analyticsService := analytics.NewService(s.repo)
	session, err := analyticsService.StartSession(c.Request.Context(), videoID, req.UserID, req.DeviceInfo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start session"})
		return
	}

	c.JSON(http.StatusOK, session)
}

func (s *Server) handleEndPlaybackSession(c *gin.Context) {
	sessionID := c.Param("session_id")

	analyticsService := analytics.NewService(s.repo)
	if err := analyticsService.EndSession(c.Request.Context(), sessionID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to end session"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "session ended successfully"})
}

func (s *Server) handleGetVideoAnalytics(c *gin.Context) {
	videoID := c.Param("id")

	analyticsService := analytics.NewService(s.repo)

	// Aggregate latest analytics
	analytics, err := analyticsService.AggregateVideoAnalytics(c.Request.Context(), videoID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get analytics"})
		return
	}

	c.JSON(http.StatusOK, analytics)
}

func (s *Server) handleGetTrendingVideos(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		limit = 10
	}

	analyticsService := analytics.NewService(s.repo)
	trending, err := analyticsService.GetTrendingVideos(c.Request.Context(), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get trending videos"})
		return
	}

	c.JSON(http.StatusOK, trending)
}

func (s *Server) handleGetVideoHeatmap(c *gin.Context) {
	videoID := c.Param("id")

	resolutionStr := c.DefaultQuery("resolution", "10")
	resolution, err := strconv.Atoi(resolutionStr)
	if err != nil || resolution < 1 {
		resolution = 10
	}

	analyticsService := analytics.NewService(s.repo)
	heatmap, err := analyticsService.GenerateHeatmap(c.Request.Context(), videoID, resolution)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate heatmap"})
		return
	}

	c.JSON(http.StatusOK, heatmap)
}

func (s *Server) handleGetQoEMetrics(c *gin.Context) {
	videoID := c.Param("id")
	period := c.DefaultQuery("period", "daily")

	startStr := c.Query("start")
	endStr := c.Query("end")

	var start, end time.Time
	var err error

	if startStr != "" {
		start, err = time.Parse(time.RFC3339, startStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start time"})
			return
		}
	} else {
		start = time.Now().AddDate(0, 0, -7) // Default: last 7 days
	}

	if endStr != "" {
		end, err = time.Parse(time.RFC3339, endStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid end time"})
			return
		}
	} else {
		end = time.Now()
	}

	analyticsService := analytics.NewService(s.repo)
	metrics, err := analyticsService.GetQoEMetrics(c.Request.Context(), videoID, period, start, end)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get QoE metrics"})
		return
	}

	c.JSON(http.StatusOK, metrics)
}

// Register Phase 7 routes
func (s *Server) registerPhase7Routes() {
	v1 := s.router.Group("/api/v1")

	// Scene detection
	v1.POST("/videos/:id/scenes/detect", s.handleSceneDetection)

	// Watermarking
	v1.POST("/videos/:id/watermark", s.handleApplyWatermark)

	// Concatenation
	v1.POST("/videos/concatenate", s.handleConcatenateVideos)

	// Analytics
	analytics := v1.Group("/analytics")
	{
		analytics.POST("/events", s.handleTrackPlaybackEvent)
		analytics.POST("/sessions/:id/start", s.handleStartPlaybackSession)
		analytics.POST("/sessions/:session_id/end", s.handleEndPlaybackSession)
		analytics.GET("/videos/:id", s.handleGetVideoAnalytics)
		analytics.GET("/videos/:id/heatmap", s.handleGetVideoHeatmap)
		analytics.GET("/videos/:id/qoe", s.handleGetQoEMetrics)
		analytics.GET("/trending", s.handleGetTrendingVideos)
	}
}
