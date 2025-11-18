package database

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/therealutkarshpriyadarshi/transcode/pkg/models"
)

// Repository provides database operations
type Repository struct {
	db *DB
}

// NewRepository creates a new repository
func NewRepository(db *DB) *Repository {
	return &Repository{db: db}
}

// Videos

// CreateVideo creates a new video record
func (r *Repository) CreateVideo(ctx context.Context, video *models.Video) error {
	if video.ID == "" {
		video.ID = uuid.New().String()
	}

	query := `
		INSERT INTO videos (id, filename, original_url, size, duration, width, height, codec, bitrate, frame_rate, metadata, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING created_at, updated_at
	`

	err := r.db.Pool.QueryRow(ctx, query,
		video.ID, video.Filename, video.OriginalURL, video.Size, video.Duration,
		video.Width, video.Height, video.Codec, video.Bitrate, video.FrameRate,
		video.Metadata, video.Status,
	).Scan(&video.CreatedAt, &video.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create video: %w", err)
	}

	return nil
}

// GetVideo retrieves a video by ID
func (r *Repository) GetVideo(ctx context.Context, id string) (*models.Video, error) {
	var video models.Video

	query := `
		SELECT id, filename, original_url, size, duration, width, height, codec,
		       bitrate, frame_rate, metadata, status, created_at, updated_at
		FROM videos
		WHERE id = $1
	`

	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&video.ID, &video.Filename, &video.OriginalURL, &video.Size, &video.Duration,
		&video.Width, &video.Height, &video.Codec, &video.Bitrate, &video.FrameRate,
		&video.Metadata, &video.Status, &video.CreatedAt, &video.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("video not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get video: %w", err)
	}

	return &video, nil
}

// UpdateVideo updates a video record
func (r *Repository) UpdateVideo(ctx context.Context, video *models.Video) error {
	query := `
		UPDATE videos
		SET filename = $2, original_url = $3, size = $4, duration = $5, width = $6,
		    height = $7, codec = $8, bitrate = $9, frame_rate = $10, metadata = $11, status = $12
		WHERE id = $1
	`

	_, err := r.db.Pool.Exec(ctx, query,
		video.ID, video.Filename, video.OriginalURL, video.Size, video.Duration,
		video.Width, video.Height, video.Codec, video.Bitrate, video.FrameRate,
		video.Metadata, video.Status,
	)

	if err != nil {
		return fmt.Errorf("failed to update video: %w", err)
	}

	return nil
}

// ListVideos retrieves all videos with pagination
func (r *Repository) ListVideos(ctx context.Context, limit, offset int) ([]*models.Video, error) {
	query := `
		SELECT id, filename, original_url, size, duration, width, height, codec,
		       bitrate, frame_rate, metadata, status, created_at, updated_at
		FROM videos
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.Pool.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list videos: %w", err)
	}
	defer rows.Close()

	var videos []*models.Video
	for rows.Next() {
		var video models.Video
		err := rows.Scan(
			&video.ID, &video.Filename, &video.OriginalURL, &video.Size, &video.Duration,
			&video.Width, &video.Height, &video.Codec, &video.Bitrate, &video.FrameRate,
			&video.Metadata, &video.Status, &video.CreatedAt, &video.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan video: %w", err)
		}
		videos = append(videos, &video)
	}

	return videos, nil
}

// Jobs

// CreateJob creates a new job record
func (r *Repository) CreateJob(ctx context.Context, job *models.Job) error {
	if job.ID == "" {
		job.ID = uuid.New().String()
	}

	query := `
		INSERT INTO jobs (id, video_id, status, priority, progress, retry_count, config)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING created_at, updated_at
	`

	err := r.db.Pool.QueryRow(ctx, query,
		job.ID, job.VideoID, job.Status, job.Priority, job.Progress, job.RetryCount, job.Config,
	).Scan(&job.CreatedAt, &job.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create job: %w", err)
	}

	return nil
}

// GetJob retrieves a job by ID
func (r *Repository) GetJob(ctx context.Context, id string) (*models.Job, error) {
	var job models.Job

	query := `
		SELECT id, video_id, status, priority, progress, error_msg, retry_count,
		       worker_id, started_at, completed_at, created_at, updated_at, config
		FROM jobs
		WHERE id = $1
	`

	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&job.ID, &job.VideoID, &job.Status, &job.Priority, &job.Progress,
		&job.ErrorMsg, &job.RetryCount, &job.WorkerID, &job.StartedAt,
		&job.CompletedAt, &job.CreatedAt, &job.UpdatedAt, &job.Config,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("job not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get job: %w", err)
	}

	return &job, nil
}

// UpdateJob updates a job record
func (r *Repository) UpdateJob(ctx context.Context, job *models.Job) error {
	query := `
		UPDATE jobs
		SET status = $2, priority = $3, progress = $4, error_msg = $5,
		    retry_count = $6, worker_id = $7, started_at = $8, completed_at = $9, config = $10
		WHERE id = $1
	`

	_, err := r.db.Pool.Exec(ctx, query,
		job.ID, job.Status, job.Priority, job.Progress, job.ErrorMsg,
		job.RetryCount, job.WorkerID, job.StartedAt, job.CompletedAt, job.Config,
	)

	if err != nil {
		return fmt.Errorf("failed to update job: %w", err)
	}

	return nil
}

// GetJobsByVideoID retrieves all jobs for a video
func (r *Repository) GetJobsByVideoID(ctx context.Context, videoID string) ([]*models.Job, error) {
	query := `
		SELECT id, video_id, status, priority, progress, error_msg, retry_count,
		       worker_id, started_at, completed_at, created_at, updated_at, config
		FROM jobs
		WHERE video_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.Pool.Query(ctx, query, videoID)
	if err != nil {
		return nil, fmt.Errorf("failed to get jobs: %w", err)
	}
	defer rows.Close()

	var jobs []*models.Job
	for rows.Next() {
		var job models.Job
		err := rows.Scan(
			&job.ID, &job.VideoID, &job.Status, &job.Priority, &job.Progress,
			&job.ErrorMsg, &job.RetryCount, &job.WorkerID, &job.StartedAt,
			&job.CompletedAt, &job.CreatedAt, &job.UpdatedAt, &job.Config,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan job: %w", err)
		}
		jobs = append(jobs, &job)
	}

	return jobs, nil
}

