package models

import "time"

// Thumbnail represents a video thumbnail
type Thumbnail struct {
	ID             string    `json:"id" db:"id"`
	VideoID        string    `json:"video_id" db:"video_id"`
	ThumbnailType  string    `json:"thumbnail_type" db:"thumbnail_type"`
	URL            string    `json:"url" db:"url"`
	Path           string    `json:"path" db:"path"`
	Width          int       `json:"width" db:"width"`
	Height         int       `json:"height" db:"height"`
	Timestamp      *float64  `json:"timestamp,omitempty" db:"timestamp"`
	SpriteColumns  *int      `json:"sprite_columns,omitempty" db:"sprite_columns"`
	SpriteRows     *int      `json:"sprite_rows,omitempty" db:"sprite_rows"`
	IntervalSeconds *float64 `json:"interval_seconds,omitempty" db:"interval_seconds"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
}

// ThumbnailType constants
const (
	ThumbnailTypeSingle   = "single"
	ThumbnailTypeSprite   = "sprite"
	ThumbnailTypeAnimated = "animated"
)
