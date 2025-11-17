package models

import (
	"time"
)

// Output represents a transcoded output file
type Output struct {
	ID         string    `json:"id" db:"id"`
	JobID      string    `json:"job_id" db:"job_id"`
	VideoID    string    `json:"video_id" db:"video_id"`
	Format     string    `json:"format" db:"format"`
	Resolution string    `json:"resolution" db:"resolution"`
	Width      int       `json:"width" db:"width"`
	Height     int       `json:"height" db:"height"`
	Codec      string    `json:"codec" db:"codec"`
	Bitrate    int64     `json:"bitrate" db:"bitrate"`
	Size       int64     `json:"size" db:"size"`
	Duration   float64   `json:"duration" db:"duration"`
	URL        string    `json:"url" db:"url"`
	Path       string    `json:"path" db:"path"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}
