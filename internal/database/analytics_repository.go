package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/therealutkarshpriyadarshi/transcode/pkg/models"
)

// Analytics Repository Methods

// CreatePlaybackEvent records a playback event
func (r *Repository) CreatePlaybackEvent(ctx context.Context, event *models.PlaybackEvent) error {
	query := `
		INSERT INTO playback_events (
			id, video_id, output_id, session_id, user_id, event_type, timestamp,
			position, duration, buffer_time, bitrate, resolution, device_type,
			browser, os, country, ip_address, cdn_node, error_code, error_message
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)
	`

	_, err := r.db.Pool.Exec(ctx, query,
		event.ID, event.VideoID, event.OutputID, event.SessionID, event.UserID,
		event.EventType, event.Timestamp, event.Position, event.Duration,
		event.BufferTime, event.Bitrate, event.Resolution, event.DeviceType,
		event.Browser, event.OS, event.Country, event.IPAddress, event.CDNNode,
		event.ErrorCode, event.ErrorMessage,
	)

	if err != nil {
		return fmt.Errorf("failed to create playback event: %w", err)
	}

	return nil
}

// CreatePlaybackSession creates a new playback session
func (r *Repository) CreatePlaybackSession(ctx context.Context, session *models.PlaybackSession) error {
	query := `
		INSERT INTO playback_sessions (
			id, video_id, user_id, start_time, device_type, browser, os, country
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := r.db.Pool.Exec(ctx, query,
		session.ID, session.VideoID, session.UserID, session.StartTime,
		session.DeviceType, session.Browser, session.OS, session.Country,
	)

	if err != nil {
		return fmt.Errorf("failed to create playback session: %w", err)
	}

	return nil
}

// GetPlaybackSession retrieves a playback session by ID
func (r *Repository) GetPlaybackSession(ctx context.Context, sessionID string) (*models.PlaybackSession, error) {
	var session models.PlaybackSession

	query := `
		SELECT id, video_id, user_id, start_time, end_time, duration, watch_time,
		       completion_rate, total_buffer_time, buffer_count, seek_count,
		       quality_changes, average_bitrate, peak_bitrate, startup_time,
		       device_type, browser, os, country, completed, error_occurred
		FROM playback_sessions
		WHERE id = $1
	`

	err := r.db.Pool.QueryRow(ctx, query, sessionID).Scan(
		&session.ID, &session.VideoID, &session.UserID, &session.StartTime,
		&session.EndTime, &session.Duration, &session.WatchTime,
		&session.CompletionRate, &session.TotalBufferTime, &session.BufferCount,
		&session.SeekCount, &session.QualityChanges, &session.AverageBitrate,
		&session.PeakBitrate, &session.StartupTime, &session.DeviceType,
		&session.Browser, &session.OS, &session.Country, &session.Completed,
		&session.ErrorOccurred,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("session not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get playback session: %w", err)
	}

	return &session, nil
}

// UpdatePlaybackSession updates a playback session
func (r *Repository) UpdatePlaybackSession(ctx context.Context, session *models.PlaybackSession) error {
	query := `
		UPDATE playback_sessions
		SET end_time = $2, duration = $3, watch_time = $4, completion_rate = $5,
		    total_buffer_time = $6, buffer_count = $7, seek_count = $8,
		    quality_changes = $9, average_bitrate = $10, peak_bitrate = $11,
		    startup_time = $12, completed = $13, error_occurred = $14
		WHERE id = $1
	`

	_, err := r.db.Pool.Exec(ctx, query,
		session.ID, session.EndTime, session.Duration, session.WatchTime,
		session.CompletionRate, session.TotalBufferTime, session.BufferCount,
		session.SeekCount, session.QualityChanges, session.AverageBitrate,
		session.PeakBitrate, session.StartupTime, session.Completed,
		session.ErrorOccurred,
	)

	if err != nil {
		return fmt.Errorf("failed to update playback session: %w", err)
	}

	return nil
}

// GetSessionEvents retrieves all events for a session
func (r *Repository) GetSessionEvents(ctx context.Context, sessionID string) ([]*models.PlaybackEvent, error) {
	query := `
		SELECT id, video_id, output_id, session_id, user_id, event_type, timestamp,
		       position, duration, buffer_time, bitrate, resolution, device_type,
		       browser, os, country, ip_address, cdn_node, error_code, error_message
		FROM playback_events
		WHERE session_id = $1
		ORDER BY timestamp ASC
	`

	rows, err := r.db.Pool.Query(ctx, query, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to query session events: %w", err)
	}
	defer rows.Close()

	var events []*models.PlaybackEvent
	for rows.Next() {
		var event models.PlaybackEvent
		err := rows.Scan(
			&event.ID, &event.VideoID, &event.OutputID, &event.SessionID,
			&event.UserID, &event.EventType, &event.Timestamp, &event.Position,
			&event.Duration, &event.BufferTime, &event.Bitrate, &event.Resolution,
			&event.DeviceType, &event.Browser, &event.OS, &event.Country,
			&event.IPAddress, &event.CDNNode, &event.ErrorCode, &event.ErrorMessage,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}
		events = append(events, &event)
	}

	return events, nil
}

// GetVideoSessions retrieves all sessions for a video
func (r *Repository) GetVideoSessions(ctx context.Context, videoID string) ([]*models.PlaybackSession, error) {
	query := `
		SELECT id, video_id, user_id, start_time, end_time, duration, watch_time,
		       completion_rate, total_buffer_time, buffer_count, seek_count,
		       quality_changes, average_bitrate, peak_bitrate, startup_time,
		       device_type, browser, os, country, completed, error_occurred
		FROM playback_sessions
		WHERE video_id = $1
		ORDER BY start_time DESC
	`

	rows, err := r.db.Pool.Query(ctx, query, videoID)
	if err != nil {
		return nil, fmt.Errorf("failed to query video sessions: %w", err)
	}
	defer rows.Close()

	var sessions []*models.PlaybackSession
	for rows.Next() {
		var session models.PlaybackSession
		err := rows.Scan(
			&session.ID, &session.VideoID, &session.UserID, &session.StartTime,
			&session.EndTime, &session.Duration, &session.WatchTime,
			&session.CompletionRate, &session.TotalBufferTime, &session.BufferCount,
			&session.SeekCount, &session.QualityChanges, &session.AverageBitrate,
			&session.PeakBitrate, &session.StartupTime, &session.DeviceType,
			&session.Browser, &session.OS, &session.Country, &session.Completed,
			&session.ErrorOccurred,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan session: %w", err)
		}
		sessions = append(sessions, &session)
	}

	return sessions, nil
}

// GetVideoEvents retrieves all events for a video
func (r *Repository) GetVideoEvents(ctx context.Context, videoID string) ([]*models.PlaybackEvent, error) {
	query := `
		SELECT id, video_id, output_id, session_id, user_id, event_type, timestamp,
		       position, duration, buffer_time, bitrate, resolution, device_type,
		       browser, os, country, ip_address, cdn_node, error_code, error_message
		FROM playback_events
		WHERE video_id = $1
		ORDER BY timestamp ASC
	`

	rows, err := r.db.Pool.Query(ctx, query, videoID)
	if err != nil {
		return nil, fmt.Errorf("failed to query video events: %w", err)
	}
	defer rows.Close()

	var events []*models.PlaybackEvent
	for rows.Next() {
		var event models.PlaybackEvent
		err := rows.Scan(
			&event.ID, &event.VideoID, &event.OutputID, &event.SessionID,
			&event.UserID, &event.EventType, &event.Timestamp, &event.Position,
			&event.Duration, &event.BufferTime, &event.Bitrate, &event.Resolution,
			&event.DeviceType, &event.Browser, &event.OS, &event.Country,
			&event.IPAddress, &event.CDNNode, &event.ErrorCode, &event.ErrorMessage,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}
		events = append(events, &event)
	}

	return events, nil
}

// UpsertVideoAnalytics creates or updates video analytics
func (r *Repository) UpsertVideoAnalytics(ctx context.Context, analytics *models.VideoAnalytics) error {
	query := `
		INSERT INTO video_analytics (
			video_id, total_views, unique_viewers, total_watch_time,
			average_watch_time, completion_rate, average_buffer_time,
			buffer_rate, error_rate, average_startup_time, last_updated
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (video_id) DO UPDATE SET
			total_views = EXCLUDED.total_views,
			unique_viewers = EXCLUDED.unique_viewers,
			total_watch_time = EXCLUDED.total_watch_time,
			average_watch_time = EXCLUDED.average_watch_time,
			completion_rate = EXCLUDED.completion_rate,
			average_buffer_time = EXCLUDED.average_buffer_time,
			buffer_rate = EXCLUDED.buffer_rate,
			error_rate = EXCLUDED.error_rate,
			average_startup_time = EXCLUDED.average_startup_time,
			last_updated = EXCLUDED.last_updated
	`

	_, err := r.db.Pool.Exec(ctx, query,
		analytics.VideoID, analytics.TotalViews, analytics.UniqueViewers,
		analytics.TotalWatchTime, analytics.AverageWatchTime, analytics.CompletionRate,
		analytics.AverageBufferTime, analytics.BufferRate, analytics.ErrorRate,
		analytics.AverageStartupTime, analytics.LastUpdated,
	)

	if err != nil {
		return fmt.Errorf("failed to upsert video analytics: %w", err)
	}

	return nil
}

// GetVideoAnalytics retrieves analytics for a video
func (r *Repository) GetVideoAnalytics(ctx context.Context, videoID string) (*models.VideoAnalytics, error) {
	var analytics models.VideoAnalytics

	query := `
		SELECT video_id, total_views, unique_viewers, total_watch_time,
		       average_watch_time, completion_rate, average_buffer_time,
		       buffer_rate, error_rate, average_startup_time, last_updated
		FROM video_analytics
		WHERE video_id = $1
	`

	err := r.db.Pool.QueryRow(ctx, query, videoID).Scan(
		&analytics.VideoID, &analytics.TotalViews, &analytics.UniqueViewers,
		&analytics.TotalWatchTime, &analytics.AverageWatchTime, &analytics.CompletionRate,
		&analytics.AverageBufferTime, &analytics.BufferRate, &analytics.ErrorRate,
		&analytics.AverageStartupTime, &analytics.LastUpdated,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("analytics not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get video analytics: %w", err)
	}

	// Initialize maps
	analytics.PopularResolutions = make(map[string]int64)
	analytics.GeographicData = make(map[string]int64)
	analytics.DeviceBreakdown = make(map[string]int64)

	return &analytics, nil
}

// GetQoEMetrics retrieves QoE metrics
func (r *Repository) GetQoEMetrics(ctx context.Context, videoID string, period string, start, end time.Time) ([]*models.QoEMetrics, error) {
	query := `
		SELECT video_id, output_id, period, timestamp, view_count, average_qoe,
		       rebuffer_ratio, startup_time, bitrate_utilization, error_rate, completion_rate
		FROM qoe_metrics
		WHERE video_id = $1 AND period = $2 AND timestamp BETWEEN $3 AND $4
		ORDER BY timestamp ASC
	`

	rows, err := r.db.Pool.Query(ctx, query, videoID, period, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to query QoE metrics: %w", err)
	}
	defer rows.Close()

	var metrics []*models.QoEMetrics
	for rows.Next() {
		var metric models.QoEMetrics
		err := rows.Scan(
			&metric.VideoID, &metric.OutputID, &metric.Period, &metric.Timestamp,
			&metric.ViewCount, &metric.AverageQoE, &metric.RebufferRatio,
			&metric.StartupTime, &metric.BitrateUtilization, &metric.ErrorRate,
			&metric.CompletionRate,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan QoE metric: %w", err)
		}
		metrics = append(metrics, &metric)
	}

	return metrics, nil
}

// CreateBandwidthUsage records bandwidth usage
func (r *Repository) CreateBandwidthUsage(ctx context.Context, usage *models.BandwidthUsage) error {
	query := `
		INSERT INTO bandwidth_usage (
			id, video_id, output_id, timestamp, bytes_served, request_count, cdn_node, country
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := r.db.Pool.Exec(ctx, query,
		usage.ID, usage.VideoID, usage.OutputID, usage.Timestamp,
		usage.BytesServed, usage.RequestCount, usage.CDNNode, usage.Country,
	)

	if err != nil {
		return fmt.Errorf("failed to create bandwidth usage: %w", err)
	}

	return nil
}

// GetBandwidthUsage retrieves total bandwidth usage for a video in a time range
func (r *Repository) GetBandwidthUsage(ctx context.Context, videoID string, start, end time.Time) (int64, error) {
	var totalBytes int64

	query := `
		SELECT COALESCE(SUM(bytes_served), 0)
		FROM bandwidth_usage
		WHERE video_id = $1 AND timestamp BETWEEN $2 AND $3
	`

	err := r.db.Pool.QueryRow(ctx, query, videoID, start, end).Scan(&totalBytes)
	if err != nil {
		return 0, fmt.Errorf("failed to get bandwidth usage: %w", err)
	}

	return totalBytes, nil
}

// GetTrendingVideos retrieves trending videos
func (r *Repository) GetTrendingVideos(ctx context.Context, limit int) ([]*models.TrendingVideo, error) {
	// Calculate trending videos based on recent view growth
	query := `
		WITH recent_views AS (
			SELECT video_id, COUNT(*) as views
			FROM playback_sessions
			WHERE start_time > NOW() - INTERVAL '24 hours'
			GROUP BY video_id
		),
		previous_views AS (
			SELECT video_id, COUNT(*) as views
			FROM playback_sessions
			WHERE start_time BETWEEN NOW() - INTERVAL '48 hours' AND NOW() - INTERVAL '24 hours'
			GROUP BY video_id
		)
		SELECT
			v.id as video_id,
			v.filename as title,
			COALESCE(r.views, 0) as views,
			CASE
				WHEN COALESCE(p.views, 0) = 0 THEN 100.0
				ELSE ((COALESCE(r.views, 0)::float - COALESCE(p.views, 0)::float) / COALESCE(p.views, 0)::float * 100)
			END as view_growth,
			COALESCE(r.views, 0) as trending_score,
			NOW() as last_updated
		FROM videos v
		LEFT JOIN recent_views r ON v.id = r.video_id
		LEFT JOIN previous_views p ON v.id = p.video_id
		WHERE COALESCE(r.views, 0) > 0
		ORDER BY trending_score DESC, view_growth DESC
		LIMIT $1
	`

	rows, err := r.db.Pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query trending videos: %w", err)
	}
	defer rows.Close()

	var trending []*models.TrendingVideo
	for rows.Next() {
		var video models.TrendingVideo
		err := rows.Scan(
			&video.VideoID, &video.Title, &video.Views, &video.ViewGrowth,
			&video.TrendingScore, &video.LastUpdated,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan trending video: %w", err)
		}
		trending = append(trending, &video)
	}

	return trending, nil
}
