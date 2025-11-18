package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/therealutkarshpriyadarshi/transcode/pkg/models"
)

// CreateLiveStream creates a new live stream
func (api *API) createLiveStream(c *gin.Context) {
	var req struct {
		Title       string                     `json:"title" binding:"required"`
		Description string                     `json:"description"`
		UserID      string                     `json:"user_id" binding:"required"`
		DVREnabled  bool                       `json:"dvr_enabled"`
		DVRWindow   int                        `json:"dvr_window"` // in seconds
		LowLatency  bool                       `json:"low_latency"`
		Settings    *models.LiveStreamSettings `json:"settings"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Generate stream key
	streamKey := uuid.New().String()

	// Get server host from config or use default
	rtmpHost := "localhost" // In production, this would come from config
	rtmpPort := 1935
	rtmpIngestURL := fmt.Sprintf("rtmp://%s:%d/live/%s", rtmpHost, rtmpPort, streamKey)

	// Use default settings if not provided
	settings := models.DefaultLiveStreamSettings()
	if req.Settings != nil {
		settings = *req.Settings
	}

	// Set default DVR window if not specified
	dvrWindow := req.DVRWindow
	if dvrWindow == 0 {
		dvrWindow = 7200 // 2 hours default
	}

	stream := &models.LiveStream{
		Title:         req.Title,
		Description:   req.Description,
		UserID:        req.UserID,
		StreamKey:     streamKey,
		RTMPIngestURL: rtmpIngestURL,
		Status:        models.LiveStreamStatusIdle,
		DVREnabled:    req.DVREnabled,
		DVRWindow:     dvrWindow,
		LowLatency:    req.LowLatency,
		Settings:      settings,
		Metadata:      models.Metadata{},
	}

	if err := api.repo.CreateLiveStream(c.Request.Context(), stream); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to create live stream: %v", err)})
		return
	}

	c.JSON(http.StatusCreated, stream)
}

// GetLiveStream retrieves a live stream by ID
func (api *API) getLiveStream(c *gin.Context) {
	streamID := c.Param("id")

	stream, err := api.repo.GetLiveStream(c.Request.Context(), streamID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Live stream not found"})
		return
	}

	c.JSON(http.StatusOK, stream)
}

// ListLiveStreams lists all live streams with optional filtering
func (api *API) listLiveStreams(c *gin.Context) {
	userID := c.Query("user_id")
	status := c.Query("status")
	limit := 20
	offset := 0

	streams, err := api.repo.ListLiveStreams(c.Request.Context(), userID, status, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"streams": streams,
		"limit":   limit,
		"offset":  offset,
	})
}

// StartLiveStream starts a live stream
func (api *API) startLiveStream(c *gin.Context) {
	streamID := c.Param("id")

	// Get stream from database
	stream, err := api.repo.GetLiveStream(c.Request.Context(), streamID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Live stream not found"})
		return
	}

	// Check if stream is already live
	if stream.Status == models.LiveStreamStatusLive || stream.Status == models.LiveStreamStatusStarting {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Stream is already active"})
		return
	}

	// Update status to starting
	if err := api.repo.UpdateLiveStreamStatus(c.Request.Context(), streamID, models.LiveStreamStatusStarting); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update stream status"})
		return
	}

	// In production, this would trigger the RTMP server to start accepting the stream
	// and begin transcoding. For now, we'll update the status after a delay.
	go func() {
		time.Sleep(2 * time.Second)
		now := time.Now()
		api.repo.UpdateLiveStreamStartTime(c.Request.Context(), streamID, &now)
		api.repo.UpdateLiveStreamStatus(c.Request.Context(), streamID, models.LiveStreamStatusLive)

		// Create initial DVR recording if enabled
		if stream.DVREnabled {
			recording := &models.DVRRecording{
				LiveStreamID: streamID,
				StartTime:    now,
				Status:       models.DVRRecordingStatusRecording,
			}
			api.repo.CreateDVRRecording(c.Request.Context(), recording)
		}
	}()

	c.JSON(http.StatusOK, gin.H{
		"message":         "Stream is starting",
		"stream_id":       streamID,
		"rtmp_ingest_url": stream.RTMPIngestURL,
		"stream_key":      stream.StreamKey,
	})
}

// StopLiveStream stops a live stream
func (api *API) stopLiveStream(c *gin.Context) {
	streamID := c.Param("id")

	// Get stream from database
	stream, err := api.repo.GetLiveStream(c.Request.Context(), streamID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Live stream not found"})
		return
	}

	// Check if stream is live
	if stream.Status != models.LiveStreamStatusLive {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Stream is not live"})
		return
	}

	// Update status to ending
	if err := api.repo.UpdateLiveStreamStatus(c.Request.Context(), streamID, models.LiveStreamStatusEnding); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update stream status"})
		return
	}

	// Stop the stream
	go func() {
		time.Sleep(1 * time.Second)
		now := time.Now()
		api.repo.UpdateLiveStreamEndTime(c.Request.Context(), streamID, &now)
		api.repo.UpdateLiveStreamStatus(c.Request.Context(), streamID, models.LiveStreamStatusEnded)

		// Finalize DVR recordings
		recordings, _ := api.repo.ListDVRRecordings(c.Request.Context(), streamID)
		for _, rec := range recordings {
			if rec.Status == models.DVRRecordingStatusRecording {
				api.repo.UpdateDVRRecordingStatus(c.Request.Context(), rec.ID, models.DVRRecordingStatusProcessing)
			}
		}
	}()

	c.JSON(http.StatusOK, gin.H{"message": "Stream is stopping"})
}

// DeleteLiveStream deletes a live stream
func (api *API) deleteLiveStream(c *gin.Context) {
	streamID := c.Param("id")

	// Get stream to check status
	stream, err := api.repo.GetLiveStream(c.Request.Context(), streamID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Live stream not found"})
		return
	}

	// Can't delete a live stream that's currently active
	if stream.Status == models.LiveStreamStatusLive || stream.Status == models.LiveStreamStatusStarting {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot delete an active stream"})
		return
	}

	if err := api.repo.DeleteLiveStream(c.Request.Context(), streamID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete stream"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Stream deleted successfully"})
}

// GetLiveStreamVariants retrieves all quality variants for a live stream
func (api *API) getLiveStreamVariants(c *gin.Context) {
	streamID := c.Param("id")

	variants, err := api.repo.GetLiveStreamVariants(c.Request.Context(), streamID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"variants": variants})
}

// GetLiveStreamAnalytics retrieves analytics for a live stream
func (api *API) getLiveStreamAnalytics(c *gin.Context) {
	streamID := c.Param("id")

	// Parse time range from query params
	fromStr := c.Query("from")
	toStr := c.Query("to")

	var from, to time.Time
	var err error

	if fromStr != "" {
		from, err = time.Parse(time.RFC3339, fromStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid 'from' timestamp"})
			return
		}
	} else {
		from = time.Now().Add(-1 * time.Hour) // Default to last hour
	}

	if toStr != "" {
		to, err = time.Parse(time.RFC3339, toStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid 'to' timestamp"})
			return
		}
	} else {
		to = time.Now()
	}

	analytics, err := api.repo.GetLiveStreamAnalytics(c.Request.Context(), streamID, from, to)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"analytics": analytics})
}

// GetLiveStreamEvents retrieves events for a live stream
func (api *API) getLiveStreamEvents(c *gin.Context) {
	streamID := c.Param("id")
	limit := 100

	events, err := api.repo.GetLiveStreamEvents(c.Request.Context(), streamID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"events": events})
}

// GetDVRRecordings retrieves DVR recordings for a live stream
func (api *API) getDVRRecordings(c *gin.Context) {
	streamID := c.Param("id")

	recordings, err := api.repo.ListDVRRecordings(c.Request.Context(), streamID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"recordings": recordings})
}

// GetDVRRecording retrieves a specific DVR recording
func (api *API) getDVRRecording(c *gin.Context) {
	recordingID := c.Param("recording_id")

	recording, err := api.repo.GetDVRRecording(c.Request.Context(), recordingID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Recording not found"})
		return
	}

	c.JSON(http.StatusOK, recording)
}

// GetActiveViewers retrieves currently active viewers for a live stream
func (api *API) getActiveViewers(c *gin.Context) {
	streamID := c.Param("id")

	viewers, err := api.repo.GetActiveViewers(c.Request.Context(), streamID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Update viewer count in stream
	api.repo.UpdateLiveStreamViewerCount(c.Request.Context(), streamID, len(viewers))

	c.JSON(http.StatusOK, gin.H{
		"viewers": viewers,
		"count":   len(viewers),
	})
}

// TrackViewerSession tracks a viewer watching a live stream
func (api *API) trackViewerSession(c *gin.Context) {
	streamID := c.Param("id")

	var req struct {
		SessionID      string  `json:"session_id" binding:"required"`
		UserID         *string `json:"user_id"`
		Resolution     string  `json:"resolution"`
		DeviceType     string  `json:"device_type"`
		Location       string  `json:"location"`
		BufferEvents   int     `json:"buffer_events"`
		QualityChanges int     `json:"quality_changes"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	viewer := &models.LiveStreamViewer{
		LiveStreamID:   streamID,
		SessionID:      req.SessionID,
		UserID:         req.UserID,
		JoinedAt:       time.Now(),
		Resolution:     req.Resolution,
		DeviceType:     req.DeviceType,
		Location:       req.Location,
		IPAddress:      c.ClientIP(),
		UserAgent:      c.Request.UserAgent(),
		BufferEvents:   req.BufferEvents,
		QualityChanges: req.QualityChanges,
	}

	if err := api.repo.TrackViewer(c.Request.Context(), viewer); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to track viewer"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Viewer tracked successfully"})
}
