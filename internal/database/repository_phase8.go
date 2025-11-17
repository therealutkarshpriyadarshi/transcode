package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/therealutkarshpriyadarshi/transcode/pkg/models"
)

// CreateLiveStream creates a new live stream
func (r *Repository) CreateLiveStream(ctx context.Context, stream *models.LiveStream) error {
	if stream.ID == "" {
		stream.ID = uuid.New().String()
	}

	query := `
		INSERT INTO live_streams (
			id, title, description, user_id, stream_key, rtmp_ingest_url,
			status, dvr_enabled, dvr_window, low_latency, settings, metadata,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING created_at, updated_at
	`

	return r.db.DB.QueryRowContext(ctx, query,
		stream.ID, stream.Title, stream.Description, stream.UserID, stream.StreamKey,
		stream.RTMPIngestURL, stream.Status, stream.DVREnabled, stream.DVRWindow,
		stream.LowLatency, stream.Settings, stream.Metadata, time.Now(), time.Now(),
	).Scan(&stream.CreatedAt, &stream.UpdatedAt)
}

// GetLiveStream retrieves a live stream by ID
func (r *Repository) GetLiveStream(ctx context.Context, id string) (*models.LiveStream, error) {
	query := `
		SELECT id, title, description, user_id, stream_key, rtmp_ingest_url,
			status, master_playlist, viewer_count, peak_viewer_count,
			dvr_enabled, dvr_window, low_latency, settings, metadata,
			started_at, ended_at, created_at, updated_at
		FROM live_streams
		WHERE id = $1
	`

	stream := &models.LiveStream{}
	err := r.db.DB.QueryRowContext(ctx, query, id).Scan(
		&stream.ID, &stream.Title, &stream.Description, &stream.UserID,
		&stream.StreamKey, &stream.RTMPIngestURL, &stream.Status, &stream.MasterPlaylist,
		&stream.ViewerCount, &stream.PeakViewerCount, &stream.DVREnabled,
		&stream.DVRWindow, &stream.LowLatency, &stream.Settings, &stream.Metadata,
		&stream.StartedAt, &stream.EndedAt, &stream.CreatedAt, &stream.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("live stream not found")
	}

	return stream, err
}

// GetLiveStreamByKey retrieves a live stream by stream key
func (r *Repository) GetLiveStreamByKey(ctx context.Context, streamKey string) (*models.LiveStream, error) {
	query := `
		SELECT id, title, description, user_id, stream_key, rtmp_ingest_url,
			status, master_playlist, viewer_count, peak_viewer_count,
			dvr_enabled, dvr_window, low_latency, settings, metadata,
			started_at, ended_at, created_at, updated_at
		FROM live_streams
		WHERE stream_key = $1
	`

	stream := &models.LiveStream{}
	err := r.db.DB.QueryRowContext(ctx, query, streamKey).Scan(
		&stream.ID, &stream.Title, &stream.Description, &stream.UserID,
		&stream.StreamKey, &stream.RTMPIngestURL, &stream.Status, &stream.MasterPlaylist,
		&stream.ViewerCount, &stream.PeakViewerCount, &stream.DVREnabled,
		&stream.DVRWindow, &stream.LowLatency, &stream.Settings, &stream.Metadata,
		&stream.StartedAt, &stream.EndedAt, &stream.CreatedAt, &stream.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("live stream not found")
	}

	return stream, err
}

