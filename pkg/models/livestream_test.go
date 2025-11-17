package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLiveStreamSettings_ValueScan(t *testing.T) {
	settings := LiveStreamSettings{
		EnableTranscoding: true,
		Resolutions:       []string{"1080p", "720p", "480p"},
		Codec:             "h264",
		SegmentDuration:   6,
		PlaylistLength:    10,
		PartDuration:      0.5,
		AudioCodec:        "aac",
		AudioBitrate:      128,
		KeyframeInterval:  2,
		GPUAcceleration:   true,
	}

	// Test Value (marshaling)
	value, err := settings.Value()
	require.NoError(t, err)
	assert.NotNil(t, value)

	// Test Scan (unmarshaling)
	var scanned LiveStreamSettings
	err = scanned.Scan(value)
	require.NoError(t, err)
	assert.Equal(t, settings.EnableTranscoding, scanned.EnableTranscoding)
	assert.Equal(t, settings.Resolutions, scanned.Resolutions)
	assert.Equal(t, settings.Codec, scanned.Codec)
	assert.Equal(t, settings.SegmentDuration, scanned.SegmentDuration)
}

func TestLiveStreamSettings_ScanNil(t *testing.T) {
	var settings LiveStreamSettings
	err := settings.Scan(nil)
	require.NoError(t, err)
	assert.Equal(t, LiveStreamSettings{}, settings)
}

func TestDefaultLiveStreamSettings(t *testing.T) {
	settings := DefaultLiveStreamSettings()

	assert.True(t, settings.EnableTranscoding)
	assert.Equal(t, []string{"1080p", "720p", "480p", "360p"}, settings.Resolutions)
	assert.Equal(t, "h264", settings.Codec)
	assert.Equal(t, 6, settings.SegmentDuration)
	assert.Equal(t, 10, settings.PlaylistLength)
	assert.Equal(t, 0.5, settings.PartDuration)
	assert.Equal(t, "aac", settings.AudioCodec)
	assert.Equal(t, 128, settings.AudioBitrate)
	assert.Equal(t, 2, settings.KeyframeInterval)
	assert.True(t, settings.GPUAcceleration)
}

func TestLiveStreamStatus(t *testing.T) {
	statuses := []string{
		LiveStreamStatusIdle,
		LiveStreamStatusStarting,
		LiveStreamStatusLive,
		LiveStreamStatusEnding,
		LiveStreamStatusEnded,
		LiveStreamStatusFailed,
	}

	assert.Equal(t, "idle", statuses[0])
	assert.Equal(t, "starting", statuses[1])
	assert.Equal(t, "live", statuses[2])
	assert.Equal(t, "ending", statuses[3])
	assert.Equal(t, "ended", statuses[4])
	assert.Equal(t, "failed", statuses[5])
}

func TestDVRRecordingStatus(t *testing.T) {
	statuses := []string{
		DVRRecordingStatusRecording,
		DVRRecordingStatusProcessing,
		DVRRecordingStatusAvailable,
		DVRRecordingStatusArchived,
		DVRRecordingStatusFailed,
	}

	assert.Equal(t, "recording", statuses[0])
	assert.Equal(t, "processing", statuses[1])
	assert.Equal(t, "available", statuses[2])
	assert.Equal(t, "archived", statuses[3])
	assert.Equal(t, "failed", statuses[4])
}

func TestLiveStreamEventTypes(t *testing.T) {
	eventTypes := []string{
		LiveStreamEventStreamStarted,
		LiveStreamEventStreamEnded,
		LiveStreamEventQualityChanged,
		LiveStreamEventBufferUnderflow,
		LiveStreamEventConnectionLost,
		LiveStreamEventConnectionRestored,
		LiveStreamEventHighLatency,
		LiveStreamEventFrameDrop,
		LiveStreamEventBitrateChange,
		LiveStreamEventError,
	}

	assert.Len(t, eventTypes, 10)
	assert.Contains(t, eventTypes, "stream_started")
	assert.Contains(t, eventTypes, "error")
}

func TestEventSeverity(t *testing.T) {
	severities := []string{
		SeverityInfo,
		SeverityWarning,
		SeverityError,
		SeverityCritical,
	}

	assert.Equal(t, "info", severities[0])
	assert.Equal(t, "warning", severities[1])
	assert.Equal(t, "error", severities[2])
	assert.Equal(t, "critical", severities[3])
}

func TestLiveStream_JSON(t *testing.T) {
	now := time.Now()
	stream := &LiveStream{
		ID:              "stream-123",
		Title:           "Test Stream",
		Description:     "A test live stream",
		UserID:          "user-456",
		StreamKey:       "key-789",
		RTMPIngestURL:   "rtmp://localhost:1935/live/key-789",
		Status:          LiveStreamStatusLive,
		MasterPlaylist:  "/streams/stream-123/master.m3u8",
		ViewerCount:     100,
		PeakViewerCount: 250,
		DVREnabled:      true,
		DVRWindow:       7200,
		LowLatency:      true,
		Settings:        DefaultLiveStreamSettings(),
		Metadata:        Metadata{"key": "value"},
		StartedAt:       &now,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	// Marshal to JSON
	data, err := json.Marshal(stream)
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	// Unmarshal from JSON
	var unmarshaled LiveStream
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, stream.ID, unmarshaled.ID)
	assert.Equal(t, stream.Title, unmarshaled.Title)
	assert.Equal(t, stream.ViewerCount, unmarshaled.ViewerCount)
	assert.Equal(t, stream.DVREnabled, unmarshaled.DVREnabled)
}

