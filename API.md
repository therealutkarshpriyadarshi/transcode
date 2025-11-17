# API Documentation

## Base URL

```
http://localhost:8080/api/v1
```

## Authentication

Currently, the API does not require authentication. This will be added in Phase 3.

## Endpoints

### Health Check

Check if the API server is healthy.

**Endpoint**: `GET /health`

**Response**:
```json
{
  "status": "healthy"
}
```

---

### Videos

#### Upload Video

Upload a video file for processing.

**Endpoint**: `POST /api/v1/videos/upload`

**Content-Type**: `multipart/form-data`

**Parameters**:
- `video` (file, required): Video file to upload

**Example**:
```bash
curl -X POST http://localhost:8080/api/v1/videos/upload \
  -F "video=@/path/to/video.mp4"
```

**Response** (201 Created):
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "filename": "video.mp4",
  "original_url": "videos/550e8400-e29b-41d4-a716-446655440000/original/video.mp4",
  "size": 10485760,
  "duration": 120.5,
  "width": 1920,
  "height": 1080,
  "codec": "h264",
  "bitrate": 5000000,
  "frame_rate": 30.0,
  "metadata": {},
  "status": "pending",
  "created_at": "2025-01-17T10:00:00Z",
  "updated_at": "2025-01-17T10:00:00Z"
}
```

---

#### Get Video

Retrieve video details by ID.

**Endpoint**: `GET /api/v1/videos/:id`

**Parameters**:
- `id` (path, required): Video ID

**Example**:
```bash
curl http://localhost:8080/api/v1/videos/550e8400-e29b-41d4-a716-446655440000
```

**Response** (200 OK):
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "filename": "video.mp4",
  "original_url": "videos/550e8400-e29b-41d4-a716-446655440000/original/video.mp4",
  "size": 10485760,
  "duration": 120.5,
  "width": 1920,
  "height": 1080,
  "codec": "h264",
  "bitrate": 5000000,
  "frame_rate": 30.0,
  "metadata": {},
  "status": "completed",
  "created_at": "2025-01-17T10:00:00Z",
  "updated_at": "2025-01-17T10:05:00Z"
}
```

**Error Response** (404 Not Found):
```json
{
  "error": "Video not found"
}
```

---

#### List Videos

List all videos with pagination.

**Endpoint**: `GET /api/v1/videos`

**Query Parameters**:
- `limit` (optional, default: 20): Number of videos to return
- `offset` (optional, default: 0): Pagination offset

**Example**:
```bash
curl "http://localhost:8080/api/v1/videos?limit=10&offset=0"
```

**Response** (200 OK):
```json
{
  "videos": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "filename": "video.mp4",
      "size": 10485760,
      "duration": 120.5,
      "width": 1920,
      "height": 1080,
      "status": "completed",
      "created_at": "2025-01-17T10:00:00Z"
    }
  ],
  "limit": 10,
  "offset": 0
}
```

---

### Jobs

#### Create Transcode Job

Create a new transcoding job for a video.

**Endpoint**: `POST /api/v1/videos/:id/transcode`

**Content-Type**: `application/json`

**Parameters**:
- `id` (path, required): Video ID
- `resolution` (body, required): Target resolution (144p, 240p, 360p, 480p, 720p, 1080p, 1440p, 4k)
- `output_format` (body, optional): Output format (mp4, webm, mkv)
- `codec` (body, optional): Video codec (libx264, libx265, libvpx-vp9)
- `bitrate` (body, optional): Video bitrate in bits/sec
- `preset` (body, optional): FFmpeg preset (ultrafast, superfast, veryfast, faster, fast, medium, slow, slower, veryslow)
- `priority` (body, optional): Job priority (0=low, 5=normal, 10=high)

**Example**:
```bash
curl -X POST http://localhost:8080/api/v1/videos/550e8400-e29b-41d4-a716-446655440000/transcode \
  -H "Content-Type: application/json" \
  -d '{
    "resolution": "720p",
    "output_format": "mp4",
    "codec": "libx264",
    "bitrate": 2500000,
    "preset": "medium",
    "priority": 5
  }'
```

