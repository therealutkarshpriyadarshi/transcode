-- Phase 8: Live Streaming Support Migration

-- Live Streams Table
CREATE TABLE IF NOT EXISTS live_streams (
    id VARCHAR(36) PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    user_id VARCHAR(100) NOT NULL,
    stream_key VARCHAR(100) UNIQUE NOT NULL,
    rtmp_ingest_url VARCHAR(255) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'idle',
    master_playlist TEXT,
    viewer_count INTEGER DEFAULT 0,
    peak_viewer_count INTEGER DEFAULT 0,
    dvr_enabled BOOLEAN DEFAULT false,
    dvr_window INTEGER DEFAULT 7200, -- 2 hours default DVR window
    low_latency BOOLEAN DEFAULT false,
    settings JSONB NOT NULL DEFAULT '{}',
    metadata JSONB DEFAULT '{}',
    started_at TIMESTAMP WITH TIME ZONE,
    ended_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for live streams
CREATE INDEX IF NOT EXISTS idx_live_streams_user_id ON live_streams(user_id);
CREATE INDEX IF NOT EXISTS idx_live_streams_status ON live_streams(status);
CREATE INDEX IF NOT EXISTS idx_live_streams_stream_key ON live_streams(stream_key);
CREATE INDEX IF NOT EXISTS idx_live_streams_created_at ON live_streams(created_at);

-- Live Stream Variants Table (for multi-bitrate streaming)
CREATE TABLE IF NOT EXISTS live_stream_variants (
    id VARCHAR(36) PRIMARY KEY,
    live_stream_id VARCHAR(36) NOT NULL REFERENCES live_streams(id) ON DELETE CASCADE,
    resolution VARCHAR(20) NOT NULL,
    width INTEGER NOT NULL,
    height INTEGER NOT NULL,
    bitrate BIGINT NOT NULL,
    frame_rate DOUBLE PRECISION NOT NULL DEFAULT 30.0,
    codec VARCHAR(20) NOT NULL,
    audio_bitrate INTEGER NOT NULL,
    playlist_url TEXT NOT NULL,
    segment_pattern VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for live stream variants
CREATE INDEX IF NOT EXISTS idx_live_stream_variants_stream_id ON live_stream_variants(live_stream_id);
CREATE INDEX IF NOT EXISTS idx_live_stream_variants_resolution ON live_stream_variants(resolution);

-- DVR Recordings Table
CREATE TABLE IF NOT EXISTS dvr_recordings (
    id VARCHAR(36) PRIMARY KEY,
    live_stream_id VARCHAR(36) NOT NULL REFERENCES live_streams(id) ON DELETE CASCADE,
    video_id VARCHAR(36) REFERENCES videos(id) ON DELETE SET NULL,
    start_time TIMESTAMP WITH TIME ZONE NOT NULL,
    end_time TIMESTAMP WITH TIME ZONE,
    duration DOUBLE PRECISION DEFAULT 0,
    size BIGINT DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'recording',
    recording_url TEXT,
    manifest_url TEXT,
    thumbnail_url TEXT,
    retention_until TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for DVR recordings
CREATE INDEX IF NOT EXISTS idx_dvr_recordings_stream_id ON dvr_recordings(live_stream_id);
CREATE INDEX IF NOT EXISTS idx_dvr_recordings_status ON dvr_recordings(status);
CREATE INDEX IF NOT EXISTS idx_dvr_recordings_start_time ON dvr_recordings(start_time);
CREATE INDEX IF NOT EXISTS idx_dvr_recordings_retention_until ON dvr_recordings(retention_until);

-- Live Stream Analytics Table (real-time metrics)
CREATE TABLE IF NOT EXISTS live_stream_analytics (
    id VARCHAR(36) PRIMARY KEY,
    live_stream_id VARCHAR(36) NOT NULL REFERENCES live_streams(id) ON DELETE CASCADE,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    viewer_count INTEGER DEFAULT 0,
    bandwidth_usage BIGINT DEFAULT 0,
    ingest_bitrate BIGINT DEFAULT 0,
    dropped_frames INTEGER DEFAULT 0,
    keyframe_interval DOUBLE PRECISION DEFAULT 0,
    audio_video_sync DOUBLE PRECISION DEFAULT 0,
    buffer_health DOUBLE PRECISION DEFAULT 100,
    cdn_hit_ratio DOUBLE PRECISION DEFAULT 0,
    average_latency DOUBLE PRECISION DEFAULT 0,
    p95_latency DOUBLE PRECISION DEFAULT 0,
    error_count INTEGER DEFAULT 0,
    quality_score DOUBLE PRECISION DEFAULT 0
);

-- Create indexes for analytics (time-series data)
CREATE INDEX IF NOT EXISTS idx_live_stream_analytics_stream_id ON live_stream_analytics(live_stream_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_live_stream_analytics_timestamp ON live_stream_analytics(timestamp DESC);

-- Live Stream Events Table
CREATE TABLE IF NOT EXISTS live_stream_events (
    id VARCHAR(36) PRIMARY KEY,
    live_stream_id VARCHAR(36) NOT NULL REFERENCES live_streams(id) ON DELETE CASCADE,
    event_type VARCHAR(50) NOT NULL,
    severity VARCHAR(20) NOT NULL,
    message TEXT NOT NULL,
    details JSONB DEFAULT '{}',
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for events
CREATE INDEX IF NOT EXISTS idx_live_stream_events_stream_id ON live_stream_events(live_stream_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_live_stream_events_event_type ON live_stream_events(event_type);
CREATE INDEX IF NOT EXISTS idx_live_stream_events_severity ON live_stream_events(severity);
CREATE INDEX IF NOT EXISTS idx_live_stream_events_timestamp ON live_stream_events(timestamp DESC);

-- Live Stream Viewers Table (viewer sessions)
CREATE TABLE IF NOT EXISTS live_stream_viewers (
    id VARCHAR(36) PRIMARY KEY,
    live_stream_id VARCHAR(36) NOT NULL REFERENCES live_streams(id) ON DELETE CASCADE,
    session_id VARCHAR(100) NOT NULL,
    user_id VARCHAR(100),
    joined_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    left_at TIMESTAMP WITH TIME ZONE,
    watch_duration DOUBLE PRECISION DEFAULT 0,
    resolution VARCHAR(20),
    device_type VARCHAR(50),
    location VARCHAR(100),
    ip_address INET,
    user_agent TEXT,
    buffer_events INTEGER DEFAULT 0,
    quality_changes INTEGER DEFAULT 0
);

-- Create indexes for viewers
CREATE INDEX IF NOT EXISTS idx_live_stream_viewers_stream_id ON live_stream_viewers(live_stream_id);
CREATE INDEX IF NOT EXISTS idx_live_stream_viewers_session_id ON live_stream_viewers(session_id);
CREATE INDEX IF NOT EXISTS idx_live_stream_viewers_user_id ON live_stream_viewers(user_id);
CREATE INDEX IF NOT EXISTS idx_live_stream_viewers_joined_at ON live_stream_viewers(joined_at);

-- Create trigger to update updated_at timestamp for live streams
CREATE OR REPLACE FUNCTION update_live_streams_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_live_streams_updated_at
BEFORE UPDATE ON live_streams
FOR EACH ROW
EXECUTE FUNCTION update_live_streams_updated_at();

-- Create trigger to update updated_at timestamp for DVR recordings
CREATE TRIGGER trigger_update_dvr_recordings_updated_at
BEFORE UPDATE ON dvr_recordings
FOR EACH ROW
EXECUTE FUNCTION update_live_streams_updated_at();

-- Create function to clean up old analytics data (retention policy)
CREATE OR REPLACE FUNCTION cleanup_old_live_stream_analytics()
RETURNS void AS $$
BEGIN
    -- Delete analytics data older than 30 days
    DELETE FROM live_stream_analytics
    WHERE timestamp < CURRENT_TIMESTAMP - INTERVAL '30 days';

    -- Delete events older than 90 days
    DELETE FROM live_stream_events
    WHERE timestamp < CURRENT_TIMESTAMP - INTERVAL '90 days';
END;
$$ LANGUAGE plpgsql;

-- Create function to automatically archive expired DVR recordings
CREATE OR REPLACE FUNCTION archive_expired_dvr_recordings()
RETURNS void AS $$
BEGIN
    UPDATE dvr_recordings
    SET status = 'archived'
    WHERE status IN ('available', 'processing')
    AND retention_until IS NOT NULL
    AND retention_until < CURRENT_TIMESTAMP;
END;
$$ LANGUAGE plpgsql;
