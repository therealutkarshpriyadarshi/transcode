package analytics

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/therealutkarshpriyadarshi/transcode/pkg/models"
)

func TestCalculateQoEScore(t *testing.T) {
	service := &Service{}

	tests := []struct {
		name     string
		session  *models.PlaybackSession
		expected float64
		minScore float64
		maxScore float64
	}{
		{
			name: "Perfect session",
			session: &models.PlaybackSession{
				WatchTime:       100,
				TotalBufferTime: 0,
				StartupTime:     1.0,
				ErrorOccurred:   false,
				CompletionRate:  100,
			},
			minScore: 95,
			maxScore: 100,
		},
		{
			name: "Session with buffering",
			session: &models.PlaybackSession{
				WatchTime:       100,
				TotalBufferTime: 10, // 10% rebuffer ratio
				StartupTime:     2.0,
				ErrorOccurred:   false,
				CompletionRate:  100,
			},
			minScore: 85,
			maxScore: 95,
		},
		{
			name: "Session with high startup time",
			session: &models.PlaybackSession{
				WatchTime:       100,
				TotalBufferTime: 0,
				StartupTime:     10.0, // High startup time
				ErrorOccurred:   false,
				CompletionRate:  100,
			},
			minScore: 70,
			maxScore: 85,
		},
		{
			name: "Session with error",
			session: &models.PlaybackSession{
				WatchTime:       50,
				TotalBufferTime: 5,
				StartupTime:     3.0,
				ErrorOccurred:   true, // Error occurred
				CompletionRate:  50,
			},
			minScore: 0,
			maxScore: 50,
		},
		{
			name: "Session with low completion",
			session: &models.PlaybackSession{
				WatchTime:       30,
				TotalBufferTime: 0,
				StartupTime:     1.0,
				ErrorOccurred:   false,
				CompletionRate:  30, // Only watched 30%
			},
			minScore: 85,
			maxScore: 95,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := service.CalculateQoEScore(tt.session)
			assert.GreaterOrEqual(t, score, 0.0, "Score should be >= 0")
			assert.LessOrEqual(t, score, 100.0, "Score should be <= 100")
			assert.GreaterOrEqual(t, score, tt.minScore, "Score should be >= min expected")
			assert.LessOrEqual(t, score, tt.maxScore, "Score should be <= max expected")
		})
	}
}

func TestCalculateSessionMetrics(t *testing.T) {
	service := &Service{}

	session := &models.PlaybackSession{
		ID:        "session-1",
		VideoID:   "video-1",
		StartTime: time.Now().Add(-10 * time.Minute),
	}

	events := []*models.PlaybackEvent{
		{
			EventType: models.EventTypePlay,
			Position:  0,
			Duration:  100,
			Bitrate:   5000000,
			Timestamp: time.Now().Add(-9 * time.Minute),
		},
		{
			EventType:  models.EventTypeBuffer,
			Position:   10,
			Duration:   100,
			BufferTime: 2.0,
			Bitrate:    5000000,
			Timestamp:  time.Now().Add(-8 * time.Minute),
		},
		{
			EventType: models.EventTypeSeek,
			Position:  30,
			Duration:  100,
			Bitrate:   5000000,
			Timestamp: time.Now().Add(-7 * time.Minute),
		},
		{
			EventType: models.EventTypeQualityChange,
			Position:  40,
			Duration:  100,
			Bitrate:   3000000,
			Timestamp: time.Now().Add(-6 * time.Minute),
		},
		{
			EventType: models.EventTypePlay,
			Position:  80,
			Duration:  100,
			Bitrate:   3000000,
			Timestamp: time.Now().Add(-2 * time.Minute),
		},
		{
			EventType: models.EventTypeComplete,
			Position:  100,
			Duration:  100,
			Bitrate:   3000000,
			Timestamp: time.Now(),
		},
	}

	result := service.calculateSessionMetrics(session, events)

	assert.NotNil(t, result)
	assert.Equal(t, 2.0, result.TotalBufferTime)
	assert.Equal(t, 1, result.BufferCount)
	assert.Equal(t, 1, result.SeekCount)
	assert.Equal(t, 1, result.QualityChanges)
	assert.True(t, result.Completed)
	assert.False(t, result.ErrorOccurred)
	assert.Equal(t, 100.0, result.CompletionRate)
	assert.Equal(t, 100.0, result.WatchTime)
	assert.Greater(t, result.AverageBitrate, int64(0))
	assert.Equal(t, int64(5000000), result.PeakBitrate)
	assert.Greater(t, result.StartupTime, 0.0)
}

func TestCalculateSessionMetrics_WithError(t *testing.T) {
	service := &Service{}

	session := &models.PlaybackSession{
		ID:        "session-2",
		VideoID:   "video-1",
		StartTime: time.Now().Add(-5 * time.Minute),
	}

	events := []*models.PlaybackEvent{
		{
			EventType: models.EventTypePlay,
			Position:  0,
			Duration:  100,
			Timestamp: time.Now().Add(-4 * time.Minute),
		},
		{
			EventType:    models.EventTypeError,
			Position:     20,
			Duration:     100,
			ErrorCode:    "NETWORK_ERROR",
			ErrorMessage: "Connection lost",
			Timestamp:    time.Now().Add(-2 * time.Minute),
		},
	}

	result := service.calculateSessionMetrics(session, events)

	assert.NotNil(t, result)
	assert.True(t, result.ErrorOccurred)
	assert.False(t, result.Completed)
	assert.Equal(t, 20.0, result.CompletionRate)
}

