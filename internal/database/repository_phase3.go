package database

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/therealutkarshpriyadarshi/transcode/pkg/models"
	"golang.org/x/crypto/bcrypt"
)

// User management methods

// CreateUser creates a new user
func (r *Repository) CreateUser(ctx context.Context, user *models.User, password string) error {
	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Generate API key
	apiKey, err := generateAPIKey()
	if err != nil {
		return fmt.Errorf("failed to generate API key: %w", err)
	}

	query := `
		INSERT INTO users (id, email, password_hash, api_key, quota, used_quota, quota_reset_at, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`

	_, err = r.db.Exec(ctx, query,
		user.ID,
		user.Email,
		string(hashedPassword),
		apiKey,
		user.Quota,
		0,
		time.Now().Add(24*time.Hour),
		user.IsActive,
	)

	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	user.APIKey = apiKey
	return nil
}

// GetUserByEmail retrieves a user by email
func (r *Repository) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	query := `
		SELECT id, email, password_hash, api_key, quota, used_quota, quota_reset_at, is_active, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	var user models.User
	err := r.db.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.APIKey,
		&user.Quota,
		&user.UsedQuota,
		&user.QuotaResetAt,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

// GetUserByID retrieves a user by ID
func (r *Repository) GetUserByID(ctx context.Context, userID string) (*models.User, error) {
	query := `
		SELECT id, email, password_hash, api_key, quota, used_quota, quota_reset_at, is_active, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	var user models.User
	err := r.db.QueryRow(ctx, query, userID).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.APIKey,
		&user.Quota,
		&user.UsedQuota,
		&user.QuotaResetAt,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

// ValidateAPIKey validates an API key and returns the user
func (r *Repository) ValidateAPIKey(ctx context.Context, apiKey string) (*models.User, error) {
	query := `
		SELECT id, email, password_hash, api_key, quota, used_quota, quota_reset_at, is_active, created_at, updated_at
		FROM users
		WHERE api_key = $1
	`

	var user models.User
	err := r.db.QueryRow(ctx, query, apiKey).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.APIKey,
		&user.Quota,
		&user.UsedQuota,
		&user.QuotaResetAt,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("invalid API key")
	}

	if err != nil {
		return nil, fmt.Errorf("failed to validate API key: %w", err)
	}

	return &user, nil
}

// CheckQuota checks if user has available quota
func (r *Repository) CheckQuota(ctx context.Context, userID string) (bool, error) {
	user, err := r.GetUserByID(ctx, userID)
	if err != nil {
		return false, err
	}

	// Reset quota if expired
	if time.Now().After(user.QuotaResetAt) {
		if err := r.ResetUserQuota(ctx, userID); err != nil {
			return false, err
		}
		return true, nil
	}

	return user.UsedQuota < user.Quota, nil
}

// IncrementQuota increments user's used quota
func (r *Repository) IncrementQuota(ctx context.Context, userID string) error {
	query := `
		UPDATE users
		SET used_quota = used_quota + 1
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query, userID)
	return err
}

// ResetUserQuota resets a user's quota
func (r *Repository) ResetUserQuota(ctx context.Context, userID string) error {
	query := `
		UPDATE users
		SET used_quota = 0,
		    quota_reset_at = CURRENT_TIMESTAMP + INTERVAL '1 day'
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query, userID)
	return err
}

// Webhook management methods

// CreateWebhook creates a new webhook
func (r *Repository) CreateWebhook(ctx context.Context, webhook *models.Webhook) error {
	query := `
		INSERT INTO webhooks (id, user_id, url, events, secret, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`

	_, err := r.db.Exec(ctx, query,
		webhook.ID,
		webhook.UserID,
		webhook.URL,
		webhook.Events,
		webhook.Secret,
		webhook.IsActive,
	)

	if err != nil {
		return fmt.Errorf("failed to create webhook: %w", err)
	}

	return nil
}

// GetWebhooksByEvent retrieves webhooks subscribed to a specific event
func (r *Repository) GetWebhooksByEvent(ctx context.Context, event string) ([]*models.Webhook, error) {
	// Map event to JSONB field
	eventField := ""
	switch event {
	case models.WebhookEventJobStarted:
		eventField = "job_started"
	case models.WebhookEventJobCompleted:
		eventField = "job_completed"
	case models.WebhookEventJobFailed:
		eventField = "job_failed"
	case models.WebhookEventJobProgress:
		eventField = "job_progress"
	case models.WebhookEventVideoUploaded:
		eventField = "video_uploaded"
	default:
		return nil, fmt.Errorf("unknown event: %s", event)
	}

	query := fmt.Sprintf(`
		SELECT id, user_id, url, events, secret, is_active, created_at, updated_at
		FROM webhooks
		WHERE is_active = true
		AND (events->>'%s')::boolean = true
	`, eventField)

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get webhooks: %w", err)
	}
	defer rows.Close()

	var webhooks []*models.Webhook
	for rows.Next() {
		var webhook models.Webhook
		err := rows.Scan(
			&webhook.ID,
			&webhook.UserID,
			&webhook.URL,
			&webhook.Events,
			&webhook.Secret,
			&webhook.IsActive,
			&webhook.CreatedAt,
			&webhook.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan webhook: %w", err)
		}
		webhooks = append(webhooks, &webhook)
	}

	return webhooks, nil
}

