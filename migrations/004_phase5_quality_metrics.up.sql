-- Phase 5: AI-Powered Per-Title Encoding
-- Quality metrics and per-title encoding optimization

-- Quality analysis results table
CREATE TABLE IF NOT EXISTS quality_analysis (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    video_id UUID NOT NULL REFERENCES videos(id) ON DELETE CASCADE,
    analysis_type VARCHAR(50) NOT NULL, -- 'vmaf', 'ssim', 'psnr', 'complexity'
    segment_index INT, -- NULL for full video analysis
    segment_start_time FLOAT, -- Start time in seconds
    segment_duration FLOAT, -- Duration in seconds

    -- VMAF scores
    vmaf_score FLOAT, -- Overall VMAF score (0-100)
    vmaf_min FLOAT, -- Minimum VMAF score
    vmaf_max FLOAT, -- Maximum VMAF score
    vmaf_mean FLOAT, -- Mean VMAF score
    vmaf_harmonic_mean FLOAT, -- Harmonic mean (more sensitive to low scores)

    -- Other quality metrics
    ssim_score FLOAT, -- Structural Similarity Index (0-1)
    psnr_score FLOAT, -- Peak Signal-to-Noise Ratio (dB)

    -- Complexity metrics
    spatial_complexity FLOAT, -- SI (Spatial Information)
    temporal_complexity FLOAT, -- TI (Temporal Information)
    scene_complexity VARCHAR(20), -- 'low', 'medium', 'high'
    motion_score FLOAT, -- Motion intensity (0-1)

    -- Encoding settings used
    test_bitrate BIGINT, -- Bitrate used for this test (bits/s)
    test_resolution VARCHAR(20), -- Resolution tested
    test_codec VARCHAR(50), -- Codec used
    test_preset VARCHAR(50), -- Encoding preset

    -- Analysis metadata
    analyzed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    analysis_duration FLOAT, -- Time taken for analysis (seconds)

    CONSTRAINT unique_quality_analysis UNIQUE(video_id, analysis_type, segment_index, test_bitrate, test_resolution)
);

CREATE INDEX idx_quality_analysis_video ON quality_analysis(video_id);
CREATE INDEX idx_quality_analysis_type ON quality_analysis(analysis_type);
CREATE INDEX idx_quality_analysis_vmaf ON quality_analysis(vmaf_score);

-- Per-title encoding profiles table
CREATE TABLE IF NOT EXISTS encoding_profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    video_id UUID NOT NULL REFERENCES videos(id) ON DELETE CASCADE,
    profile_name VARCHAR(100) NOT NULL, -- 'optimized', 'high_quality', 'bandwidth_optimized'
    is_active BOOLEAN DEFAULT true,

    -- Video characteristics
    content_type VARCHAR(50), -- 'animation', 'sports', 'movie', 'presentation', 'gaming'
    complexity_level VARCHAR(20), -- 'low', 'medium', 'high', 'very_high'

    -- Recommended settings
    bitrate_ladder JSONB NOT NULL, -- Dynamic bitrate ladder: [{"resolution": "1080p", "bitrate": 5000000, "target_vmaf": 95}]
    codec_recommendation VARCHAR(50), -- Recommended codec
    preset_recommendation VARCHAR(50), -- Recommended preset

    -- Quality targets
    target_vmaf_score FLOAT, -- Target VMAF score
    min_vmaf_score FLOAT, -- Minimum acceptable VMAF

    -- Optimization results
    estimated_size_reduction FLOAT, -- Percentage size reduction vs standard ladder
    estimated_quality_improvement FLOAT, -- VMAF improvement
    confidence_score FLOAT, -- Confidence in recommendation (0-1)

    -- Metadata
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT unique_encoding_profile UNIQUE(video_id, profile_name)
);

CREATE INDEX idx_encoding_profiles_video ON encoding_profiles(video_id);
CREATE INDEX idx_encoding_profiles_active ON encoding_profiles(is_active);

-- Content complexity cache table
CREATE TABLE IF NOT EXISTS content_complexity (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    video_id UUID NOT NULL REFERENCES videos(id) ON DELETE CASCADE,

    -- Overall complexity
    overall_complexity VARCHAR(20), -- 'low', 'medium', 'high', 'very_high'
    complexity_score FLOAT, -- Normalized score (0-1)

    -- Spatial metrics
    avg_spatial_info FLOAT, -- Average SI across video
    max_spatial_info FLOAT, -- Maximum SI
    min_spatial_info FLOAT, -- Minimum SI

    -- Temporal metrics
    avg_temporal_info FLOAT, -- Average TI across video
    max_temporal_info FLOAT, -- Maximum TI
    min_temporal_info FLOAT, -- Minimum TI

    -- Motion analysis
    avg_motion_intensity FLOAT, -- Average motion (0-1)
    motion_variance FLOAT, -- Motion consistency
    scene_changes INT, -- Number of scene changes detected

    -- Color and detail
    color_variance FLOAT, -- Color diversity
    edge_density FLOAT, -- Amount of edges/detail
    contrast_ratio FLOAT, -- Dynamic range

    -- Content categorization
    content_category VARCHAR(50), -- Detected content type
    has_text_overlay BOOLEAN, -- Text/subtitles detected
    has_fast_motion BOOLEAN, -- High motion detected

    -- Analysis metadata
    sample_points INT, -- Number of frames analyzed
    analyzed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT unique_content_complexity UNIQUE(video_id)
);

