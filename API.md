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

**Version**: 1.0.0
**Last Updated**: 2025-01-17
