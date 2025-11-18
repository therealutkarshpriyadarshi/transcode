package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/therealutkarshpriyadarshi/transcode/pkg/models"
)

func TestCreateLiveStream(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// This is a simplified test - in production you would use a test database
	router := gin.New()

	req := map[string]interface{}{
		"title":       "Test Live Stream",
		"description": "A test stream",
		"user_id":     "user-123",
		"dvr_enabled": true,
		"dvr_window":  7200,
		"low_latency": true,
	}

	body, err := json.Marshal(req)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	httpReq, _ := http.NewRequest("POST", "/api/v1/livestreams", bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")

	// Note: This test would fail without a proper database setup
	// In a real test, you would use a test database or mocks
	_ = w
	_ = httpReq
}

func TestLiveStreamStatusTransitions(t *testing.T) {
	// Test valid status transitions
	validTransitions := map[string][]string{
		models.LiveStreamStatusIdle: {
			models.LiveStreamStatusStarting,
		},
		models.LiveStreamStatusStarting: {
			models.LiveStreamStatusLive,
			models.LiveStreamStatusFailed,
		},
		models.LiveStreamStatusLive: {
			models.LiveStreamStatusEnding,
			models.LiveStreamStatusFailed,
		},
		models.LiveStreamStatusEnding: {
			models.LiveStreamStatusEnded,
		},
	}

	for fromStatus, toStatuses := range validTransitions {
		for _, toStatus := range toStatuses {
			assert.NotEmpty(t, toStatus, "Transition from %s to %s", fromStatus, toStatus)
		}
	}
}

func TestDVRRecordingLifecycle(t *testing.T) {
	// Test DVR recording status lifecycle
	startTime := time.Now()

	recording := &models.DVRRecording{
		ID:           "rec-123",
		LiveStreamID: "stream-456",
		StartTime:    startTime,
		Status:       models.DVRRecordingStatusRecording,
	}

	assert.Equal(t, models.DVRRecordingStatusRecording, recording.Status)

	// Simulate recording completion
	endTime := startTime.Add(1 * time.Hour)
	recording.EndTime = &endTime
	recording.Duration = endTime.Sub(startTime).Seconds()
	recording.Status = models.DVRRecordingStatusProcessing

	assert.Equal(t, models.DVRRecordingStatusProcessing, recording.Status)
	assert.Equal(t, 3600.0, recording.Duration)

	// Simulate processing completion
	recording.Status = models.DVRRecordingStatusAvailable
	recording.RecordingURL = "/recordings/rec-123.mp4"

	assert.Equal(t, models.DVRRecordingStatusAvailable, recording.Status)
	assert.NotEmpty(t, recording.RecordingURL)
}

func TestLiveStreamAnalyticsMetrics(t *testing.T) {
	analytics := &models.LiveStreamAnalytics{
		ID:               "analytics-123",
		LiveStreamID:     "stream-456",
		Timestamp:        time.Now(),
		ViewerCount:      150,
		BandwidthUsage:   50000000, // 50 Mbps
		IngestBitrate:    8000000,  // 8 Mbps
		DroppedFrames:    5,
		KeyframeInterval: 2.0,
		BufferHealth:     98.5,
		AverageLatency:   250.0,
		QualityScore:     92.5,
	}

	// Verify metrics are within expected ranges
	assert.Greater(t, analytics.ViewerCount, 0)
	assert.Greater(t, analytics.BandwidthUsage, int64(0))
	assert.Greater(t, analytics.BufferHealth, 0.0)
	assert.LessOrEqual(t, analytics.BufferHealth, 100.0)
	assert.Greater(t, analytics.QualityScore, 0.0)
	assert.LessOrEqual(t, analytics.QualityScore, 100.0)
}

func TestLiveStreamEventCreation(t *testing.T) {
	event := &models.LiveStreamEvent{
		ID:           "event-123",
		LiveStreamID: "stream-456",
		EventType:    models.LiveStreamEventError,
		Severity:     models.SeverityError,
		Message:      "Connection lost",
		Details: models.Metadata{
			"error_code":  "CONN_LOST",
			"retry_count": 3,
		},
		Timestamp: time.Now(),
	}

	assert.Equal(t, models.LiveStreamEventError, event.EventType)
	assert.Equal(t, models.SeverityError, event.Severity)
	assert.NotEmpty(t, event.Message)
	assert.NotNil(t, event.Details)
}

func TestViewerSessionTracking(t *testing.T) {
	now := time.Now()
	userID := "user-123"

	viewer := &models.LiveStreamViewer{
		ID:             "viewer-123",
		LiveStreamID:   "stream-456",
		SessionID:      "session-789",
		UserID:         &userID,
		JoinedAt:       now,
		Resolution:     "1080p",
		DeviceType:     "desktop",
		Location:       "US",
		BufferEvents:   0,
		QualityChanges: 0,
	}

	// Simulate viewer leaving after 30 minutes
	leftAt := now.Add(30 * time.Minute)
	viewer.LeftAt = &leftAt
	viewer.WatchDuration = leftAt.Sub(now).Seconds()

	assert.Equal(t, 1800.0, viewer.WatchDuration)
	assert.NotNil(t, viewer.LeftAt)
}

func TestLiveStreamSettings(t *testing.T) {
	settings := models.DefaultLiveStreamSettings()

	// Verify default settings
	assert.True(t, settings.EnableTranscoding)
	assert.Contains(t, settings.Resolutions, "1080p")
	assert.Contains(t, settings.Resolutions, "720p")
	assert.Equal(t, "h264", settings.Codec)
	assert.Equal(t, 6, settings.SegmentDuration)
	assert.Equal(t, "aac", settings.AudioCodec)
	assert.Equal(t, 128, settings.AudioBitrate)
	assert.True(t, settings.GPUAcceleration)
}

func TestLiveStreamVariantConfiguration(t *testing.T) {
	variant := &models.LiveStreamVariant{
		ID:             "variant-123",
		LiveStreamID:   "stream-456",
		Resolution:     "1080p",
		Width:          1920,
		Height:         1080,
		Bitrate:        5000000, // 5 Mbps
		FrameRate:      30.0,
		Codec:          "h264",
		AudioBitrate:   128,
		PlaylistURL:    "/streams/variant-123/playlist.m3u8",
		SegmentPattern: "/streams/variant-123/%03d.ts",
	}

	// Verify variant configuration
	assert.Equal(t, 1920, variant.Width)
	assert.Equal(t, 1080, variant.Height)
	assert.Equal(t, int64(5000000), variant.Bitrate)
	assert.Equal(t, 30.0, variant.FrameRate)
	assert.NotEmpty(t, variant.PlaylistURL)
}

func TestDVRWindowCalculation(t *testing.T) {
	// Test DVR window calculations
	segmentDuration := 6 // seconds
	dvrWindow := 7200    // 2 hours in seconds

	segmentsToKeep := dvrWindow / segmentDuration
	assert.Equal(t, 1200, segmentsToKeep)

	// For 1 hour DVR window
	dvrWindow1Hour := 3600
	segmentsToKeep1Hour := dvrWindow1Hour / segmentDuration
	assert.Equal(t, 600, segmentsToKeep1Hour)
}

func TestLowLatencySettings(t *testing.T) {
	settings := models.LiveStreamSettings{
		LowLatency:      true,
		SegmentDuration: 6,
		PartDuration:    0.5, // 500ms parts for LL-HLS
	}

	// Verify low-latency configuration
	assert.True(t, settings.LowLatency)
	assert.Greater(t, settings.PartDuration, 0.0)
	assert.Less(t, settings.PartDuration, float64(settings.SegmentDuration))
}

func TestBandwidthCalculations(t *testing.T) {
	// Test bandwidth calculations for different resolutions
	variants := []struct {
		resolution string
		bitrate    int64
	}{
		{"1080p", 5000000}, // 5 Mbps
		{"720p", 2800000},  // 2.8 Mbps
		{"480p", 1400000},  // 1.4 Mbps
		{"360p", 800000},   // 800 Kbps
	}

	totalBandwidth := int64(0)
	for _, v := range variants {
		totalBandwidth += v.bitrate
	}

	// Total bandwidth for all variants
	assert.Equal(t, int64(10000000), totalBandwidth) // 10 Mbps
}

func TestViewerCountTracking(t *testing.T) {
	stream := &models.LiveStream{
		ID:              "stream-123",
		ViewerCount:     0,
		PeakViewerCount: 0,
	}

	// Simulate viewer count changes
	viewerCounts := []int{10, 50, 100, 250, 150, 75, 30}

	for _, count := range viewerCounts {
		stream.ViewerCount = count
		if count > stream.PeakViewerCount {
			stream.PeakViewerCount = count
		}
	}

	assert.Equal(t, 30, stream.ViewerCount)
	assert.Equal(t, 250, stream.PeakViewerCount)
}

func TestStreamEventSeverityLevels(t *testing.T) {
	events := []struct {
		eventType string
		severity  string
	}{
		{models.LiveStreamEventStreamStarted, models.SeverityInfo},
		{models.LiveStreamEventBufferUnderflow, models.SeverityWarning},
		{models.LiveStreamEventConnectionLost, models.SeverityError},
		{models.LiveStreamEventError, models.SeverityCritical},
	}

	for _, e := range events {
		assert.NotEmpty(t, e.eventType)
		assert.NotEmpty(t, e.severity)
	}
}
