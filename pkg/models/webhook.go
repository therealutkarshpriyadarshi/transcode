package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// Webhook represents a webhook configuration
type Webhook struct {
	ID        string         `json:"id" db:"id"`
	UserID    string         `json:"user_id" db:"user_id"`
	URL       string         `json:"url" db:"url"`
	Events    WebhookEvents  `json:"events" db:"events"`
	Secret    string         `json:"secret,omitempty" db:"secret"`
	IsActive  bool           `json:"is_active" db:"is_active"`
	CreatedAt time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt time.Time      `json:"updated_at" db:"updated_at"`
}

// WebhookEvents holds the events a webhook subscribes to
type WebhookEvents struct {
	JobStarted    bool `json:"job_started"`
	JobCompleted  bool `json:"job_completed"`
	JobFailed     bool `json:"job_failed"`
	JobProgress   bool `json:"job_progress"`
	VideoUploaded bool `json:"video_uploaded"`
}

// Value implements driver.Valuer for database storage
func (we WebhookEvents) Value() (driver.Value, error) {
	return json.Marshal(we)
}

// Scan implements sql.Scanner for database retrieval
func (we *WebhookEvents) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}

	return json.Unmarshal(bytes, we)
}

// WebhookDelivery represents a webhook delivery attempt
type WebhookDelivery struct {
	ID            string    `json:"id" db:"id"`
	WebhookID     string    `json:"webhook_id" db:"webhook_id"`
	Event         string    `json:"event" db:"event"`
	Payload       string    `json:"payload" db:"payload"`
	Status        string    `json:"status" db:"status"`
	StatusCode    int       `json:"status_code" db:"status_code"`
	ResponseBody  string    `json:"response_body,omitempty" db:"response_body"`
	RetryCount    int       `json:"retry_count" db:"retry_count"`
	NextRetryAt   *time.Time `json:"next_retry_at,omitempty" db:"next_retry_at"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	CompletedAt   *time.Time `json:"completed_at,omitempty" db:"completed_at"`
}

// WebhookDeliveryStatus constants
const (
	WebhookDeliveryStatusPending   = "pending"
	WebhookDeliveryStatusDelivered = "delivered"
	WebhookDeliveryStatusFailed    = "failed"
)

// WebhookEvent represents the payload sent to webhooks
type WebhookEvent struct {
	Event     string      `json:"event"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data"`
}

// Webhook event types
const (
	WebhookEventJobStarted    = "job.started"
	WebhookEventJobCompleted  = "job.completed"
	WebhookEventJobFailed     = "job.failed"
	WebhookEventJobProgress   = "job.progress"
	WebhookEventVideoUploaded = "video.uploaded"
)
