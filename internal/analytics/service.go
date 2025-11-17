package analytics

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/therealutkarshpriyadarshi/transcode/internal/database"
	"github.com/therealutkarshpriyadarshi/transcode/pkg/models"
)

// Service handles analytics tracking and aggregation
type Service struct {
	repo *database.Repository
}

// NewService creates a new analytics service
func NewService(repo *database.Repository) *Service {
	return &Service{
		repo: repo,
	}
}

// TrackEvent records a playback event
func (s *Service) TrackEvent(ctx context.Context, event *models.PlaybackEvent) error {
	// Generate ID if not set
	if event.ID == "" {
		event.ID = uuid.New().String()
	}

	// Set timestamp if not set
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// Store event (this would typically go to a time-series database or analytics platform)
	// For now, we'll use the main database
	return s.repo.CreatePlaybackEvent(ctx, event)
}

// StartSession creates a new playback session
func (s *Service) StartSession(ctx context.Context, videoID, userID string, deviceInfo map[string]string) (*models.PlaybackSession, error) {
	session := &models.PlaybackSession{
		ID:         uuid.New().String(),
		VideoID:    videoID,
		UserID:     userID,
		StartTime:  time.Now(),
		DeviceType: deviceInfo["device_type"],
		Browser:    deviceInfo["browser"],
		OS:         deviceInfo["os"],
		Country:    deviceInfo["country"],
	}

	if err := s.repo.CreatePlaybackSession(ctx, session); err != nil {
		return nil, err
	}

	return session, nil
}

// EndSession finalizes a playback session
func (s *Service) EndSession(ctx context.Context, sessionID string) error {
	session, err := s.repo.GetPlaybackSession(ctx, sessionID)
	if err != nil {
		return err
	}

	now := time.Now()
	session.EndTime = &now
	session.Duration = now.Sub(session.StartTime).Seconds()

	// Calculate metrics from events
	events, err := s.repo.GetSessionEvents(ctx, sessionID)
	if err != nil {
		return err
	}

	session = s.calculateSessionMetrics(session, events)

	return s.repo.UpdatePlaybackSession(ctx, session)
}

// calculateSessionMetrics computes session metrics from events
func (s *Service) calculateSessionMetrics(session *models.PlaybackSession, events []*models.PlaybackEvent) *models.PlaybackSession {
	var totalBufferTime float64
	var bufferCount int
	var seekCount int
	var qualityChanges int
	var bitrateSum int64
	var bitrateCount int64
	var maxBitrate int64
	var maxPosition float64
	var videoDuration float64
	var startupTime float64
	var firstPlayTime *time.Time

	for _, event := range events {
		// Track position to calculate completion rate
		if event.Position > maxPosition {
			maxPosition = event.Position
		}

		if event.Duration > 0 {
			videoDuration = event.Duration
		}

		switch event.EventType {
		case models.EventTypeBuffer:
			bufferCount++
			totalBufferTime += event.BufferTime
		case models.EventTypeSeek:
			seekCount++
		case models.EventTypeQualityChange:
			qualityChanges++
		case models.EventTypePlay:
			if firstPlayTime == nil {
				firstPlayTime = &event.Timestamp
				// Calculate startup time (time from session start to first play)
				startupTime = event.Timestamp.Sub(session.StartTime).Seconds()
			}
		case models.EventTypeComplete:
			session.Completed = true
		case models.EventTypeError:
			session.ErrorOccurred = true
		}

		// Track bitrate stats
		if event.Bitrate > 0 {
			bitrateSum += event.Bitrate
			bitrateCount++
			if event.Bitrate > maxBitrate {
				maxBitrate = event.Bitrate
			}
		}
	}

	// Update session with calculated metrics
	session.TotalBufferTime = totalBufferTime
	session.BufferCount = bufferCount
	session.SeekCount = seekCount
	session.QualityChanges = qualityChanges
	session.StartupTime = startupTime
	session.PeakBitrate = maxBitrate

	if bitrateCount > 0 {
		session.AverageBitrate = bitrateSum / bitrateCount
	}

	if videoDuration > 0 {
		session.CompletionRate = (maxPosition / videoDuration) * 100
		session.WatchTime = maxPosition
	}

	return session
}

// GetVideoAnalytics retrieves aggregated analytics for a video
func (s *Service) GetVideoAnalytics(ctx context.Context, videoID string) (*models.VideoAnalytics, error) {
	return s.repo.GetVideoAnalytics(ctx, videoID)
}