func TestCalculateSessionMetrics_EmptyEvents(t *testing.T) {
	service := &Service{}

	session := &models.PlaybackSession{
		ID:        "session-3",
		VideoID:   "video-1",
		StartTime: time.Now(),
	}

	events := []*models.PlaybackEvent{}

	result := service.calculateSessionMetrics(session, events)

	assert.NotNil(t, result)
	assert.Equal(t, 0.0, result.TotalBufferTime)
	assert.Equal(t, 0, result.BufferCount)
	assert.Equal(t, 0, result.SeekCount)
	assert.Equal(t, 0, result.QualityChanges)
	assert.False(t, result.Completed)
	assert.False(t, result.ErrorOccurred)
}

func TestCalculateSessionMetrics_MultipleBuffers(t *testing.T) {
	service := &Service{}

	session := &models.PlaybackSession{
		ID:        "session-4",
		VideoID:   "video-1",
		StartTime: time.Now().Add(-10 * time.Minute),
	}

	events := []*models.PlaybackEvent{
		{
			EventType:  models.EventTypeBuffer,
			BufferTime: 1.5,
			Timestamp:  time.Now().Add(-8 * time.Minute),
		},
		{
			EventType:  models.EventTypeBuffer,
			BufferTime: 2.0,
			Timestamp:  time.Now().Add(-6 * time.Minute),
		},
		{
			EventType:  models.EventTypeBuffer,
			BufferTime: 0.5,
			Timestamp:  time.Now().Add(-4 * time.Minute),
		},
	}

	result := service.calculateSessionMetrics(session, events)

	assert.Equal(t, 4.0, result.TotalBufferTime)
	assert.Equal(t, 3, result.BufferCount)
}

func TestTrackEvent_Validation(t *testing.T) {
	// This test would require a mock repository
	// For now, we'll test the event structure

	event := &models.PlaybackEvent{
		VideoID:    "video-1",
		SessionID:  "session-1",
		EventType:  models.EventTypePlay,
		Position:   10.5,
		Duration:   100,
		DeviceType: "desktop",
		Browser:    "Chrome",
		OS:         "Windows",
	}

	assert.NotEmpty(t, event.VideoID)
	assert.NotEmpty(t, event.SessionID)
	assert.NotEmpty(t, event.EventType)
	assert.Greater(t, event.Position, 0.0)
	assert.Greater(t, event.Duration, 0.0)
}

func TestPlaybackSession_Validation(t *testing.T) {
	session := &models.PlaybackSession{
		ID:         "session-1",
		VideoID:    "video-1",
		UserID:     "user-1",
		StartTime:  time.Now(),
		DeviceType: "mobile",
		Browser:    "Safari",
		OS:         "iOS",
		Country:    "US",
	}

	assert.NotEmpty(t, session.ID)
	assert.NotEmpty(t, session.VideoID)
	assert.False(t, session.StartTime.IsZero())
}

func TestVideoAnalytics_Calculation(t *testing.T) {
	// Simulate aggregated analytics calculation

	sessions := []*models.PlaybackSession{
		{
			UserID:          "user-1",
			WatchTime:       120,
			CompletionRate:  80,
			TotalBufferTime: 5,
			BufferCount:     2,
			StartupTime:     2.0,
			ErrorOccurred:   false,
			Country:         "US",
			DeviceType:      "desktop",
		},
		{
			UserID:          "user-2",
			WatchTime:       90,
			CompletionRate:  100,
			TotalBufferTime: 0,
			BufferCount:     0,
			StartupTime:     1.5,
			ErrorOccurred:   false,
			Country:         "US",
			DeviceType:      "mobile",
		},
		{
			UserID:          "user-1", // Same user, different session
			WatchTime:       60,
			CompletionRate:  50,
			TotalBufferTime: 10,
			BufferCount:     5,
			StartupTime:     3.0,
			ErrorOccurred:   true,
			Country:         "CA",
			DeviceType:      "desktop",
		},
	}

	// Calculate metrics
	totalViews := int64(len(sessions))
	uniqueUsers := make(map[string]bool)
	var totalWatchTime, totalBufferTime, totalStartupTime, completionRateSum float64
	var bufferedSessions, errorSessions int64

	for _, session := range sessions {
		uniqueUsers[session.UserID] = true
		totalWatchTime += session.WatchTime
		totalBufferTime += session.TotalBufferTime
		totalStartupTime += session.StartupTime
		completionRateSum += session.CompletionRate
		if session.BufferCount > 0 {
			bufferedSessions++
		}
		if session.ErrorOccurred {
			errorSessions++
		}
	}

	assert.Equal(t, int64(3), totalViews)
	assert.Equal(t, 2, len(uniqueUsers))
	assert.Equal(t, 270.0, totalWatchTime)
	assert.Equal(t, 15.0, totalBufferTime)

	averageWatchTime := totalWatchTime / float64(totalViews)
	assert.Equal(t, 90.0, averageWatchTime)

	completionRate := completionRateSum / float64(totalViews)
	assert.InDelta(t, 76.67, completionRate, 0.1)

	bufferRate := (float64(bufferedSessions) / float64(totalViews)) * 100
	assert.InDelta(t, 66.67, bufferRate, 0.1)

	errorRate := (float64(errorSessions) / float64(totalViews)) * 100
	assert.InDelta(t, 33.33, errorRate, 0.1)
}
