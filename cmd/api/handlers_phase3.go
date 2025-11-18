package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/therealutkarshpriyadarshi/transcode/internal/middleware"
	"github.com/therealutkarshpriyadarshi/transcode/pkg/models"
	"golang.org/x/crypto/bcrypt"
)

// Auth handlers

func (api *API) register(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required,min=8"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user := &models.User{
		ID:       uuid.New().String(),
		Email:    req.Email,
		Quota:    100, // Default quota
		IsActive: true,
	}

	if err := api.repo.CreateUser(c.Request.Context(), user, req.Password); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":      user.ID,
		"email":   user.Email,
		"api_key": user.APIKey,
	})
}

func (api *API) login(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := api.repo.GetUserByEmail(c.Request.Context(), req.Email)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	if !user.IsActive {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Account is inactive"})
		return
	}

	token, err := middleware.GenerateToken(user.ID, user.Email, 24*time.Hour)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":      token,
		"user_id":    user.ID,
		"email":      user.Email,
		"api_key":    user.APIKey,
		"quota":      user.Quota,
		"used_quota": user.UsedQuota,
	})
}

// Multipart upload handlers

func (api *API) initiateUpload(c *gin.Context) {
	var req struct {
		Filename  string `json:"filename" binding:"required"`
		TotalSize int64  `json:"total_size" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	upload, err := api.uploadService.InitiateUpload(req.Filename, req.TotalSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, upload)
}

func (api *API) uploadPart(c *gin.Context) {
	uploadID := c.Param("upload_id")
	partNumberStr := c.Param("part_number")

	partNumber, err := strconv.Atoi(partNumberStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid part number"})
		return
	}

	// Read request body
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read body"})
		return
	}

	part, err := api.uploadService.UploadPart(uploadID, partNumber, bytes.NewReader(body))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, part)
}

func (api *API) completeUpload(c *gin.Context) {
	uploadID := c.Param("upload_id")

	filePath, err := api.uploadService.CompleteUpload(uploadID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Extract video metadata
	videoInfo, err := api.ffmpeg.ExtractVideoInfo(c.Request.Context(), filePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to extract metadata: %v", err)})
		return
	}

	// Get upload info
	upload, err := api.uploadService.GetUpload(uploadID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Create video record
	video := &models.Video{
		ID:       uuid.New().String(),
		Filename: upload.Filename,
		Size:     upload.TotalSize,
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
	storageKey := fmt.Sprintf("videos/%s/original/%s", video.ID, upload.Filename)
	if err := api.storage.UploadFile(c.Request.Context(), storageKey, filePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to upload: %v", err)})
		return
	}

	video.OriginalURL = storageKey

	// Get user ID from context
	if userID, exists := middleware.GetUserID(c); exists {
		video.UserID = &userID
	}

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

func (api *API) abortUpload(c *gin.Context) {
	uploadID := c.Param("upload_id")

	if err := api.uploadService.AbortUpload(uploadID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Upload aborted"})
}

func (api *API) getUploadStatus(c *gin.Context) {
	uploadID := c.Param("upload_id")

	upload, err := api.uploadService.GetUpload(uploadID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Upload not found"})
		return
	}

	c.JSON(http.StatusOK, upload)
}

// Webhook handlers

func (api *API) createWebhook(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req struct {
		URL    string               `json:"url" binding:"required,url"`
		Events models.WebhookEvents `json:"events" binding:"required"`
		Secret string               `json:"secret"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	webhook := &models.Webhook{
		ID:       uuid.New().String(),
		UserID:   userID,
		URL:      req.URL,
		Events:   req.Events,
		Secret:   req.Secret,
		IsActive: true,
	}

	if err := api.repo.CreateWebhook(c.Request.Context(), webhook); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create webhook"})
		return
	}

	c.JSON(http.StatusCreated, webhook)
}

func (api *API) listWebhooks(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	webhooks, err := api.repo.GetUserWebhooks(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"webhooks": webhooks})
}

// Job control handlers

func (api *API) pauseJob(c *gin.Context) {
	jobID := c.Param("id")

	if err := api.repo.PauseJob(c.Request.Context(), jobID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Job paused"})
}

func (api *API) resumeJob(c *gin.Context) {
	jobID := c.Param("id")

	if err := api.repo.ResumeJob(c.Request.Context(), jobID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Job resumed"})
}

// Monitoring handlers

func (api *API) getMetrics(c *gin.Context) {
	if api.monitor == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Monitoring not available"})
		return
	}

	metrics := api.monitor.GetMetrics()
	c.JSON(http.StatusOK, metrics)
}

func (api *API) getWorkerHealth(c *gin.Context) {
	if api.monitor == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Monitoring not available"})
		return
	}

	workers := api.monitor.GetWorkerHealth()
	c.JSON(http.StatusOK, gin.H{"workers": workers})
}

func (api *API) getSystemHealth(c *gin.Context) {
	if api.monitor == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Monitoring not available"})
		return
	}

	health := api.monitor.GetSystemHealth()
	alerts := api.monitor.GetAlerts()

	c.JSON(http.StatusOK, gin.H{
		"status": health,
		"alerts": alerts,
	})
}

func (api *API) getQueueStats(c *gin.Context) {
	queueDepth, err := api.queue.GetQueueDepth()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get queue depth"})
		return
	}

	dlqDepth, err := api.queue.GetDLQDepth()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get DLQ depth"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"queue_depth": queueDepth,
		"dlq_depth":   dlqDepth,
	})
}
