-- Drop triggers
DROP TRIGGER IF EXISTS update_jobs_updated_at ON jobs;
DROP TRIGGER IF EXISTS update_videos_updated_at ON videos;

-- Drop function
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop indexes
DROP INDEX IF EXISTS idx_outputs_resolution;
DROP INDEX IF EXISTS idx_outputs_video_id;
DROP INDEX IF EXISTS idx_outputs_job_id;

DROP INDEX IF EXISTS idx_jobs_worker_id;
DROP INDEX IF EXISTS idx_jobs_created_at;
DROP INDEX IF EXISTS idx_jobs_priority;
DROP INDEX IF EXISTS idx_jobs_status;
DROP INDEX IF EXISTS idx_jobs_video_id;

DROP INDEX IF EXISTS idx_videos_created_at;
DROP INDEX IF EXISTS idx_videos_status;

-- Drop tables
DROP TABLE IF EXISTS outputs;
DROP TABLE IF EXISTS jobs;
DROP TABLE IF EXISTS videos;
