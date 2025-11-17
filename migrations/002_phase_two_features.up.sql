-- Phase 2: Multi-Resolution & Adaptive Streaming Schema Updates

-- Add streaming_type to outputs table for HLS/DASH support
ALTER TABLE outputs ADD COLUMN IF NOT EXISTS streaming_type VARCHAR(20) DEFAULT 'progressive';
ALTER TABLE outputs ADD COLUMN IF NOT EXISTS manifest_url TEXT;
ALTER TABLE outputs ADD COLUMN IF NOT EXISTS segment_duration DOUBLE PRECISION;
ALTER TABLE outputs ADD COLUMN IF NOT EXISTS audio_codec VARCHAR(50);
ALTER TABLE outputs ADD COLUMN IF NOT EXISTS audio_bitrate INTEGER;

-- Create thumbnails table for storing video thumbnails and sprites
CREATE TABLE IF NOT EXISTS thumbnails (
    id VARCHAR(36) PRIMARY KEY,
    video_id VARCHAR(36) NOT NULL REFERENCES videos(id) ON DELETE CASCADE,
    thumbnail_type VARCHAR(20) NOT NULL, -- 'single', 'sprite', 'animated'
    url TEXT NOT NULL,
    path TEXT NOT NULL,
    width INTEGER NOT NULL,
    height INTEGER NOT NULL,
    timestamp DOUBLE PRECISION, -- For single thumbnails
    sprite_columns INTEGER, -- For sprite sheets
    sprite_rows INTEGER, -- For sprite sheets
    interval_seconds DOUBLE PRECISION, -- For sprites
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create subtitles table for storing subtitle tracks
CREATE TABLE IF NOT EXISTS subtitles (
    id VARCHAR(36) PRIMARY KEY,
    video_id VARCHAR(36) NOT NULL REFERENCES videos(id) ON DELETE CASCADE,
    language VARCHAR(10) NOT NULL,
    label VARCHAR(100),
    format VARCHAR(20) NOT NULL, -- 'vtt', 'srt', 'ass'
    url TEXT NOT NULL,
    path TEXT NOT NULL,
    is_default BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create streaming_profiles table for adaptive streaming manifests
CREATE TABLE IF NOT EXISTS streaming_profiles (
    id VARCHAR(36) PRIMARY KEY,
    video_id VARCHAR(36) NOT NULL REFERENCES videos(id) ON DELETE CASCADE,
    job_id VARCHAR(36) REFERENCES jobs(id) ON DELETE CASCADE,
    profile_type VARCHAR(20) NOT NULL, -- 'hls', 'dash'
    master_manifest_url TEXT NOT NULL,
    master_manifest_path TEXT NOT NULL,
    variant_count INTEGER NOT NULL DEFAULT 0,
    audio_only BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create audio_tracks table for multiple audio streams
CREATE TABLE IF NOT EXISTS audio_tracks (
    id VARCHAR(36) PRIMARY KEY,
    video_id VARCHAR(36) NOT NULL REFERENCES videos(id) ON DELETE CASCADE,
    streaming_profile_id VARCHAR(36) REFERENCES streaming_profiles(id) ON DELETE CASCADE,
    language VARCHAR(10) NOT NULL,
    label VARCHAR(100),
    codec VARCHAR(50) NOT NULL,
    bitrate INTEGER NOT NULL,
    channels INTEGER DEFAULT 2,
    sample_rate INTEGER DEFAULT 48000,
    url TEXT NOT NULL,
    path TEXT NOT NULL,
    is_default BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Add indexes for new tables
CREATE INDEX IF NOT EXISTS idx_thumbnails_video_id ON thumbnails(video_id);
CREATE INDEX IF NOT EXISTS idx_thumbnails_type ON thumbnails(thumbnail_type);

CREATE INDEX IF NOT EXISTS idx_subtitles_video_id ON subtitles(video_id);
CREATE INDEX IF NOT EXISTS idx_subtitles_language ON subtitles(language);

CREATE INDEX IF NOT EXISTS idx_streaming_profiles_video_id ON streaming_profiles(video_id);
CREATE INDEX IF NOT EXISTS idx_streaming_profiles_job_id ON streaming_profiles(job_id);
CREATE INDEX IF NOT EXISTS idx_streaming_profiles_type ON streaming_profiles(profile_type);

CREATE INDEX IF NOT EXISTS idx_audio_tracks_video_id ON audio_tracks(video_id);
CREATE INDEX IF NOT EXISTS idx_audio_tracks_profile_id ON audio_tracks(streaming_profile_id);

-- Add new metadata columns to jobs for multi-resolution support
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS resolutions JSONB DEFAULT '[]'::jsonb;
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS enable_hls BOOLEAN DEFAULT false;
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS enable_dash BOOLEAN DEFAULT false;
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS generate_thumbnails BOOLEAN DEFAULT false;
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS extract_subtitles BOOLEAN DEFAULT false;
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS normalize_audio BOOLEAN DEFAULT false;
