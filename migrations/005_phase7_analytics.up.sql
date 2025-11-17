-- Phase 7: Analytics and Advanced Features Migration

-- Playback Events Table
CREATE TABLE IF NOT EXISTS playback_events (
    id VARCHAR(36) PRIMARY KEY,
    video_id VARCHAR(36) NOT NULL REFERENCES videos(id) ON DELETE CASCADE,
    output_id VARCHAR(36),
    session_id VARCHAR(36) NOT NULL,
    user_id VARCHAR(100),
    event_type VARCHAR(20) NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    position DOUBLE PRECISION DEFAULT 0,
    duration DOUBLE PRECISION DEFAULT 0,
    buffer_time DOUBLE PRECISION DEFAULT 0,
    bitrate BIGINT DEFAULT 0,
    resolution VARCHAR(20),
    device_type VARCHAR(20),
    browser VARCHAR(100),
    os VARCHAR(100),
    country VARCHAR(100),
    ip_address INET,
    cdn_node VARCHAR(100),
    error_code VARCHAR(50),
    error_message TEXT
);

-- Create indexes for playback events
CREATE INDEX IF NOT EXISTS idx_playback_events_video_id ON playback_events(video_id);
CREATE INDEX IF NOT EXISTS idx_playback_events_session_id ON playback_events(session_id);
CREATE INDEX IF NOT EXISTS idx_playback_events_timestamp ON playback_events(timestamp);
CREATE INDEX IF NOT EXISTS idx_playback_events_event_type ON playback_events(event_type);

-- Playback Sessions Table
CREATE TABLE IF NOT EXISTS playback_sessions (
    id VARCHAR(36) PRIMARY KEY,
    video_id VARCHAR(36) NOT NULL REFERENCES videos(id) ON DELETE CASCADE,
    user_id VARCHAR(100),
    start_time TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    end_time TIMESTAMP WITH TIME ZONE,
    duration DOUBLE PRECISION DEFAULT 0,
    watch_time DOUBLE PRECISION DEFAULT 0,
    completion_rate DOUBLE PRECISION DEFAULT 0,
    total_buffer_time DOUBLE PRECISION DEFAULT 0,
    buffer_count INTEGER DEFAULT 0,
    seek_count INTEGER DEFAULT 0,
    quality_changes INTEGER DEFAULT 0,
    average_bitrate BIGINT DEFAULT 0,
    peak_bitrate BIGINT DEFAULT 0,
    startup_time DOUBLE PRECISION DEFAULT 0,
    device_type VARCHAR(20),
    browser VARCHAR(100),
    os VARCHAR(100),
    country VARCHAR(100),
    completed BOOLEAN DEFAULT FALSE,
    error_occurred BOOLEAN DEFAULT FALSE
);

-- Create indexes for playback sessions
CREATE INDEX IF NOT EXISTS idx_playback_sessions_video_id ON playback_sessions(video_id);
CREATE INDEX IF NOT EXISTS idx_playback_sessions_user_id ON playback_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_playback_sessions_start_time ON playback_sessions(start_time);

