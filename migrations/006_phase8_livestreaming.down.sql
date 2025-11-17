-- Phase 8: Live Streaming Support Rollback

-- Drop triggers
DROP TRIGGER IF EXISTS trigger_update_live_streams_updated_at ON live_streams;
DROP TRIGGER IF EXISTS trigger_update_dvr_recordings_updated_at ON dvr_recordings;

-- Drop functions
DROP FUNCTION IF EXISTS update_live_streams_updated_at();
DROP FUNCTION IF EXISTS cleanup_old_live_stream_analytics();
DROP FUNCTION IF EXISTS archive_expired_dvr_recordings();

-- Drop tables (in reverse order of dependencies)
DROP TABLE IF EXISTS live_stream_viewers;
DROP TABLE IF EXISTS live_stream_events;
DROP TABLE IF EXISTS live_stream_analytics;
DROP TABLE IF EXISTS dvr_recordings;
DROP TABLE IF EXISTS live_stream_variants;
DROP TABLE IF EXISTS live_streams;
