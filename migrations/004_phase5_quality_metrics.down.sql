-- Rollback Phase 5 migrations

-- Remove added columns from existing tables
ALTER TABLE outputs DROP COLUMN IF EXISTS encoding_efficiency;
ALTER TABLE outputs DROP COLUMN IF EXISTS psnr_score;
ALTER TABLE outputs DROP COLUMN IF EXISTS ssim_score;
ALTER TABLE outputs DROP COLUMN IF EXISTS vmaf_score;

ALTER TABLE jobs DROP COLUMN IF EXISTS actual_vmaf;
ALTER TABLE jobs DROP COLUMN IF EXISTS target_vmaf;
ALTER TABLE jobs DROP COLUMN IF EXISTS quality_profile_id;

-- Drop Phase 5 tables in reverse order
DROP TABLE IF EXISTS quality_presets;
DROP TABLE IF EXISTS bitrate_experiments;
DROP TABLE IF EXISTS content_complexity;
DROP TABLE IF EXISTS encoding_profiles;
DROP TABLE IF EXISTS quality_analysis;
