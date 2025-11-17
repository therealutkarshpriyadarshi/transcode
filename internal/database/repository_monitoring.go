package database

import (
	"context"
	"fmt"
)

// Monitoring-related repository methods

// GetJobStats returns statistics about jobs
func (r *Repository) GetJobStats(ctx context.Context) (total, completed, failed, cancelled int64, err error) {
	query := `
		SELECT
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE status = 'completed') as completed,
			COUNT(*) FILTER (WHERE status = 'failed') as failed,
			COUNT(*) FILTER (WHERE status = 'cancelled') as cancelled
		FROM jobs
	`

	err = r.db.QueryRow(ctx, query).Scan(&total, &completed, &failed, &cancelled)
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("failed to get job stats: %w", err)
	}

	return total, completed, failed, cancelled, nil
}

// GetAverageWaitTime returns the average wait time for jobs (from creation to start)
func (r *Repository) GetAverageWaitTime(ctx context.Context) (float64, error) {
	query := `
		SELECT COALESCE(AVG(EXTRACT(EPOCH FROM (started_at - created_at))), 0)
		FROM jobs
		WHERE started_at IS NOT NULL
		AND created_at > NOW() - INTERVAL '24 hours'
	`

	var avgWaitTime float64
	err := r.db.QueryRow(ctx, query).Scan(&avgWaitTime)
	if err != nil {
		return 0, fmt.Errorf("failed to get average wait time: %w", err)
	}

	return avgWaitTime, nil
}

// GetAverageProcessTime returns the average processing time for jobs
func (r *Repository) GetAverageProcessTime(ctx context.Context) (float64, error) {
	query := `
		SELECT COALESCE(AVG(EXTRACT(EPOCH FROM (completed_at - started_at))), 0)
		FROM jobs
		WHERE started_at IS NOT NULL
		AND completed_at IS NOT NULL
		AND status = 'completed'
		AND created_at > NOW() - INTERVAL '24 hours'
	`

	var avgProcessTime float64
	err := r.db.QueryRow(ctx, query).Scan(&avgProcessTime)
	if err != nil {
		return 0, fmt.Errorf("failed to get average process time: %w", err)
	}

	return avgProcessTime, nil
}

// GetActiveWorkers returns the number of active workers
func (r *Repository) GetActiveWorkers(ctx context.Context) (int, error) {
	query := `
		SELECT COUNT(DISTINCT worker_id)
		FROM jobs
		WHERE worker_id IS NOT NULL
		AND worker_id != ''
		AND updated_at > NOW() - INTERVAL '5 minutes'
	`

	var count int
	err := r.db.QueryRow(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get active workers: %w", err)
	}

	return count, nil
}

// GetJobsByStatus returns count of jobs by status
func (r *Repository) GetJobsByStatus(ctx context.Context) (map[string]int64, error) {
	query := `
		SELECT status, COUNT(*)
		FROM jobs
		GROUP BY status
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get jobs by status: %w", err)
	}
	defer rows.Close()

	stats := make(map[string]int64)
	for rows.Next() {
		var status string
		var count int64
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		stats[status] = count
	}

	return stats, nil
}

// GetRecentFailedJobs returns recent failed jobs
func (r *Repository) GetRecentFailedJobs(ctx context.Context, limit int) ([]*Job, error) {
	query := `
		SELECT id, video_id, status, priority, progress, error_msg, retry_count, worker_id, started_at, completed_at, created_at, updated_at, config
		FROM jobs
		WHERE status = 'failed'
		ORDER BY updated_at DESC
		LIMIT $1
	`

	rows, err := r.db.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent failed jobs: %w", err)
	}
	defer rows.Close()

	var jobs []*Job
	for rows.Next() {
		var job Job
		err := rows.Scan(
			&job.ID,
			&job.VideoID,
			&job.Status,
			&job.Priority,
			&job.Progress,
			&job.ErrorMsg,
			&job.RetryCount,
			&job.WorkerID,
			&job.StartedAt,
			&job.CompletedAt,
			&job.CreatedAt,
			&job.UpdatedAt,
			&job.Config,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan job: %w", err)
		}
		jobs = append(jobs, &job)
	}

	return jobs, nil
}

// Job represents a minimal job structure for monitoring
type Job struct {
	ID          string
	VideoID     string
	Status      string
	Priority    int
	Progress    float64
	ErrorMsg    string
	RetryCount  int
	WorkerID    string
	StartedAt   interface{}
	CompletedAt interface{}
	CreatedAt   interface{}
	UpdatedAt   interface{}
	Config      interface{}
}
