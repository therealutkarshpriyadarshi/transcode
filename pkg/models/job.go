package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// Job represents a transcoding job
type Job struct {
	ID          string        `json:"id" db:"id"`
	VideoID     string        `json:"video_id" db:"video_id"`
	Status      string        `json:"status" db:"status"`
	Priority    int           `json:"priority" db:"priority"`
	Progress    float64       `json:"progress" db:"progress"`
	ErrorMsg    string        `json:"error_msg,omitempty" db:"error_msg"`
	RetryCount  int           `json:"retry_count" db:"retry_count"`
	WorkerID    string        `json:"worker_id,omitempty" db:"worker_id"`
	StartedAt   *time.Time    `json:"started_at,omitempty" db:"started_at"`
	CompletedAt *time.Time    `json:"completed_at,omitempty" db:"completed_at"`
	CreatedAt   time.Time     `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at" db:"updated_at"`
	Config      TranscodeConfig `json:"config" db:"config"`
}

// TranscodeConfig holds transcoding configuration for a job
type TranscodeConfig struct {
	OutputFormat string              `json:"output_format"`
	Resolution   string              `json:"resolution"`
	Bitrate      int64               `json:"bitrate"`
	Codec        string              `json:"codec"`
	Preset       string              `json:"preset"`
	AudioCodec   string              `json:"audio_codec"`
	AudioBitrate int                 `json:"audio_bitrate"`
	Extra        map[string]string   `json:"extra,omitempty"`
}

// Value implements driver.Valuer for database storage
func (tc TranscodeConfig) Value() (driver.Value, error) {
	return json.Marshal(tc)
}

// Scan implements sql.Scanner for database retrieval
func (tc *TranscodeConfig) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}

	return json.Unmarshal(bytes, tc)
}

// JobStatus constants
const (
	JobStatusPending    = "pending"
	JobStatusQueued     = "queued"
	JobStatusProcessing = "processing"
	JobStatusCompleted  = "completed"
	JobStatusFailed     = "failed"
	JobStatusCancelled  = "cancelled"
)

// JobPriority constants
const (
	JobPriorityLow    = 0
	JobPriorityNormal = 5
	JobPriorityHigh   = 10
)