// AggregateVideoAnalytics computes and updates video analytics from sessions
func (s *Service) AggregateVideoAnalytics(ctx context.Context, videoID string) (*models.VideoAnalytics, error) {
	sessions, err := s.repo.GetVideoSessions(ctx, videoID)
	if err != nil {
		return nil, err
	}

	if len(sessions) == 0 {
		return &models.VideoAnalytics{
			VideoID:     videoID,
			LastUpdated: time.Now(),
		}, nil
	}

	analytics := &models.VideoAnalytics{
		VideoID:            videoID,
		TotalViews:         int64(len(sessions)),
		PopularResolutions: make(map[string]int64),
		GeographicData:     make(map[string]int64),
		DeviceBreakdown:    make(map[string]int64),
		LastUpdated:        time.Now(),
	}

	// Track unique viewers
	uniqueUsers := make(map[string]bool)
	var totalWatchTime float64
	var totalBufferTime float64
	var totalStartupTime float64
	var completionRateSum float64
	var bufferedSessions int64
	var errorSessions int64

	for _, session := range sessions {
		// Unique viewers
		if session.UserID != "" {
			uniqueUsers[session.UserID] = true
		}

		// Watch time (convert to hours)
		totalWatchTime += session.WatchTime

		// Buffer metrics
		totalBufferTime += session.TotalBufferTime
		if session.BufferCount > 0 {
			bufferedSessions++
		}

		// Startup time
		totalStartupTime += session.StartupTime

		// Completion rate
		completionRateSum += session.CompletionRate

		// Errors
		if session.ErrorOccurred {
			errorSessions++
		}

		// Geographic data
		if session.Country != "" {
			analytics.GeographicData[session.Country]++
		}

		// Device breakdown
		if session.DeviceType != "" {
			analytics.DeviceBreakdown[session.DeviceType]++
		}
	}

	// Calculate averages
	analytics.UniqueViewers = int64(len(uniqueUsers))
	analytics.TotalWatchTime = totalWatchTime / 3600 // Convert to hours
	analytics.AverageWatchTime = totalWatchTime / float64(len(sessions))
	analytics.CompletionRate = completionRateSum / float64(len(sessions))
	analytics.AverageBufferTime = totalBufferTime / float64(len(sessions))
	analytics.AverageStartupTime = totalStartupTime / float64(len(sessions))

	if analytics.TotalViews > 0 {
		analytics.BufferRate = (float64(bufferedSessions) / float64(analytics.TotalViews)) * 100
		analytics.ErrorRate = (float64(errorSessions) / float64(analytics.TotalViews)) * 100
	}

	// Save aggregated analytics
	if err := s.repo.UpsertVideoAnalytics(ctx, analytics); err != nil {
		return nil, err
	}

	return analytics, nil
}

// GetQoEMetrics retrieves Quality of Experience metrics
func (s *Service) GetQoEMetrics(ctx context.Context, videoID string, period string, start, end time.Time) ([]*models.QoEMetrics, error) {
	return s.repo.GetQoEMetrics(ctx, videoID, period, start, end)
}

// CalculateQoEScore computes an overall QoE score (0-100)
func (s *Service) CalculateQoEScore(session *models.PlaybackSession) float64 {
	// QoE scoring algorithm based on multiple factors
	// Higher score = better experience

	score := 100.0

	// Penalize for buffering (up to -30 points)
	if session.WatchTime > 0 {
		rebufferRatio := session.TotalBufferTime / session.WatchTime
		score -= (rebufferRatio * 30)
	}

	// Penalize for high startup time (up to -20 points)
	if session.StartupTime > 5.0 {
		penalty := (session.StartupTime - 5.0) * 4 // 4 points per second over 5s
		if penalty > 20 {
			penalty = 20
		}
		score -= penalty
	}

	// Penalize for errors (-40 points)
	if session.ErrorOccurred {
		score -= 40
	}

	// Penalize for low completion rate (up to -10 points)
	if session.CompletionRate < 100 {
		score -= (100 - session.CompletionRate) * 0.1
	}

	// Ensure score is within bounds
	if score < 0 {
		score = 0
	}

	return score
}

// GetTrendingVideos returns trending videos based on recent engagement
func (s *Service) GetTrendingVideos(ctx context.Context, limit int) ([]*models.TrendingVideo, error) {
	return s.repo.GetTrendingVideos(ctx, limit)
}

// GenerateHeatmap creates engagement heatmap data for a video
func (s *Service) GenerateHeatmap(ctx context.Context, videoID string, resolution int) (*models.HeatmapData, error) {
	events, err := s.repo.GetVideoEvents(ctx, videoID)
	if err != nil {
		return nil, err
	}

	if len(events) == 0 {
		return &models.HeatmapData{
			VideoID:    videoID,
			Resolution: resolution,
			Data:       []models.HeatmapDataPoint{},
		}, nil
	}

	// Group events by time buckets
	buckets := make(map[int]*models.HeatmapDataPoint)

	for _, event := range events {
		bucket := int(event.Position) / resolution * resolution

		if buckets[bucket] == nil {
			buckets[bucket] = &models.HeatmapDataPoint{
				Timestamp: float64(bucket),
			}
		}

		// Count different event types
		if event.EventType == models.EventTypePlay {
			buckets[bucket].ViewCount++
		} else if event.EventType == models.EventTypeSeek {
			buckets[bucket].SeekCount++
		}
	}

	// Convert map to sorted slice
	var dataPoints []models.HeatmapDataPoint
	for _, point := range buckets {
		dataPoints = append(dataPoints, *point)
	}

	return &models.HeatmapData{
		VideoID:    videoID,
		Resolution: resolution,
		Data:       dataPoints,
	}, nil
}

// TrackBandwidth records bandwidth usage
func (s *Service) TrackBandwidth(ctx context.Context, usage *models.BandwidthUsage) error {
	if usage.ID == "" {
		usage.ID = uuid.New().String()
	}
	if usage.Timestamp.IsZero() {
		usage.Timestamp = time.Now()
	}

	return s.repo.CreateBandwidthUsage(ctx, usage)
}

// GetBandwidthUsage retrieves bandwidth usage statistics
func (s *Service) GetBandwidthUsage(ctx context.Context, videoID string, start, end time.Time) (int64, error) {
	return s.repo.GetBandwidthUsage(ctx, videoID, start, end)
}
