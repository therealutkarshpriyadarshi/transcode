-- Phase 7: Rollback Analytics and Advanced Features Migration

-- Drop materialized view
DROP MATERIALIZED VIEW IF EXISTS trending_videos;

-- Drop trigger and function
DROP TRIGGER IF EXISTS trigger_update_video_analytics_timestamp ON video_analytics;
DROP FUNCTION IF EXISTS update_video_analytics_timestamp();

-- Drop tables in reverse order of dependencies
DROP TABLE IF EXISTS concatenated_videos;
DROP TABLE IF EXISTS watermarked_videos;
DROP TABLE IF EXISTS scene_detection_results;
DROP TABLE IF EXISTS bandwidth_usage;
DROP TABLE IF EXISTS qoe_metrics;
DROP TABLE IF EXISTS video_analytics;
DROP TABLE IF EXISTS playback_sessions;
DROP TABLE IF EXISTS playback_events;