**Response** (201 Created):
```json
{
  "id": "660e8400-e29b-41d4-a716-446655440001",
  "video_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "queued",
  "priority": 5,
  "progress": 0.0,
  "error_msg": "",
  "retry_count": 0,
  "created_at": "2025-01-17T10:01:00Z",
  "updated_at": "2025-01-17T10:01:00Z",
  "config": {
    "output_format": "mp4",
    "resolution": "720p",
    "codec": "libx264",
    "bitrate": 2500000,
    "preset": "medium",
    "audio_codec": "aac",
    "audio_bitrate": 128
  }
}
```

---

#### Get Job

Retrieve job details and progress.

**Endpoint**: `GET /api/v1/jobs/:id`

**Parameters**:
- `id` (path, required): Job ID

**Example**:
```bash
curl http://localhost:8080/api/v1/jobs/660e8400-e29b-41d4-a716-446655440001
```

**Response** (200 OK):
```json
{
  "id": "660e8400-e29b-41d4-a716-446655440001",
  "video_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "processing",
  "priority": 5,
  "progress": 45.5,
  "error_msg": "",
  "retry_count": 0,
  "worker_id": "770e8400-e29b-41d4-a716-446655440002",
  "started_at": "2025-01-17T10:02:00Z",
  "created_at": "2025-01-17T10:01:00Z",
  "updated_at": "2025-01-17T10:03:00Z",
  "config": {
    "output_format": "mp4",
    "resolution": "720p",
    "codec": "libx264",
    "bitrate": 2500000
  }
}
```

---

#### Get Video Jobs

List all jobs for a specific video.

**Endpoint**: `GET /api/v1/videos/:id/jobs`

**Parameters**:
- `id` (path, required): Video ID

**Example**:
```bash
curl http://localhost:8080/api/v1/videos/550e8400-e29b-41d4-a716-446655440000/jobs
```

**Response** (200 OK):
```json
{
  "jobs": [
    {
      "id": "660e8400-e29b-41d4-a716-446655440001",
      "video_id": "550e8400-e29b-41d4-a716-446655440000",
      "status": "completed",
      "priority": 5,
      "progress": 100.0,
      "config": {
        "resolution": "720p"
      },
      "completed_at": "2025-01-17T10:05:00Z"
    },
    {
      "id": "660e8400-e29b-41d4-a716-446655440002",
      "video_id": "550e8400-e29b-41d4-a716-446655440000",
      "status": "processing",
      "priority": 5,
      "progress": 25.0,
      "config": {
        "resolution": "480p"
      }
    }
  ]
}
```

---

### Outputs

#### Get Video Outputs

List all transcoded outputs for a video.

**Endpoint**: `GET /api/v1/videos/:id/outputs`

**Parameters**:
- `id` (path, required): Video ID

**Example**:
```bash
curl http://localhost:8080/api/v1/videos/550e8400-e29b-41d4-a716-446655440000/outputs
```

**Response** (200 OK):
```json
{
  "outputs": [
    {
      "id": "770e8400-e29b-41d4-a716-446655440003",
      "job_id": "660e8400-e29b-41d4-a716-446655440001",
      "video_id": "550e8400-e29b-41d4-a716-446655440000",
      "format": "mp4",
      "resolution": "720p",
      "width": 1280,
      "height": 720,
      "codec": "h264",
      "bitrate": 2500000,
      "size": 5242880,
      "duration": 120.5,
      "url": "https://storage.example.com/videos/550e8400.../outputs/output_720p.mp4",
      "path": "videos/550e8400-e29b-41d4-a716-446655440000/outputs/output_720p.mp4",
      "created_at": "2025-01-17T10:05:00Z"
    }
  ]
}
```

---

## Job Status Values

- `pending`: Job created but not yet queued
- `queued`: Job queued for processing
- `processing`: Job currently being processed
- `completed`: Job completed successfully
- `failed`: Job failed with error
- `cancelled`: Job was cancelled

## Video Status Values

- `pending`: Video uploaded but no jobs created
- `processing`: One or more jobs are processing
- `completed`: All jobs completed successfully
- `failed`: One or more jobs failed

## Resolution Values

- `144p`: 256x144
- `240p`: 426x240
- `360p`: 640x360
- `480p`: 854x480
- `720p`: 1280x720
- `1080p`: 1920x1080
- `1440p`: 2560x1440
- `4k` or `2160p`: 3840x2160

## Error Responses

All error responses follow this format:

```json
{
  "error": "Error message description"
}
```

### Common HTTP Status Codes

- `200 OK`: Request successful
- `201 Created`: Resource created successfully
- `400 Bad Request`: Invalid request parameters
- `404 Not Found`: Resource not found
- `500 Internal Server Error`: Server error
- `503 Service Unavailable`: Service temporarily unavailable

