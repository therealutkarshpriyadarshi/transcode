package models

import (
	"time"
)

// PlaybackEvent represents a single playback event
type PlaybackEvent struct {
	ID           string    `json:"id" db:"id"`
	VideoID      string    `json:"video_id" db:"video_id"`
	OutputID     string    `json:"output_id,omitempty" db:"output_id"` // Which output variant was played
	SessionID    string    `json:"session_id" db:"session_id"`          // Unique session identifier
	UserID       string    `json:"user_id,omitempty" db:"user_id"`      // Optional user ID
	EventType    string    `json:"event_type" db:"event_type"`          // play, pause, seek, buffer, complete, error
	Timestamp    time.Time `json:"timestamp" db:"timestamp"`
	Position     float64   `json:"position" db:"position"`                         // Current playback position in seconds
	Duration     float64   `json:"duration,omitempty" db:"duration"`               // Total video duration
	BufferTime   float64   `json:"buffer_time,omitempty" db:"buffer_time"`         // Time spent buffering in seconds
	Bitrate      int64     `json:"bitrate,omitempty" db:"bitrate"`                 // Current bitrate
	Resolution   string    `json:"resolution,omitempty" db:"resolution"`           // Current resolution
	DeviceType   string    `json:"device_type,omitempty" db:"device_type"`         // mobile, desktop, tablet, tv
	Browser      string    `json:"browser,omitempty" db:"browser"`                 // Browser/player name
	OS           string    `json:"os,omitempty" db:"os"`                           // Operating system
	Country      string    `json:"country,omitempty" db:"country"`                 // Geographic location
	IPAddress    string    `json:"ip_address,omitempty" db:"ip_address"`           // Client IP
	CDNNode      string    `json:"cdn_node,omitempty" db:"cdn_node"`               // CDN node serving the video
	ErrorCode    string    `json:"error_code,omitempty" db:"error_code"`           // Error code if event_type is error
	ErrorMessage string    `json:"error_message,omitempty" db:"error_message"`     // Error message
}

// PlaybackEventType constants
const (
	EventTypePlay     = "play"
	EventTypePause    = "pause"
	EventTypeSeek     = "seek"
	EventTypeBuffer   = "buffer"
	EventTypeComplete = "complete"
	EventTypeError    = "error"
	EventTypeQualityChange = "quality_change"
)

// PlaybackSession represents an aggregated playback session
type PlaybackSession struct {
	ID                string    `json:"id" db:"id"`
	VideoID           string    `json:"video_id" db:"video_id"`
	UserID            string    `json:"user_id,omitempty" db:"user_id"`
	StartTime         time.Time `json:"start_time" db:"start_time"`
	EndTime           *time.Time `json:"end_time,omitempty" db:"end_time"`
	Duration          float64   `json:"duration" db:"duration"`                     // Session duration in seconds
	WatchTime         float64   `json:"watch_time" db:"watch_time"`                 // Actual watch time (excluding pauses)
	CompletionRate    float64   `json:"completion_rate" db:"completion_rate"`       // Percentage watched (0-100)
	TotalBufferTime   float64   `json:"total_buffer_time" db:"total_buffer_time"`   // Total buffering time
	BufferCount       int       `json:"buffer_count" db:"buffer_count"`             // Number of buffering events
	SeekCount         int       `json:"seek_count" db:"seek_count"`                 // Number of seeks
	QualityChanges    int       `json:"quality_changes" db:"quality_changes"`       // Number of quality switches
	AverageBitrate    int64     `json:"average_bitrate" db:"average_bitrate"`       // Average bitrate during session
	PeakBitrate       int64     `json:"peak_bitrate" db:"peak_bitrate"`             // Peak bitrate
	StartupTime       float64   `json:"startup_time" db:"startup_time"`             // Time to first frame
	DeviceType        string    `json:"device_type" db:"device_type"`
	Browser           string    `json:"browser" db:"browser"`
	OS                string    `json:"os" db:"os"`
	Country           string    `json:"country" db:"country"`
	Completed         bool      `json:"completed" db:"completed"`                   // Whether video was fully watched
	ErrorOccurred     bool      `json:"error_occurred" db:"error_occurred"`
}

