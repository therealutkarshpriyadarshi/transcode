package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// LiveStream represents a live streaming session
type LiveStream struct {
	ID              string            `json:"id" db:"id"`
	Title           string            `json:"title" db:"title"`
	Description     string            `json:"description,omitempty" db:"description"`
	UserID          string            `json:"user_id" db:"user_id"`
	StreamKey       string            `json:"stream_key,omitempty" db:"stream_key"`
	RTMPIngestURL   string            `json:"rtmp_ingest_url" db:"rtmp_ingest_url"`
	Status          string            `json:"status" db:"status"`
	MasterPlaylist  string            `json:"master_playlist,omitempty" db:"master_playlist"`
	ViewerCount     int               `json:"viewer_count" db:"viewer_count"`
	PeakViewerCount int               `json:"peak_viewer_count" db:"peak_viewer_count"`
	DVREnabled      bool              `json:"dvr_enabled" db:"dvr_enabled"`
	DVRWindow       int               `json:"dvr_window" db:"dvr_window"` // DVR window in seconds
	LowLatency      bool              `json:"low_latency" db:"low_latency"`
	Settings        LiveStreamSettings `json:"settings" db:"settings"`
	Metadata        Metadata          `json:"metadata,omitempty" db:"metadata"`
	StartedAt       *time.Time        `json:"started_at,omitempty" db:"started_at"`
	EndedAt         *time.Time        `json:"ended_at,omitempty" db:"ended_at"`
	CreatedAt       time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at" db:"updated_at"`
}

// LiveStreamSettings holds configuration for a live stream
type LiveStreamSettings struct {
	// Transcoding settings
	EnableTranscoding bool     `json:"enable_transcoding"`
	Resolutions       []string `json:"resolutions"` // e.g., ["1080p", "720p", "480p"]
	Codec             string   `json:"codec"`       // e.g., "h264", "h265"

	// Segment settings
	SegmentDuration   int `json:"segment_duration"` // in seconds, default 6
	PlaylistLength    int `json:"playlist_length"`  // number of segments to keep

	// Low-latency settings
	PartDuration      float64 `json:"part_duration,omitempty"` // for LL-HLS, in seconds

	// Audio settings
	AudioCodec        string `json:"audio_codec"`
	AudioBitrate      int    `json:"audio_bitrate"` // in kbps

	// Advanced settings
	KeyframeInterval  int  `json:"keyframe_interval"` // in seconds
	GPUAcceleration   bool `json:"gpu_acceleration"`
}

// Value implements driver.Valuer for database storage
func (s LiveStreamSettings) Value() (driver.Value, error) {
	return json.Marshal(s)
}

// Scan implements sql.Scanner for database retrieval
func (s *LiveStreamSettings) Scan(value interface{}) error {
	if value == nil {
		*s = LiveStreamSettings{}
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}

	return json.Unmarshal(bytes, s)
}

// LiveStreamStatus constants
const (
	LiveStreamStatusIdle       = "idle"       // Created but not started
	LiveStreamStatusStarting   = "starting"   // Starting up
	LiveStreamStatusLive       = "live"       // Currently streaming
	LiveStreamStatusEnding     = "ending"     // Being stopped
	LiveStreamStatusEnded      = "ended"      // Completed
	LiveStreamStatusFailed     = "failed"     // Failed to start or crashed
)

// LiveStreamVariant represents a quality variant of a live stream
type LiveStreamVariant struct {
	ID              string    `json:"id" db:"id"`
	LiveStreamID    string    `json:"live_stream_id" db:"live_stream_id"`
	Resolution      string    `json:"resolution" db:"resolution"`       // e.g., "1080p", "720p"
	Width           int       `json:"width" db:"width"`
	Height          int       `json:"height" db:"height"`
	Bitrate         int64     `json:"bitrate" db:"bitrate"`             // in bps
	FrameRate       float64   `json:"frame_rate" db:"frame_rate"`
	Codec           string    `json:"codec" db:"codec"`
	AudioBitrate    int       `json:"audio_bitrate" db:"audio_bitrate"` // in kbps
	PlaylistURL     string    `json:"playlist_url" db:"playlist_url"`
	SegmentPattern  string    `json:"segment_pattern" db:"segment_pattern"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
}

// DVRRecording represents a recorded live stream for DVR functionality
type DVRRecording struct {
	ID             string     `json:"id" db:"id"`
	LiveStreamID   string     `json:"live_stream_id" db:"live_stream_id"`
	VideoID        *string    `json:"video_id,omitempty" db:"video_id"` // Link to VOD after conversion
	StartTime      time.Time  `json:"start_time" db:"start_time"`
	EndTime        *time.Time `json:"end_time,omitempty" db:"end_time"`
	Duration       float64    `json:"duration" db:"duration"` // in seconds
	Size           int64      `json:"size" db:"size"`         // in bytes
	Status         string     `json:"status" db:"status"`
	RecordingURL   string     `json:"recording_url,omitempty" db:"recording_url"`
	ManifestURL    string     `json:"manifest_url,omitempty" db:"manifest_url"`
	ThumbnailURL   string     `json:"thumbnail_url,omitempty" db:"thumbnail_url"`
	RetentionUntil *time.Time `json:"retention_until,omitempty" db:"retention_until"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at" db:"updated_at"`
}

