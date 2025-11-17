package models

import "time"

// Subtitle represents a subtitle/caption track
type Subtitle struct {
	ID        string    `json:"id" db:"id"`
	VideoID   string    `json:"video_id" db:"video_id"`
	Language  string    `json:"language" db:"language"`
	Label     string    `json:"label,omitempty" db:"label"`
	Format    string    `json:"format" db:"format"`
	URL       string    `json:"url" db:"url"`
	Path      string    `json:"path" db:"path"`
	IsDefault bool      `json:"is_default" db:"is_default"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// SubtitleFormat constants
const (
	SubtitleFormatVTT = "vtt"
	SubtitleFormatSRT = "srt"
	SubtitleFormatASS = "ass"
)