// VideoAnalytics represents aggregated analytics for a video
type VideoAnalytics struct {
	VideoID            string    `json:"video_id" db:"video_id"`
	TotalViews         int64     `json:"total_views" db:"total_views"`
	UniqueViewers      int64     `json:"unique_viewers" db:"unique_viewers"`
	TotalWatchTime     float64   `json:"total_watch_time" db:"total_watch_time"`        // Total watch time in hours
	AverageWatchTime   float64   `json:"average_watch_time" db:"average_watch_time"`    // Average watch time per session
	CompletionRate     float64   `json:"completion_rate" db:"completion_rate"`          // Average completion rate
	AverageBufferTime  float64   `json:"average_buffer_time" db:"average_buffer_time"`  // Average buffering time
	BufferRate         float64   `json:"buffer_rate" db:"buffer_rate"`                  // Percentage of sessions with buffering
	ErrorRate          float64   `json:"error_rate" db:"error_rate"`                    // Percentage of sessions with errors
	AverageStartupTime float64   `json:"average_startup_time" db:"average_startup_time"` // Average time to first frame
	PopularResolutions map[string]int64 `json:"popular_resolutions" db:"popular_resolutions"` // Resolution usage statistics
	GeographicData     map[string]int64 `json:"geographic_data" db:"geographic_data"`         // Views by country
	DeviceBreakdown    map[string]int64 `json:"device_breakdown" db:"device_breakdown"`       // Views by device type
	PeakViewingTime    time.Time `json:"peak_viewing_time" db:"peak_viewing_time"`       // Time with most concurrent viewers
	LastUpdated        time.Time `json:"last_updated" db:"last_updated"`
}

// QualityOfExperience (QoE) metrics
type QoEMetrics struct {
	VideoID            string    `json:"video_id" db:"video_id"`
	OutputID           string    `json:"output_id,omitempty" db:"output_id"`
	Period             string    `json:"period" db:"period"`                        // hourly, daily, weekly
	Timestamp          time.Time `json:"timestamp" db:"timestamp"`
	ViewCount          int64     `json:"view_count" db:"view_count"`
	AverageQoE         float64   `json:"average_qoe" db:"average_qoe"`             // Overall QoE score (0-100)
	RebufferRatio      float64   `json:"rebuffer_ratio" db:"rebuffer_ratio"`       // Rebuffering time / play time
	StartupTime        float64   `json:"startup_time" db:"startup_time"`           // Average startup time
	BitrateUtilization float64   `json:"bitrate_utilization" db:"bitrate_utilization"` // Average bitrate / available bandwidth
	ErrorRate          float64   `json:"error_rate" db:"error_rate"`
	CompletionRate     float64   `json:"completion_rate" db:"completion_rate"`
}

// BandwidthUsage tracks bandwidth consumption
type BandwidthUsage struct {
	ID         string    `json:"id" db:"id"`
	VideoID    string    `json:"video_id" db:"video_id"`
	OutputID   string    `json:"output_id" db:"output_id"`
	Timestamp  time.Time `json:"timestamp" db:"timestamp"`
	BytesServed int64    `json:"bytes_served" db:"bytes_served"` // Total bytes served
	RequestCount int64   `json:"request_count" db:"request_count"` // Number of requests
	CDNNode    string    `json:"cdn_node,omitempty" db:"cdn_node"`
	Country    string    `json:"country,omitempty" db:"country"`
}

// AnalyticsAggregation represents different time period aggregations
type AnalyticsAggregation struct {
	Period    string    `json:"period"` // hourly, daily, weekly, monthly
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
	Metrics   VideoAnalytics `json:"metrics"`
}

// TrendingVideo represents a trending video based on recent engagement
type TrendingVideo struct {
	VideoID        string    `json:"video_id"`
	Title          string    `json:"title"`
	Views          int64     `json:"views"`
	ViewGrowth     float64   `json:"view_growth"`      // Percentage growth
	TrendingScore  float64   `json:"trending_score"`   // Composite score for ranking
	Category       string    `json:"category,omitempty"`
	LastUpdated    time.Time `json:"last_updated"`
}

// HeatmapData represents viewer engagement heatmap data
type HeatmapData struct {
	VideoID    string              `json:"video_id"`
	Resolution int                 `json:"resolution"` // Time resolution in seconds
	Data       []HeatmapDataPoint  `json:"data"`
}

// HeatmapDataPoint represents engagement at a specific time point
type HeatmapDataPoint struct {
	Timestamp float64 `json:"timestamp"` // Video timestamp
	ViewCount int64   `json:"view_count"` // Number of viewers at this point
	SeekCount int64   `json:"seek_count"` // Number of seeks to this point
}