## Rate Limiting

Rate limiting will be implemented in Phase 3. Current implementation has no rate limits.

## Examples

### Complete Workflow Example

```bash
# 1. Upload a video
VIDEO_ID=$(curl -X POST http://localhost:8080/api/v1/videos/upload \
  -F "video=@sample.mp4" | jq -r '.id')

echo "Uploaded video: $VIDEO_ID"

# 2. Create transcoding jobs for multiple resolutions
curl -X POST "http://localhost:8080/api/v1/videos/$VIDEO_ID/transcode" \
  -H "Content-Type: application/json" \
  -d '{"resolution": "720p", "codec": "libx264", "preset": "medium"}'

curl -X POST "http://localhost:8080/api/v1/videos/$VIDEO_ID/transcode" \
  -H "Content-Type: application/json" \
  -d '{"resolution": "480p", "codec": "libx264", "preset": "fast"}'

# 3. Check job status
sleep 5
curl "http://localhost:8080/api/v1/videos/$VIDEO_ID/jobs" | jq

# 4. Wait for completion and get outputs
sleep 60
curl "http://localhost:8080/api/v1/videos/$VIDEO_ID/outputs" | jq
```

---

## Phase 7: Advanced Features

### Scene Detection

Detect scene changes and generate intelligent thumbnails.

**Endpoint**: `POST /api/v1/videos/:id/scenes/detect`

**Request Body**:
```json
{
  "threshold": 0.4,
  "min_scene_duration": 1.0,
  "max_scenes": 20
}
```

**Response** (200 OK):
```json
{
  "total_scenes": 15,
  "scenes": [
    {
      "scene_number": 1,
      "start_time": 0.0,
      "end_time": 5.2,
      "duration": 5.2,
      "frame_path": "/tmp/scenes/scene_001.jpg"
    }
  ],
  "best_scene": {
    "scene_number": 5,
    "start_time": 20.1,
    "end_time": 30.8,
    "duration": 10.7,
    "frame_path": "/tmp/scenes/scene_005.jpg"
  }
}
```

---

### Watermarking

Apply text or image watermarks to videos.

**Endpoint**: `POST /api/v1/videos/:id/watermark`

**Request Body (Text Watermark)**:
```json
{
  "watermark_text": "© 2025 My Company",
  "position": "bottom-right",
  "opacity": 0.8,
  "font_size": 24,
  "font_color": "white",
  "padding": 10,
  "output_format": "mp4"
}
```

**Request Body (Image Watermark)**:
```json
{
  "watermark_image": "https://example.com/logo.png",
  "position": "top-right",
  "opacity": 0.7,
  "scale": 0.15,
  "padding": 20,
  "output_format": "mp4"
}
```

**Response** (200 OK):
```json
{
  "message": "watermark applied successfully",
  "url": "https://storage.example.com/videos/123/watermarked/output.mp4",
  "path": "videos/123/watermarked/output.mp4"
}
```

---

### Video Concatenation

Concatenate multiple videos into one.

**Endpoint**: `POST /api/v1/videos/concatenate`

**Request Body**:
```json
{
  "video_ids": ["video-1", "video-2", "video-3"],
  "method": "filter",
  "transition_type": "fade",
  "transition_duration": 1.0,
  "re_encode": true
}
```

**Parameters**:
- `video_ids` (array, required): Array of video IDs to concatenate
- `method` (string): "concat" (fast, no re-encoding) or "filter" (slower, supports transitions)
- `transition_type` (string): "none", "fade", or "dissolve"
- `transition_duration` (number): Transition duration in seconds
- `re_encode` (boolean): Force re-encoding

**Response** (200 OK):
```json
{
  "message": "videos concatenated successfully",
  "url": "https://storage.example.com/videos/concatenated/abc123.mp4",
  "path": "videos/concatenated/abc123.mp4"
}
```

---

### Analytics

#### Track Playback Event

Track individual playback events.

**Endpoint**: `POST /api/v1/analytics/events`

**Request Body**:
```json
{
  "video_id": "video-123",
  "session_id": "session-456",
  "user_id": "user-789",
  "event_type": "play",
  "position": 10.5,
  "duration": 120.0,
  "bitrate": 5000000,
  "resolution": "1080p",
  "device_type": "desktop",
  "browser": "Chrome",
  "os": "Windows",
  "country": "US"
}
```

