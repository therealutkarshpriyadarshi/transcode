-- Rollback Phase 2 features

-- Drop new tables
DROP TABLE IF EXISTS audio_tracks;
DROP TABLE IF EXISTS streaming_profiles;
DROP TABLE IF EXISTS subtitles;
DROP TABLE IF EXISTS thumbnails;

-- Remove columns from outputs
ALTER TABLE outputs DROP COLUMN IF EXISTS streaming_type;
ALTER TABLE outputs DROP COLUMN IF EXISTS manifest_url;
ALTER TABLE outputs DROP COLUMN IF EXISTS segment_duration;
ALTER TABLE outputs DROP COLUMN IF EXISTS audio_codec;
ALTER TABLE outputs DROP COLUMN IF EXISTS audio_bitrate;

-- Remove columns from jobs
ALTER TABLE jobs DROP COLUMN IF EXISTS resolutions;
ALTER TABLE jobs DROP COLUMN IF EXISTS enable_hls;
ALTER TABLE jobs DROP COLUMN IF EXISTS enable_dash;
ALTER TABLE jobs DROP COLUMN IF EXISTS generate_thumbnails;
ALTER TABLE jobs DROP COLUMN IF EXISTS extract_subtitles;
ALTER TABLE jobs DROP COLUMN IF EXISTS normalize_audio;