// Outputs

// CreateOutput creates a new output record
func (r *Repository) CreateOutput(ctx context.Context, output *models.Output) error {
	if output.ID == "" {
		output.ID = uuid.New().String()
	}

	query := `
		INSERT INTO outputs (id, job_id, video_id, format, resolution, width, height,
		                     codec, bitrate, size, duration, url, path)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING created_at
	`

	err := r.db.Pool.QueryRow(ctx, query,
		output.ID, output.JobID, output.VideoID, output.Format, output.Resolution,
		output.Width, output.Height, output.Codec, output.Bitrate, output.Size,
		output.Duration, output.URL, output.Path,
	).Scan(&output.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to create output: %w", err)
	}

	return nil
}

// GetOutputsByJobID retrieves all outputs for a job
func (r *Repository) GetOutputsByJobID(ctx context.Context, jobID string) ([]*models.Output, error) {
	query := `
		SELECT id, job_id, video_id, format, resolution, width, height, codec,
		       bitrate, size, duration, url, path, created_at
		FROM outputs
		WHERE job_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.Pool.Query(ctx, query, jobID)
	if err != nil {
		return nil, fmt.Errorf("failed to get outputs: %w", err)
	}
	defer rows.Close()

	var outputs []*models.Output
	for rows.Next() {
		var output models.Output
		err := rows.Scan(
			&output.ID, &output.JobID, &output.VideoID, &output.Format, &output.Resolution,
			&output.Width, &output.Height, &output.Codec, &output.Bitrate, &output.Size,
			&output.Duration, &output.URL, &output.Path, &output.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan output: %w", err)
		}
		outputs = append(outputs, &output)
	}

	return outputs, nil
}

// DeleteVideo deletes a video and all associated records
func (r *Repository) DeleteVideo(ctx context.Context, videoID string) error {
	// Start a transaction to ensure all deletions succeed or fail together
	tx, err := r.db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Delete outputs
	_, err = tx.Exec(ctx, "DELETE FROM outputs WHERE video_id = $1", videoID)
	if err != nil {
		return fmt.Errorf("failed to delete outputs: %w", err)
	}

	// Delete jobs
	_, err = tx.Exec(ctx, "DELETE FROM jobs WHERE video_id = $1", videoID)
	if err != nil {
		return fmt.Errorf("failed to delete jobs: %w", err)
	}

	// Delete thumbnails (if table exists)
	_, err = tx.Exec(ctx, "DELETE FROM thumbnails WHERE video_id = $1", videoID)
	if err != nil {
		// Ignore error if table doesn't exist
		// We'll continue with video deletion
	}

	// Delete subtitles (if table exists)
	_, err = tx.Exec(ctx, "DELETE FROM subtitles WHERE video_id = $1", videoID)
	if err != nil {
		// Ignore error if table doesn't exist
	}

	// Delete audio tracks (if table exists)
	_, err = tx.Exec(ctx, "DELETE FROM audio_tracks WHERE video_id = $1", videoID)
	if err != nil {
		// Ignore error if table doesn't exist
	}

	// Delete the video itself
	result, err := tx.Exec(ctx, "DELETE FROM videos WHERE id = $1", videoID)
	if err != nil {
		return fmt.Errorf("failed to delete video: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("video not found")
	}

	// Commit the transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// CancelJob cancels a job by updating its status
func (r *Repository) CancelJob(ctx context.Context, jobID string) error {
	query := `
		UPDATE jobs
		SET status = 'cancelled', updated_at = NOW()
		WHERE id = $1 AND status IN ('pending', 'queued', 'processing')
		RETURNING id
	`

	var id string
	err := r.db.Pool.QueryRow(ctx, query, jobID).Scan(&id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf("job not found or cannot be cancelled")
		}
		return fmt.Errorf("failed to cancel job: %w", err)
	}

	return nil
}

// GetOutputsByVideoID retrieves all outputs for a video
func (r *Repository) GetOutputsByVideoID(ctx context.Context, videoID string) ([]*models.Output, error) {
	query := `
		SELECT id, job_id, video_id, format, resolution, width, height, codec,
		       bitrate, size, duration, url, path, created_at
		FROM outputs
		WHERE video_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.Pool.Query(ctx, query, videoID)
	if err != nil {
		return nil, fmt.Errorf("failed to get outputs: %w", err)
	}
	defer rows.Close()

	var outputs []*models.Output
	for rows.Next() {
		var output models.Output
		err := rows.Scan(
			&output.ID, &output.JobID, &output.VideoID, &output.Format, &output.Resolution,
			&output.Width, &output.Height, &output.Codec, &output.Bitrate, &output.Size,
			&output.Duration, &output.URL, &output.Path, &output.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan output: %w", err)
		}
		outputs = append(outputs, &output)
	}

	return outputs, nil
}