// GetUserWebhooks retrieves all webhooks for a user
func (r *Repository) GetUserWebhooks(ctx context.Context, userID string) ([]*models.Webhook, error) {
	query := `
		SELECT id, user_id, url, events, secret, is_active, created_at, updated_at
		FROM webhooks
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get webhooks: %w", err)
	}
	defer rows.Close()

	var webhooks []*models.Webhook
	for rows.Next() {
		var webhook models.Webhook
		err := rows.Scan(
			&webhook.ID,
			&webhook.UserID,
			&webhook.URL,
			&webhook.Events,
			&webhook.Secret,
			&webhook.IsActive,
			&webhook.CreatedAt,
			&webhook.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan webhook: %w", err)
		}
		webhooks = append(webhooks, &webhook)
	}

	return webhooks, nil
}

// Webhook delivery methods

// CreateDelivery creates a new webhook delivery record
func (r *Repository) CreateDelivery(ctx context.Context, delivery *models.WebhookDelivery) error {
	query := `
		INSERT INTO webhook_deliveries (id, webhook_id, event, payload, status, status_code, response_body, retry_count, next_retry_at, created_at, completed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, CURRENT_TIMESTAMP, $10)
	`

	_, err := r.db.Exec(ctx, query,
		delivery.ID,
		delivery.WebhookID,
		delivery.Event,
		delivery.Payload,
		delivery.Status,
		delivery.StatusCode,
		delivery.ResponseBody,
		delivery.RetryCount,
		delivery.NextRetryAt,
		delivery.CompletedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create delivery: %w", err)
	}

	return nil
}

// UpdateDelivery updates a webhook delivery record
func (r *Repository) UpdateDelivery(ctx context.Context, delivery *models.WebhookDelivery) error {
	query := `
		UPDATE webhook_deliveries
		SET status = $2,
		    status_code = $3,
		    response_body = $4,
		    retry_count = $5,
		    next_retry_at = $6,
		    completed_at = $7
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query,
		delivery.ID,
		delivery.Status,
		delivery.StatusCode,
		delivery.ResponseBody,
		delivery.RetryCount,
		delivery.NextRetryAt,
		delivery.CompletedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to update delivery: %w", err)
	}

	return nil
}

// GetPendingDeliveries retrieves pending webhook deliveries
func (r *Repository) GetPendingDeliveries(ctx context.Context, limit int) ([]*models.WebhookDelivery, error) {
	query := `
		SELECT id, webhook_id, event, payload, status, status_code, response_body, retry_count, next_retry_at, created_at, completed_at
		FROM webhook_deliveries
		WHERE status = $1
		AND (next_retry_at IS NULL OR next_retry_at <= CURRENT_TIMESTAMP)
		ORDER BY created_at ASC
		LIMIT $2
	`

	rows, err := r.db.Query(ctx, query, models.WebhookDeliveryStatusPending, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending deliveries: %w", err)
	}
	defer rows.Close()

	var deliveries []*models.WebhookDelivery
	for rows.Next() {
		var delivery models.WebhookDelivery
		err := rows.Scan(
			&delivery.ID,
			&delivery.WebhookID,
			&delivery.Event,
			&delivery.Payload,
			&delivery.Status,
			&delivery.StatusCode,
			&delivery.ResponseBody,
			&delivery.RetryCount,
			&delivery.NextRetryAt,
			&delivery.CreatedAt,
			&delivery.CompletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan delivery: %w", err)
		}
		deliveries = append(deliveries, &delivery)
	}

	return deliveries, nil
}

// Job management methods

// UpdateJobStatus updates a job's status
func (r *Repository) UpdateJobStatus(ctx context.Context, jobID, status string) error {
	query := `
		UPDATE jobs
		SET status = $2
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query, jobID, status)
	return err
}

// GetPendingJobs retrieves pending jobs
func (r *Repository) GetPendingJobs(ctx context.Context, limit int) ([]*models.Job, error) {
	query := `
		SELECT id, video_id, status, priority, progress, error_msg, retry_count, worker_id, started_at, completed_at, created_at, updated_at, config
		FROM jobs
		WHERE status = $1
		ORDER BY priority DESC, created_at ASC
		LIMIT $2
	`

	rows, err := r.db.Query(ctx, query, models.JobStatusPending, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending jobs: %w", err)
	}
	defer rows.Close()

	var jobs []*models.Job
	for rows.Next() {
		var job models.Job
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

// GetJobByID retrieves a job by ID
func (r *Repository) GetJobByID(ctx context.Context, jobID string) (*models.Job, error) {
	return r.GetJob(ctx, jobID)
}

// CancelJob marks a job as cancelled
func (r *Repository) CancelJob(ctx context.Context, jobID string) error {
	query := `
		UPDATE jobs
		SET status = $2,
		    completed_at = CURRENT_TIMESTAMP
		WHERE id = $1
		AND status IN ($3, $4)
	`

	result, err := r.db.Exec(ctx, query, jobID, models.JobStatusCancelled, models.JobStatusPending, models.JobStatusQueued)
	if err != nil {
		return fmt.Errorf("failed to cancel job: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("job not found or cannot be cancelled")
	}

	return nil
}

// PauseJob pauses a job
func (r *Repository) PauseJob(ctx context.Context, jobID string) error {
	query := `
		UPDATE jobs
		SET paused_at = CURRENT_TIMESTAMP
		WHERE id = $1
		AND status = $2
		AND paused_at IS NULL
	`

	result, err := r.db.Exec(ctx, query, jobID, models.JobStatusProcessing)
	if err != nil {
		return fmt.Errorf("failed to pause job: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("job not found or cannot be paused")
	}

	return nil
}

// ResumeJob resumes a paused job
func (r *Repository) ResumeJob(ctx context.Context, jobID string) error {
	query := `
		UPDATE jobs
		SET paused_at = NULL,
		    resume_at = CURRENT_TIMESTAMP
		WHERE id = $1
		AND paused_at IS NOT NULL
	`

	result, err := r.db.Exec(ctx, query, jobID)
	if err != nil {
		return fmt.Errorf("failed to resume job: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("job not found or is not paused")
	}

	return nil
}

// Helper functions

func generateAPIKey() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