// ListLiveStreams lists live streams with optional filtering
func (r *Repository) ListLiveStreams(ctx context.Context, userID string, status string, limit, offset int) ([]*models.LiveStream, error) {
	query := `
		SELECT id, title, description, user_id, stream_key, rtmp_ingest_url,
			status, master_playlist, viewer_count, peak_viewer_count,
			dvr_enabled, dvr_window, low_latency, settings, metadata,
			started_at, ended_at, created_at, updated_at
		FROM live_streams
		WHERE ($1 = '' OR user_id = $1)
		AND ($2 = '' OR status = $2)
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4
	`

	rows, err := r.db.DB.QueryContext(ctx, query, userID, status, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	streams := []*models.LiveStream{}
	for rows.Next() {
		stream := &models.LiveStream{}
		if err := rows.Scan(
			&stream.ID, &stream.Title, &stream.Description, &stream.UserID,
			&stream.StreamKey, &stream.RTMPIngestURL, &stream.Status, &stream.MasterPlaylist,
			&stream.ViewerCount, &stream.PeakViewerCount, &stream.DVREnabled,
			&stream.DVRWindow, &stream.LowLatency, &stream.Settings, &stream.Metadata,
			&stream.StartedAt, &stream.EndedAt, &stream.CreatedAt, &stream.UpdatedAt,
		); err != nil {
			return nil, err
		}
		streams = append(streams, stream)
	}

	return streams, rows.Err()
}

// UpdateLiveStreamStatus updates the status of a live stream
func (r *Repository) UpdateLiveStreamStatus(ctx context.Context, id, status string) error {
	query := `UPDATE live_streams SET status = $1, updated_at = $2 WHERE id = $3`
	_, err := r.db.DB.ExecContext(ctx, query, status, time.Now(), id)
	return err
}

// UpdateLiveStreamStartTime updates the start time of a live stream
func (r *Repository) UpdateLiveStreamStartTime(ctx context.Context, id string, startTime *time.Time) error {
	query := `UPDATE live_streams SET started_at = $1, updated_at = $2 WHERE id = $3`
	_, err := r.db.DB.ExecContext(ctx, query, startTime, time.Now(), id)
	return err
}

// UpdateLiveStreamEndTime updates the end time of a live stream
func (r *Repository) UpdateLiveStreamEndTime(ctx context.Context, id string, endTime *time.Time) error {
	query := `UPDATE live_streams SET ended_at = $1, updated_at = $2 WHERE id = $3`
	_, err := r.db.DB.ExecContext(ctx, query, endTime, time.Now(), id)
	return err
}

// UpdateLiveStreamMasterPlaylist updates the master playlist for a live stream
func (r *Repository) UpdateLiveStreamMasterPlaylist(ctx context.Context, id, playlistURL string) error {
	query := `UPDATE live_streams SET master_playlist = $1, updated_at = $2 WHERE id = $3`
	_, err := r.db.DB.ExecContext(ctx, query, playlistURL, time.Now(), id)
	return err
}

// UpdateLiveStreamViewerCount updates the viewer count for a live stream
func (r *Repository) UpdateLiveStreamViewerCount(ctx context.Context, id string, count int) error {
	query := `
		UPDATE live_streams
		SET viewer_count = $1,
			peak_viewer_count = GREATEST(peak_viewer_count, $1),
			updated_at = $2
		WHERE id = $3
	`
	_, err := r.db.DB.ExecContext(ctx, query, count, time.Now(), id)
	return err
}

// DeleteLiveStream deletes a live stream
func (r *Repository) DeleteLiveStream(ctx context.Context, id string) error {
	query := `DELETE FROM live_streams WHERE id = $1`
	_, err := r.db.DB.ExecContext(ctx, query, id)
	return err
}

// CreateLiveStreamVariant creates a new live stream variant
func (r *Repository) CreateLiveStreamVariant(ctx context.Context, variant *models.LiveStreamVariant) error {
	if variant.ID == "" {
		variant.ID = uuid.New().String()
	}

	query := `
		INSERT INTO live_stream_variants (
			id, live_stream_id, resolution, width, height, bitrate,
			frame_rate, codec, audio_bitrate, playlist_url, segment_pattern,
			created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING created_at
	`

	return r.db.DB.QueryRowContext(ctx, query,
		variant.ID, variant.LiveStreamID, variant.Resolution, variant.Width,
		variant.Height, variant.Bitrate, variant.FrameRate, variant.Codec,
		variant.AudioBitrate, variant.PlaylistURL, variant.SegmentPattern, time.Now(),
	).Scan(&variant.CreatedAt)
}

// GetLiveStreamVariants retrieves all variants for a live stream
func (r *Repository) GetLiveStreamVariants(ctx context.Context, streamID string) ([]*models.LiveStreamVariant, error) {
	query := `
		SELECT id, live_stream_id, resolution, width, height, bitrate,
			frame_rate, codec, audio_bitrate, playlist_url, segment_pattern, created_at
		FROM live_stream_variants
		WHERE live_stream_id = $1
		ORDER BY bitrate DESC
	`

	rows, err := r.db.DB.QueryContext(ctx, query, streamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	variants := []*models.LiveStreamVariant{}
	for rows.Next() {
		variant := &models.LiveStreamVariant{}
		if err := rows.Scan(
			&variant.ID, &variant.LiveStreamID, &variant.Resolution, &variant.Width,
			&variant.Height, &variant.Bitrate, &variant.FrameRate, &variant.Codec,
			&variant.AudioBitrate, &variant.PlaylistURL, &variant.SegmentPattern,
			&variant.CreatedAt,
		); err != nil {
			return nil, err
		}
		variants = append(variants, variant)
	}

	return variants, rows.Err()
}

// CreateDVRRecording creates a new DVR recording
func (r *Repository) CreateDVRRecording(ctx context.Context, recording *models.DVRRecording) error {
	if recording.ID == "" {
		recording.ID = uuid.New().String()
	}

	query := `
		INSERT INTO dvr_recordings (
			id, live_stream_id, video_id, start_time, end_time, duration,
			size, status, recording_url, manifest_url, thumbnail_url,
			retention_until, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING created_at, updated_at
	`

	return r.db.DB.QueryRowContext(ctx, query,
		recording.ID, recording.LiveStreamID, recording.VideoID, recording.StartTime,
		recording.EndTime, recording.Duration, recording.Size, recording.Status,
		recording.RecordingURL, recording.ManifestURL, recording.ThumbnailURL,
		recording.RetentionUntil, time.Now(), time.Now(),
	).Scan(&recording.CreatedAt, &recording.UpdatedAt)
}

// GetDVRRecording retrieves a DVR recording by ID
func (r *Repository) GetDVRRecording(ctx context.Context, id string) (*models.DVRRecording, error) {
	query := `
		SELECT id, live_stream_id, video_id, start_time, end_time, duration,
			size, status, recording_url, manifest_url, thumbnail_url,
			retention_until, created_at, updated_at
		FROM dvr_recordings
		WHERE id = $1
	`

	recording := &models.DVRRecording{}
	err := r.db.DB.QueryRowContext(ctx, query, id).Scan(
		&recording.ID, &recording.LiveStreamID, &recording.VideoID, &recording.StartTime,
		&recording.EndTime, &recording.Duration, &recording.Size, &recording.Status,
		&recording.RecordingURL, &recording.ManifestURL, &recording.ThumbnailURL,
		&recording.RetentionUntil, &recording.CreatedAt, &recording.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("DVR recording not found")
	}

	return recording, err
}

// ListDVRRecordings lists DVR recordings for a live stream
func (r *Repository) ListDVRRecordings(ctx context.Context, streamID string) ([]*models.DVRRecording, error) {
	query := `
		SELECT id, live_stream_id, video_id, start_time, end_time, duration,
			size, status, recording_url, manifest_url, thumbnail_url,
			retention_until, created_at, updated_at
		FROM dvr_recordings
		WHERE live_stream_id = $1
		ORDER BY start_time DESC
	`

	rows, err := r.db.DB.QueryContext(ctx, query, streamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	recordings := []*models.DVRRecording{}
	for rows.Next() {
		recording := &models.DVRRecording{}
		if err := rows.Scan(
			&recording.ID, &recording.LiveStreamID, &recording.VideoID, &recording.StartTime,
			&recording.EndTime, &recording.Duration, &recording.Size, &recording.Status,
			&recording.RecordingURL, &recording.ManifestURL, &recording.ThumbnailURL,
			&recording.RetentionUntil, &recording.CreatedAt, &recording.UpdatedAt,
		); err != nil {
			return nil, err
		}
		recordings = append(recordings, recording)
	}

	return recordings, rows.Err()
}

// UpdateDVRRecordingStatus updates the status of a DVR recording
func (r *Repository) UpdateDVRRecordingStatus(ctx context.Context, id, status string) error {
	query := `UPDATE dvr_recordings SET status = $1, updated_at = $2 WHERE id = $3`
	_, err := r.db.DB.ExecContext(ctx, query, status, time.Now(), id)
	return err
}

// CreateLiveStreamAnalytics creates a new analytics record
func (r *Repository) CreateLiveStreamAnalytics(ctx context.Context, analytics *models.LiveStreamAnalytics) error {
	if analytics.ID == "" {
		analytics.ID = uuid.New().String()
	}

	query := `
		INSERT INTO live_stream_analytics (
			id, live_stream_id, timestamp, viewer_count, bandwidth_usage,
			ingest_bitrate, dropped_frames, keyframe_interval, audio_video_sync,
			buffer_health, cdn_hit_ratio, average_latency, p95_latency,
			error_count, quality_score
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`

	_, err := r.db.DB.ExecContext(ctx, query,
		analytics.ID, analytics.LiveStreamID, analytics.Timestamp, analytics.ViewerCount,
		analytics.BandwidthUsage, analytics.IngestBitrate, analytics.DroppedFrames,
		analytics.KeyframeInterval, analytics.AudioVideoSync, analytics.BufferHealth,
		analytics.CDNHitRatio, analytics.AverageLatency, analytics.P95Latency,
		analytics.ErrorCount, analytics.QualityScore,
	)

	return err
}

// GetLiveStreamAnalytics retrieves analytics for a time range
func (r *Repository) GetLiveStreamAnalytics(ctx context.Context, streamID string, from, to time.Time) ([]*models.LiveStreamAnalytics, error) {
	query := `
		SELECT id, live_stream_id, timestamp, viewer_count, bandwidth_usage,
			ingest_bitrate, dropped_frames, keyframe_interval, audio_video_sync,
			buffer_health, cdn_hit_ratio, average_latency, p95_latency,
			error_count, quality_score
		FROM live_stream_analytics
		WHERE live_stream_id = $1
		AND timestamp BETWEEN $2 AND $3
		ORDER BY timestamp ASC
	`

	rows, err := r.db.DB.QueryContext(ctx, query, streamID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	analytics := []*models.LiveStreamAnalytics{}
	for rows.Next() {
		a := &models.LiveStreamAnalytics{}
		if err := rows.Scan(
			&a.ID, &a.LiveStreamID, &a.Timestamp, &a.ViewerCount, &a.BandwidthUsage,
			&a.IngestBitrate, &a.DroppedFrames, &a.KeyframeInterval, &a.AudioVideoSync,
			&a.BufferHealth, &a.CDNHitRatio, &a.AverageLatency, &a.P95Latency,
			&a.ErrorCount, &a.QualityScore,
		); err != nil {
			return nil, err
		}
		analytics = append(analytics, a)
	}

	return analytics, rows.Err()
}

// CreateLiveStreamEvent creates a new event record
func (r *Repository) CreateLiveStreamEvent(ctx context.Context, event *models.LiveStreamEvent) error {
	if event.ID == "" {
		event.ID = uuid.New().String()
	}

	query := `
		INSERT INTO live_stream_events (
			id, live_stream_id, event_type, severity, message, details, timestamp
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := r.db.DB.ExecContext(ctx, query,
		event.ID, event.LiveStreamID, event.EventType, event.Severity,
		event.Message, event.Details, event.Timestamp,
	)

	return err
}

// GetLiveStreamEvents retrieves events for a live stream
func (r *Repository) GetLiveStreamEvents(ctx context.Context, streamID string, limit int) ([]*models.LiveStreamEvent, error) {
	query := `
		SELECT id, live_stream_id, event_type, severity, message, details, timestamp
		FROM live_stream_events
		WHERE live_stream_id = $1
		ORDER BY timestamp DESC
		LIMIT $2
	`

	rows, err := r.db.DB.QueryContext(ctx, query, streamID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := []*models.LiveStreamEvent{}
	for rows.Next() {
		event := &models.LiveStreamEvent{}
		if err := rows.Scan(
			&event.ID, &event.LiveStreamID, &event.EventType, &event.Severity,
			&event.Message, &event.Details, &event.Timestamp,
		); err != nil {
			return nil, err
		}
		events = append(events, event)
	}

	return events, rows.Err()
}

// TrackViewer creates or updates a viewer session
func (r *Repository) TrackViewer(ctx context.Context, viewer *models.LiveStreamViewer) error {
	if viewer.ID == "" {
		viewer.ID = uuid.New().String()
	}

	query := `
		INSERT INTO live_stream_viewers (
			id, live_stream_id, session_id, user_id, joined_at, left_at,
			watch_duration, resolution, device_type, location, ip_address,
			user_agent, buffer_events, quality_changes
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		ON CONFLICT (id) DO UPDATE SET
			left_at = EXCLUDED.left_at,
			watch_duration = EXCLUDED.watch_duration,
			buffer_events = EXCLUDED.buffer_events,
			quality_changes = EXCLUDED.quality_changes
	`

	_, err := r.db.DB.ExecContext(ctx, query,
		viewer.ID, viewer.LiveStreamID, viewer.SessionID, viewer.UserID,
		viewer.JoinedAt, viewer.LeftAt, viewer.WatchDuration, viewer.Resolution,
		viewer.DeviceType, viewer.Location, viewer.IPAddress, viewer.UserAgent,
		viewer.BufferEvents, viewer.QualityChanges,
	)

	return err
}

// GetActiveViewers retrieves currently active viewers for a live stream
func (r *Repository) GetActiveViewers(ctx context.Context, streamID string) ([]*models.LiveStreamViewer, error) {
	query := `
		SELECT id, live_stream_id, session_id, user_id, joined_at, left_at,
			watch_duration, resolution, device_type, location, ip_address,
			user_agent, buffer_events, quality_changes
		FROM live_stream_viewers
		WHERE live_stream_id = $1 AND left_at IS NULL
		ORDER BY joined_at DESC
	`

	rows, err := r.db.DB.QueryContext(ctx, query, streamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	viewers := []*models.LiveStreamViewer{}
	for rows.Next() {
		viewer := &models.LiveStreamViewer{}
		if err := rows.Scan(
			&viewer.ID, &viewer.LiveStreamID, &viewer.SessionID, &viewer.UserID,
			&viewer.JoinedAt, &viewer.LeftAt, &viewer.WatchDuration, &viewer.Resolution,
			&viewer.DeviceType, &viewer.Location, &viewer.IPAddress, &viewer.UserAgent,
			&viewer.BufferEvents, &viewer.QualityChanges,
		); err != nil {
			return nil, err
		}
		viewers = append(viewers, viewer)
	}

	return viewers, rows.Err()
}