**Event Types**:
- `play`: Video started playing
- `pause`: Video paused
- `seek`: User seeked to a position
- `buffer`: Video buffering occurred
- `complete`: Video playback completed
- `error`: Playback error occurred
- `quality_change`: Video quality changed

**Response** (200 OK):
```json
{
  "message": "event tracked successfully"
}
```

---

#### Start Playback Session

Start a new playback session.

**Endpoint**: `POST /api/v1/analytics/sessions/:id/start`

**Request Body**:
```json
{
  "user_id": "user-123",
  "device_info": {
    "device_type": "mobile",
    "browser": "Safari",
    "os": "iOS",
    "country": "US"
  }
}
```

**Response** (200 OK):
```json
{
  "id": "session-456",
  "video_id": "video-123",
  "user_id": "user-123",
  "start_time": "2025-01-17T10:00:00Z",
  "device_type": "mobile",
  "browser": "Safari",
  "os": "iOS",
  "country": "US"
}
```

---

#### End Playback Session

End a playback session and calculate metrics.

**Endpoint**: `POST /api/v1/analytics/sessions/:session_id/end`

**Response** (200 OK):
```json
{
  "message": "session ended successfully"
}
```

---

#### Get Video Analytics

Get aggregated analytics for a video.

**Endpoint**: `GET /api/v1/analytics/videos/:id`

**Response** (200 OK):
```json
{
  "video_id": "video-123",
  "total_views": 1000,
  "unique_viewers": 750,
  "total_watch_time": 5000.5,
  "average_watch_time": 5.0,
  "completion_rate": 75.5,
  "average_buffer_time": 2.5,
  "buffer_rate": 15.5,
  "error_rate": 2.5,
  "average_startup_time": 1.8,
  "popular_resolutions": {
    "1080p": 600,
    "720p": 300,
    "480p": 100
  },
  "geographic_data": {
    "US": 500,
    "CA": 200,
    "UK": 150,
    "DE": 150
  },
  "device_breakdown": {
    "desktop": 600,
    "mobile": 300,
    "tablet": 100
  },
  "last_updated": "2025-01-17T12:00:00Z"
}
```

---

#### Get Video Heatmap

Get viewer engagement heatmap data.

**Endpoint**: `GET /api/v1/analytics/videos/:id/heatmap?resolution=10`

**Query Parameters**:
- `resolution` (integer): Time resolution in seconds (default: 10)

**Response** (200 OK):
```json
{
  "video_id": "video-123",
  "resolution": 10,
  "data": [
    {
      "timestamp": 0,
      "view_count": 1000,
      "seek_count": 50
    },
    {
      "timestamp": 10,
      "view_count": 950,
      "seek_count": 20
    }
  ]
}
```

---

#### Get QoE Metrics

Get Quality of Experience metrics for a video.

**Endpoint**: `GET /api/v1/analytics/videos/:id/qoe`

**Query Parameters**:
- `period` (string): "hourly", "daily", "weekly", or "monthly" (default: "daily")
- `start` (ISO 8601 date): Start date
- `end` (ISO 8601 date): End date

**Example**:
```bash
curl "http://localhost:8080/api/v1/analytics/videos/video-123/qoe?period=daily&start=2025-01-01T00:00:00Z&end=2025-01-17T00:00:00Z"
```

**Response** (200 OK):
```json
[
  {
    "video_id": "video-123",
    "period": "daily",
    "timestamp": "2025-01-17T00:00:00Z",
    "view_count": 100,
    "average_qoe": 85.5,
    "rebuffer_ratio": 0.05,
    "startup_time": 2.1,
    "bitrate_utilization": 0.8,
    "error_rate": 0.02,
    "completion_rate": 75.0
  }
]
```

---

#### Get Trending Videos

Get a list of trending videos.

**Endpoint**: `GET /api/v1/analytics/trending?limit=10`

**Query Parameters**:
- `limit` (integer): Number of trending videos to return (default: 10, max: 100)

**Response** (200 OK):
```json
[
  {
    "video_id": "video-123",
    "title": "Sample Video.mp4",
    "views": 500,
    "view_growth": 150.5,
    "trending_score": 750.0,
    "last_updated": "2025-01-17T12:00:00Z"
  }
]
```

---

**Version**: 7.0.0
**Last Updated**: 2025-01-17
