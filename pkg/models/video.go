package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// Video represents a video file in the system
type Video struct {
	ID          string    `json:"id" db:"id"`
	Filename    string    `json:"filename" db:"filename"`
	OriginalURL string    `json:"original_url" db:"original_url"`
	Size        int64     `json:"size" db:"size"`
	Duration    float64   `json:"duration" db:"duration"`
	Width       int       `json:"width" db:"width"`
	Height      int       `json:"height" db:"height"`
	Codec       string    `json:"codec" db:"codec"`
	Bitrate     int64     `json:"bitrate" db:"bitrate"`
	FrameRate   float64   `json:"frame_rate" db:"frame_rate"`
	Metadata    Metadata  `json:"metadata" db:"metadata"`
	Status      string    `json:"status" db:"status"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// Metadata holds additional video metadata
type Metadata map[string]interface{}

// Value implements driver.Valuer for database storage
func (m Metadata) Value() (driver.Value, error) {
	return json.Marshal(m)
}

// Scan implements sql.Scanner for database retrieval
func (m *Metadata) Scan(value interface{}) error {
	if value == nil {
		*m = make(Metadata)
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}

	return json.Unmarshal(bytes, m)
}

// VideoStatus constants
const (
	VideoStatusPending    = "pending"
	VideoStatusProcessing = "processing"
	VideoStatusCompleted  = "completed"
	VideoStatusFailed     = "failed"
)