func TestLiveStreamVariant_JSON(t *testing.T) {
	variant := &LiveStreamVariant{
		ID:             "variant-123",
		LiveStreamID:   "stream-456",
		Resolution:     "1080p",
		Width:          1920,
		Height:         1080,
		Bitrate:        5000000,
		FrameRate:      30.0,
		Codec:          "h264",
		AudioBitrate:   128,
		PlaylistURL:    "/streams/variant-123/playlist.m3u8",
		SegmentPattern: "/streams/variant-123/%03d.ts",
		CreatedAt:      time.Now(),
	}

	data, err := json.Marshal(variant)
	require.NoError(t, err)

	var unmarshaled LiveStreamVariant
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, variant.Resolution, unmarshaled.Resolution)
	assert.Equal(t, variant.Width, unmarshaled.Width)
	assert.Equal(t, variant.Height, unmarshaled.Height)
	assert.Equal(t, variant.Bitrate, unmarshaled.Bitrate)
}

func TestDVRRecording_JSON(t *testing.T) {
	now := time.Now()
	videoID := "video-123"
	retention := now.Add(7 * 24 * time.Hour)

	recording := &DVRRecording{
		ID:             "recording-123",
		LiveStreamID:   "stream-456",
		VideoID:        &videoID,
		StartTime:      now.Add(-1 * time.Hour),
		EndTime:        &now,
		Duration:       3600,
		Size:           1024 * 1024 * 500, // 500 MB
		Status:         DVRRecordingStatusAvailable,
		RecordingURL:   "/recordings/recording-123.mp4",
		ManifestURL:    "/recordings/recording-123/manifest.m3u8",
		ThumbnailURL:   "/recordings/recording-123/thumb.jpg",
		RetentionUntil: &retention,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	data, err := json.Marshal(recording)
	require.NoError(t, err)

	var unmarshaled DVRRecording
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, recording.ID, unmarshaled.ID)
	assert.Equal(t, recording.Duration, unmarshaled.Duration)
	assert.Equal(t, recording.Status, unmarshaled.Status)
	assert.NotNil(t, unmarshaled.VideoID)
}

func TestLiveStreamAnalytics_JSON(t *testing.T) {
	analytics := &LiveStreamAnalytics{
		ID:               "analytics-123",
		LiveStreamID:     "stream-456",
		Timestamp:        time.Now(),
		ViewerCount:      150,
		BandwidthUsage:   50000000, // 50 Mbps
		IngestBitrate:    8000000,  // 8 Mbps
		DroppedFrames:    5,
		KeyframeInterval: 2.0,
		AudioVideoSync:   10.5,
		BufferHealth:     98.5,
		CDNHitRatio:      95.0,
		AverageLatency:   250.0,
		P95Latency:       500.0,
		ErrorCount:       2,
		QualityScore:     92.5,
	}

	data, err := json.Marshal(analytics)
	require.NoError(t, err)

	var unmarshaled LiveStreamAnalytics
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, analytics.ViewerCount, unmarshaled.ViewerCount)
	assert.Equal(t, analytics.BandwidthUsage, unmarshaled.BandwidthUsage)
	assert.Equal(t, analytics.QualityScore, unmarshaled.QualityScore)
}

func TestLiveStreamEvent_JSON(t *testing.T) {
	event := &LiveStreamEvent{
		ID:           "event-123",
		LiveStreamID: "stream-456",
		EventType:    LiveStreamEventError,
		Severity:     SeverityError,
		Message:      "Connection lost",
		Details:      Metadata{"error_code": "CONN_LOST", "retry_count": 3},
		Timestamp:    time.Now(),
	}

	data, err := json.Marshal(event)
	require.NoError(t, err)

	var unmarshaled LiveStreamEvent
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, event.EventType, unmarshaled.EventType)
	assert.Equal(t, event.Severity, unmarshaled.Severity)
	assert.Equal(t, event.Message, unmarshaled.Message)
}

func TestLiveStreamViewer_JSON(t *testing.T) {
	now := time.Now()
	userID := "user-123"
	leftAt := now.Add(30 * time.Minute)

	viewer := &LiveStreamViewer{
		ID:             "viewer-123",
		LiveStreamID:   "stream-456",
		SessionID:      "session-789",
		UserID:         &userID,
		JoinedAt:       now,
		LeftAt:         &leftAt,
		WatchDuration:  1800, // 30 minutes
		Resolution:     "1080p",
		DeviceType:     "desktop",
		Location:       "US",
		IPAddress:      "192.168.1.1",
		UserAgent:      "Mozilla/5.0",
		BufferEvents:   3,
		QualityChanges: 2,
	}

	data, err := json.Marshal(viewer)
	require.NoError(t, err)

	var unmarshaled LiveStreamViewer
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, viewer.SessionID, unmarshaled.SessionID)
	assert.Equal(t, viewer.WatchDuration, unmarshaled.WatchDuration)
	assert.Equal(t, viewer.Resolution, unmarshaled.Resolution)
	assert.NotNil(t, unmarshaled.UserID)
}