// DVRRecordingStatus constants
const (
	DVRRecordingStatusRecording  = "recording"
	DVRRecordingStatusProcessing = "processing"
	DVRRecordingStatusAvailable  = "available"
	DVRRecordingStatusArchived   = "archived"
	DVRRecordingStatusFailed     = "failed"
)

// LiveStreamAnalytics represents real-time analytics for a live stream
type LiveStreamAnalytics struct {
	ID                  string    `json:"id" db:"id"`
	LiveStreamID        string    `json:"live_stream_id" db:"live_stream_id"`
	Timestamp           time.Time `json:"timestamp" db:"timestamp"`
	ViewerCount         int       `json:"viewer_count" db:"viewer_count"`
	BandwidthUsage      int64     `json:"bandwidth_usage" db:"bandwidth_usage"`         // in bps
	IngestBitrate       int64     `json:"ingest_bitrate" db:"ingest_bitrate"`           // in bps
	DroppedFrames       int       `json:"dropped_frames" db:"dropped_frames"`
	KeyframeInterval    float64   `json:"keyframe_interval" db:"keyframe_interval"`     // in seconds
	AudioVideoSync      float64   `json:"audio_video_sync" db:"audio_video_sync"`       // in ms
	BufferHealth        float64   `json:"buffer_health" db:"buffer_health"`             // percentage
	CDNHitRatio         float64   `json:"cdn_hit_ratio,omitempty" db:"cdn_hit_ratio"`   // percentage
	AverageLatency      float64   `json:"average_latency" db:"average_latency"`         // in ms
	P95Latency          float64   `json:"p95_latency,omitempty" db:"p95_latency"`       // in ms
	ErrorCount          int       `json:"error_count" db:"error_count"`
	QualityScore        float64   `json:"quality_score,omitempty" db:"quality_score"`   // 0-100
}

// LiveStreamEvent represents significant events during a live stream
type LiveStreamEvent struct {
	ID           string    `json:"id" db:"id"`
	LiveStreamID string    `json:"live_stream_id" db:"live_stream_id"`
	EventType    string    `json:"event_type" db:"event_type"`
	Severity     string    `json:"severity" db:"severity"` // info, warning, error, critical
	Message      string    `json:"message" db:"message"`
	Details      Metadata  `json:"details,omitempty" db:"details"`
	Timestamp    time.Time `json:"timestamp" db:"timestamp"`
}

// LiveStreamEventType constants
const (
	LiveStreamEventStreamStarted      = "stream_started"
	LiveStreamEventStreamEnded        = "stream_ended"
	LiveStreamEventQualityChanged     = "quality_changed"
	LiveStreamEventBufferUnderflow    = "buffer_underflow"
	LiveStreamEventConnectionLost     = "connection_lost"
	LiveStreamEventConnectionRestored = "connection_restored"
	LiveStreamEventHighLatency        = "high_latency"
	LiveStreamEventFrameDrop          = "frame_drop"
	LiveStreamEventBitrateChange      = "bitrate_change"
	LiveStreamEventError              = "error"
)

// EventSeverity constants
const (
	SeverityInfo     = "info"
	SeverityWarning  = "warning"
	SeverityError    = "error"
	SeverityCritical = "critical"
)

// LiveStreamViewer represents a viewer watching a live stream
type LiveStreamViewer struct {
	ID            string    `json:"id" db:"id"`
	LiveStreamID  string    `json:"live_stream_id" db:"live_stream_id"`
	SessionID     string    `json:"session_id" db:"session_id"`
	UserID        *string   `json:"user_id,omitempty" db:"user_id"`
	JoinedAt      time.Time `json:"joined_at" db:"joined_at"`
	LeftAt        *time.Time `json:"left_at,omitempty" db:"left_at"`
	WatchDuration float64   `json:"watch_duration" db:"watch_duration"` // in seconds
	Resolution    string    `json:"resolution,omitempty" db:"resolution"`
	DeviceType    string    `json:"device_type,omitempty" db:"device_type"`
	Location      string    `json:"location,omitempty" db:"location"`
	IPAddress     string    `json:"ip_address,omitempty" db:"ip_address"`
	UserAgent     string    `json:"user_agent,omitempty" db:"user_agent"`
	BufferEvents  int       `json:"buffer_events" db:"buffer_events"`
	QualityChanges int      `json:"quality_changes" db:"quality_changes"`
}

// DefaultLiveStreamSettings returns default settings for a live stream
func DefaultLiveStreamSettings() LiveStreamSettings {
	return LiveStreamSettings{
		EnableTranscoding: true,
		Resolutions:       []string{"1080p", "720p", "480p", "360p"},
		Codec:             "h264",
		SegmentDuration:   6,
		PlaylistLength:    10,
		PartDuration:      0.5, // for LL-HLS
		AudioCodec:        "aac",
		AudioBitrate:      128,
		KeyframeInterval:  2,
		GPUAcceleration:   true,
	}
}