CREATE INDEX idx_content_complexity_video ON content_complexity(video_id);
CREATE INDEX idx_content_complexity_level ON content_complexity(overall_complexity);

-- Bitrate ladder experiments table (for A/B testing)
CREATE TABLE IF NOT EXISTS bitrate_experiments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    video_id UUID NOT NULL REFERENCES videos(id) ON DELETE CASCADE,
    experiment_name VARCHAR(100) NOT NULL,

    -- Experiment configuration
    ladder_config JSONB NOT NULL, -- Bitrate ladder configuration
    encoding_params JSONB, -- Additional encoding parameters

    -- Results
    total_size BIGINT, -- Total size of all outputs
    avg_vmaf_score FLOAT, -- Average VMAF across resolutions
    min_vmaf_score FLOAT, -- Worst VMAF score
    encoding_time FLOAT, -- Total encoding time

    -- Comparison to baseline
    size_vs_baseline FLOAT, -- Percentage difference
    quality_vs_baseline FLOAT, -- VMAF difference

    -- Status
    status VARCHAR(20) DEFAULT 'pending', -- 'pending', 'running', 'completed', 'failed'
    started_at TIMESTAMP,
    completed_at TIMESTAMP,

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_bitrate_experiments_video ON bitrate_experiments(video_id);
CREATE INDEX idx_bitrate_experiments_status ON bitrate_experiments(status);

-- Quality presets table
CREATE TABLE IF NOT EXISTS quality_presets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL UNIQUE,
    description TEXT,

    -- Quality targets
    target_vmaf FLOAT NOT NULL, -- Target VMAF score
    min_vmaf FLOAT NOT NULL, -- Minimum acceptable VMAF

    -- Encoding preferences
    prefer_quality BOOLEAN DEFAULT false, -- Prioritize quality over size
    max_bitrate_multiplier FLOAT DEFAULT 1.5, -- Maximum bitrate relative to standard
    min_bitrate_multiplier FLOAT DEFAULT 0.6, -- Minimum bitrate relative to standard

    -- Standard bitrate ladder (baseline)
    standard_ladder JSONB NOT NULL,

    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Insert default quality presets
INSERT INTO quality_presets (name, description, target_vmaf, min_vmaf, prefer_quality, standard_ladder) VALUES
('high_quality', 'High quality encoding - VMAF 95+ - Ideal for premium content', 95.0, 93.0, true,
 '[
    {"resolution": "2160p", "bitrate": 25000000},
    {"resolution": "1440p", "bitrate": 16000000},
    {"resolution": "1080p", "bitrate": 8000000},
    {"resolution": "720p", "bitrate": 4000000},
    {"resolution": "480p", "bitrate": 2000000},
    {"resolution": "360p", "bitrate": 1000000}
  ]'::jsonb),
('standard_quality', 'Standard quality - VMAF 85+ - Balanced quality and size', 87.0, 82.0, false,
 '[
    {"resolution": "1080p", "bitrate": 5000000},
    {"resolution": "720p", "bitrate": 2500000},
    {"resolution": "480p", "bitrate": 1200000},
    {"resolution": "360p", "bitrate": 700000}
  ]'::jsonb),
('bandwidth_optimized', 'Low bandwidth - VMAF 75+ - Maximum compression', 78.0, 72.0, false,
 '[
    {"resolution": "720p", "bitrate": 1500000},
    {"resolution": "480p", "bitrate": 800000},
    {"resolution": "360p", "bitrate": 400000},
    {"resolution": "240p", "bitrate": 200000}
  ]'::jsonb);

-- Add quality_profile_id to jobs table (optional reference)
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS quality_profile_id UUID REFERENCES quality_presets(id);
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS target_vmaf FLOAT;
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS actual_vmaf FLOAT;

-- Add quality metrics to outputs table
ALTER TABLE outputs ADD COLUMN IF NOT EXISTS vmaf_score FLOAT;
ALTER TABLE outputs ADD COLUMN IF NOT EXISTS ssim_score FLOAT;
ALTER TABLE outputs ADD COLUMN IF NOT EXISTS psnr_score FLOAT;
ALTER TABLE outputs ADD COLUMN IF NOT EXISTS encoding_efficiency FLOAT; -- bits per VMAF point

-- Comments for documentation
COMMENT ON TABLE quality_analysis IS 'Stores VMAF and quality analysis results for videos';
COMMENT ON TABLE encoding_profiles IS 'Per-title optimized encoding profiles';
COMMENT ON TABLE content_complexity IS 'Content complexity analysis for intelligent encoding';
COMMENT ON TABLE bitrate_experiments IS 'A/B testing results for bitrate ladders';
COMMENT ON TABLE quality_presets IS 'Reusable quality presets with target VMAF scores';
