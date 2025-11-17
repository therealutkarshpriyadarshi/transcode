package models

import "time"

// StreamingProfile represents an adaptive streaming profile (HLS/DASH)
type StreamingProfile struct {
	ID                 string    `json:"id" db:"id"`
	VideoID            string    `json:"video_id" db:"video_id"`
	JobID              *string   `json:"job_id,omitempty" db:"job_id"`
	ProfileType        string    `json:"profile_type" db:"profile_type"`
	MasterManifestURL  string    `json:"master_manifest_url" db:"master_manifest_url"`
	MasterManifestPath string    `json:"master_manifest_path" db:"master_manifest_path"`
	VariantCount       int       `json:"variant_count" db:"variant_count"`
	AudioOnly          bool      `json:"audio_only" db:"audio_only"`
	CreatedAt          time.Time `json:"created_at" db:"created_at"`
}

// AudioTrack represents an audio track for streaming
type AudioTrack struct {
	ID                  string    `json:"id" db:"id"`
	VideoID             string    `json:"video_id" db:"video_id"`
	StreamingProfileID  *string   `json:"streaming_profile_id,omitempty" db:"streaming_profile_id"`
	Language            string    `json:"language" db:"language"`
	Label               string    `json:"label,omitempty" db:"label"`
	Codec               string    `json:"codec" db:"codec"`
	Bitrate             int       `json:"bitrate" db:"bitrate"`
	Channels            int       `json:"channels" db:"channels"`
	SampleRate          int       `json:"sample_rate" db:"sample_rate"`
	URL                 string    `json:"url" db:"url"`
	Path                string    `json:"path" db:"path"`
	IsDefault           bool      `json:"is_default" db:"is_default"`
	CreatedAt           time.Time `json:"created_at" db:"created_at"`
}

// StreamingType constants
const (
	StreamingTypeProgressive = "progressive"
	StreamingTypeHLS         = "hls"
	StreamingTypeDASH        = "dash"
)

// ProfileType constants
const (
	ProfileTypeHLS  = "hls"
	ProfileTypeDASH = "dash"
)