-- Video Analytics Table (Aggregated Statistics)
CREATE TABLE IF NOT EXISTS video_analytics (
    video_id VARCHAR(36) PRIMARY KEY REFERENCES videos(id) ON DELETE CASCADE,
    total_views BIGINT DEFAULT 0,
    unique_viewers BIGINT DEFAULT 0,
    total_watch_time DOUBLE PRECISION DEFAULT 0, -- in hours
    average_watch_time DOUBLE PRECISION DEFAULT 0,
    completion_rate DOUBLE PRECISION DEFAULT 0,
    average_buffer_time DOUBLE PRECISION DEFAULT 0,
    buffer_rate DOUBLE PRECISION DEFAULT 0,
    error_rate DOUBLE PRECISION DEFAULT 0,
    average_startup_time DOUBLE PRECISION DEFAULT 0,
    last_updated TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Quality of Experience (QoE) Metrics Table
CREATE TABLE IF NOT EXISTS qoe_metrics (
    id SERIAL PRIMARY KEY,
    video_id VARCHAR(36) NOT NULL REFERENCES videos(id) ON DELETE CASCADE,
    output_id VARCHAR(36),
    period VARCHAR(20) NOT NULL, -- hourly, daily, weekly, monthly
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    view_count BIGINT DEFAULT 0,
    average_qoe DOUBLE PRECISION DEFAULT 0, -- 0-100 score
    rebuffer_ratio DOUBLE PRECISION DEFAULT 0,
    startup_time DOUBLE PRECISION DEFAULT 0,
    bitrate_utilization DOUBLE PRECISION DEFAULT 0,
    error_rate DOUBLE PRECISION DEFAULT 0,
    completion_rate DOUBLE PRECISION DEFAULT 0
);

-- Create indexes for QoE metrics
CREATE INDEX IF NOT EXISTS idx_qoe_metrics_video_id ON qoe_metrics(video_id);
CREATE INDEX IF NOT EXISTS idx_qoe_metrics_timestamp ON qoe_metrics(timestamp);
CREATE INDEX IF NOT EXISTS idx_qoe_metrics_period ON qoe_metrics(period);

-- Bandwidth Usage Table
CREATE TABLE IF NOT EXISTS bandwidth_usage (
    id VARCHAR(36) PRIMARY KEY,
    video_id VARCHAR(36) NOT NULL REFERENCES videos(id) ON DELETE CASCADE,
    output_id VARCHAR(36),
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    bytes_served BIGINT DEFAULT 0,
    request_count BIGINT DEFAULT 0,
    cdn_node VARCHAR(100),
    country VARCHAR(100)
);

-- Create indexes for bandwidth usage
CREATE INDEX IF NOT EXISTS idx_bandwidth_usage_video_id ON bandwidth_usage(video_id);
CREATE INDEX IF NOT EXISTS idx_bandwidth_usage_timestamp ON bandwidth_usage(timestamp);

-- Scene Detection Results Table (for caching scene analysis)
CREATE TABLE IF NOT EXISTS scene_detection_results (
    id VARCHAR(36) PRIMARY KEY,
    video_id VARCHAR(36) NOT NULL REFERENCES videos(id) ON DELETE CASCADE,
    total_scenes INTEGER DEFAULT 0,
    threshold DOUBLE PRECISION,
    scenes JSONB DEFAULT '[]'::jsonb, -- Array of scene information
    best_scene JSONB, -- Best scene metadata
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_scene_detection_video_id ON scene_detection_results(video_id);

-- Watermarked Videos Table
CREATE TABLE IF NOT EXISTS watermarked_videos (
    id VARCHAR(36) PRIMARY KEY,
    original_video_id VARCHAR(36) NOT NULL REFERENCES videos(id) ON DELETE CASCADE,
    watermark_type VARCHAR(20) NOT NULL, -- 'text' or 'image'
    watermark_config JSONB NOT NULL, -- Watermark settings
    output_path TEXT NOT NULL,
    output_url TEXT NOT NULL,
    size BIGINT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_watermarked_videos_original ON watermarked_videos(original_video_id);

-- Concatenated Videos Table
CREATE TABLE IF NOT EXISTS concatenated_videos (
    id VARCHAR(36) PRIMARY KEY,
    source_video_ids TEXT[] NOT NULL, -- Array of source video IDs
    method VARCHAR(20) NOT NULL, -- 'concat' or 'filter'
    transition_type VARCHAR(20), -- 'none', 'fade', 'dissolve'
    output_path TEXT NOT NULL,
    output_url TEXT NOT NULL,
    size BIGINT NOT NULL,
    duration DOUBLE PRECISION NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create function to update video_analytics timestamp
CREATE OR REPLACE FUNCTION update_video_analytics_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.last_updated = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger for video_analytics
DROP TRIGGER IF EXISTS trigger_update_video_analytics_timestamp ON video_analytics;
CREATE TRIGGER trigger_update_video_analytics_timestamp
    BEFORE UPDATE ON video_analytics
    FOR EACH ROW
    EXECUTE FUNCTION update_video_analytics_timestamp();

-- Create materialized view for trending videos (refreshed periodically)
CREATE MATERIALIZED VIEW IF NOT EXISTS trending_videos AS
WITH recent_views AS (
    SELECT
        video_id,
        COUNT(*) as views,
        COUNT(DISTINCT user_id) as unique_users
    FROM playback_sessions
    WHERE start_time > NOW() - INTERVAL '24 hours'
    GROUP BY video_id
),
previous_views AS (
    SELECT
        video_id,
        COUNT(*) as views
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
    COALESCE(r.views, 0) * 1.0 + COALESCE(r.unique_users, 0) * 0.5 as trending_score,
    NOW() as last_updated
FROM videos v
LEFT JOIN recent_views r ON v.id = r.video_id
LEFT JOIN previous_views p ON v.id = p.video_id
WHERE COALESCE(r.views, 0) > 0
ORDER BY trending_score DESC;

-- Create index on materialized view
CREATE INDEX IF NOT EXISTS idx_trending_videos_score ON trending_videos(trending_score DESC);

-- Add comments for documentation
COMMENT ON TABLE playback_events IS 'Individual playback events for detailed analytics';
COMMENT ON TABLE playback_sessions IS 'Aggregated playback sessions for performance analysis';
COMMENT ON TABLE video_analytics IS 'Pre-computed analytics summaries for videos';
COMMENT ON TABLE qoe_metrics IS 'Quality of Experience metrics over time';
COMMENT ON TABLE bandwidth_usage IS 'Bandwidth consumption tracking';
COMMENT ON TABLE scene_detection_results IS 'Cached scene detection results';
COMMENT ON TABLE watermarked_videos IS 'Videos with applied watermarks';
COMMENT ON TABLE concatenated_videos IS 'Videos created from concatenation';
