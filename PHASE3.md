# Phase 3: API & Job Management - Complete Documentation

## Overview

Phase 3 enhances the video transcoding service with production-ready API features, advanced job management, and comprehensive monitoring capabilities.

## Table of Contents

- [Features](#features)
- [Architecture](#architecture)
- [API Reference](#api-reference)
- [Authentication](#authentication)
- [Rate Limiting & Quotas](#rate-limiting--quotas)
- [Multipart Uploads](#multipart-uploads)
- [Webhooks](#webhooks)
- [Job Management](#job-management)
- [Monitoring](#monitoring)
- [Configuration](#configuration)
- [Testing](#testing)

## Features

### Authentication & Authorization
- **JWT-based authentication** for stateless API access
- **API key authentication** for server-to-server integrations
- **User management** with quota tracking
- **Role-based access control** (foundation for future expansion)

### Multipart Upload System
- **Resumable uploads** for large video files
- **Chunked transfer** with configurable part sizes (default 5MB)
- **Upload session management** with automatic cleanup
- **MD5 verification** for data integrity
- **24-hour upload expiration** for stale sessions

### Webhook Notifications
- **Event-driven notifications** for job lifecycle events
- **HMAC-SHA256 signatures** for webhook verification
- **Automatic retry with exponential backoff** (1min, 5min, 15min, 1hr, 4hr, 12hr)
- **Delivery tracking** and status monitoring
- **Configurable event subscriptions**

### Advanced Job Scheduling
- **Priority-based queue** with FIFO ordering within priority levels
- **Resource-aware scheduling** based on worker capacity
- **Job pause/resume** functionality
- **Job cancellation** with proper cleanup
- **Dependency management** (foundation for multi-stage pipelines)

### Dead Letter Queue (DLQ)
- **Failed job isolation** for analysis and recovery
- **Configurable retry attempts** (max 5 retries)
- **Exponential backoff retry strategy**
- **Manual retry capability** from DLQ
- **Failure reason tracking** and categorization

### Monitoring & Observability
- **Real-time metrics** collection
- **Worker health monitoring** with heartbeat tracking
- **Queue depth tracking** for both main and DLQ
- **Performance metrics** (average wait time, process time)
- **System health status** with alert generation
- **Job statistics** by status

### Rate Limiting & Quotas
- **Per-user and per-IP rate limiting**
- **Token bucket algorithm** implementation
- **Daily quota management** with automatic reset
- **Configurable limits** per user tier
- **Quota overage alerts**

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                      Client Applications                        │
└────────────────────────────┬────────────────────────────────────┘
                             │
                    ┌────────▼────────┐
                    │   API Gateway   │
                    │  (Gin Router)   │
                    │   + Middleware  │
                    └────────┬────────┘
                             │
         ┌───────────────────┼───────────────────┐
         │                   │                   │
    ┌────▼─────┐      ┌─────▼──────┐     ┌─────▼──────┐
    │PostgreSQL│      │   Redis    │     │  MinIO/S3  │
    │(Metadata)│      │  (Cache)   │     │  (Videos)  │
    └──────────┘      └────────────┘     └────────────┘
                             │
              ┌──────────────┼──────────────┐
              │              │              │
      ┌───────▼──────┐ ┌─────▼──────┐ ┌────▼──────┐
      │ Job Scheduler│ │  Webhooks  │ │ Monitoring│
      │  (Priority)  │ │  (Retry)   │ │ (Metrics) │
      └───────┬──────┘ └────────────┘ └───────────┘
              │
    ┌─────────▼─────────┐
    │     RabbitMQ      │
    │  ┌─────────────┐  │
    │  │ Main Queue  │  │
    │  └─────┬───────┘  │
    │        │          │
    │  ┌─────▼───────┐  │
    │  │ Retry Queue │  │
    │  └─────┬───────┘  │
    │        │          │
    │  ┌─────▼───────┐  │
    │  │     DLQ     │  │
    │  └─────────────┘  │
    └───────────────────┘
              │
      ┌───────┴───────┐
      │    Workers    │
      │   (FFmpeg)    │
      └───────────────┘
```

## API Reference

### Base URL
```
http://localhost:8080/api/v1
```

### Authentication

#### Register User
```http
POST /auth/register
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "secure_password_123"
}

Response:
{
  "id": "user-uuid",
  "email": "user@example.com",
  "api_key": "generated-api-key"
}
```

#### Login
```http
POST /auth/login
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "secure_password_123"
}

Response:
{
  "token": "jwt-token",
  "user_id": "user-uuid",
  "email": "user@example.com",
  "api_key": "your-api-key",
  "quota": 100,
  "used_quota": 15
}
```

### Video Upload

#### Standard Upload
```http
POST /videos/upload
Authorization: Bearer <jwt-token>
# OR
X-API-Key: <api-key>
Content-Type: multipart/form-data

video: <file>

Response:
{
  "id": "video-uuid",
  "filename": "video.mp4",
  "size": 1234567,
  "duration": 120.5,
  "width": 1920,
  "height": 1080,
  "codec": "h264",
  "bitrate": 5000000,
  "status": "pending"
}
```

#### Multipart Upload (for large files)

**Step 1: Initiate Upload**
```http
POST /uploads/initiate
Authorization: Bearer <jwt-token>
Content-Type: application/json

{
  "filename": "large_video.mp4",
  "total_size": 5368709120
}

Response:
{
  "id": "upload-uuid",
  "filename": "large_video.mp4",
  "total_size": 5368709120,
  "part_size": 5242880,
  "total_parts": 1024,
  "status": "active",
  "expires_at": "2025-01-18T10:00:00Z"
}
```

**Step 2: Upload Parts**
```http
PUT /uploads/:upload_id/parts/:part_number
Authorization: Bearer <jwt-token>
Content-Type: application/octet-stream

<binary data>

Response:
{
  "part_number": 1,
  "size": 5242880,
  "etag": "md5-hash",
  "uploaded": true,
  "uploaded_at": "2025-01-17T10:05:00Z"
}
```

**Step 3: Complete Upload**
```http
POST /uploads/:upload_id/complete
Authorization: Bearer <jwt-token>

Response:
{
  "id": "video-uuid",
  "filename": "large_video.mp4",
  ...
}
```

**Abort Upload**
```http
DELETE /uploads/:upload_id
Authorization: Bearer <jwt-token>
```

### Job Management

#### Create Transcode Job
```http
POST /videos/:id/transcode
Authorization: Bearer <jwt-token>
Content-Type: application/json

{
  "resolution": "1080p",
  "output_format": "mp4",
  "codec": "libx264",
  "bitrate": 5000000,
  "preset": "medium",
  "priority": 10
}

Response:
{
  "id": "job-uuid",
  "video_id": "video-uuid",
  "status": "queued",
  "priority": 10,
  "progress": 0,
  "config": { ... }
}
```

#### Get Job Status
```http
GET /jobs/:id
Authorization: Bearer <jwt-token>

Response:
{
  "id": "job-uuid",
  "video_id": "video-uuid",
  "status": "processing",
  "priority": 10,
  "progress": 45.5,
  "worker_id": "worker-1",
  "started_at": "2025-01-17T10:00:00Z",
  "created_at": "2025-01-17T09:55:00Z"
}
```

#### Cancel Job
```http
POST /jobs/:id/cancel
Authorization: Bearer <jwt-token>

Response:
{
  "message": "Job cancelled"
}
```

#### Pause Job
```http
POST /jobs/:id/pause
Authorization: Bearer <jwt-token>

Response:
{
  "message": "Job paused"
}
```

#### Resume Job
```http
POST /jobs/:id/resume
Authorization: Bearer <jwt-token>

Response:
{
  "message": "Job resumed"
}
```

### Webhooks

#### Create Webhook
```http
POST /webhooks
Authorization: Bearer <jwt-token>
Content-Type: application/json

{
  "url": "https://your-app.com/webhooks",
  "events": {
    "job_started": true,
    "job_completed": true,
    "job_failed": true,
    "job_progress": false,
    "video_uploaded": true
  },
  "secret": "your-webhook-secret"
}

Response:
{
  "id": "webhook-uuid",
  "url": "https://your-app.com/webhooks",
  "events": { ... },
  "is_active": true
}
```

#### List Webhooks
```http
GET /webhooks
Authorization: Bearer <jwt-token>

Response:
{
  "webhooks": [
    {
      "id": "webhook-uuid",
      "url": "https://your-app.com/webhooks",
      "events": { ... },
      "is_active": true,
      "created_at": "2025-01-17T10:00:00Z"
    }
  ]
}
```

### Webhook Payload Format

```json
{
  "event": "job.completed",
  "timestamp": "2025-01-17T10:30:00Z",
  "data": {
    "id": "job-uuid",
    "video_id": "video-uuid",
    "status": "completed",
    "progress": 100,
    ...
  }
}
```

### Webhook Signature Verification

Webhooks include an `X-Webhook-Signature` header:
```
X-Webhook-Signature: sha256=<hmac-sha256-hex>
```

Verification example (Python):
```python
import hmac
import hashlib

def verify_webhook(payload, signature, secret):
    expected = 'sha256=' + hmac.new(
        secret.encode(),
        payload.encode(),
        hashlib.sha256
    ).hexdigest()
    return hmac.compare_digest(expected, signature)
```

### Monitoring

#### Get System Metrics
```http
GET /metrics
Authorization: Bearer <jwt-token>

Response:
{
  "queue_depth": 45,
  "dlq_depth": 3,
  "active_jobs": 10,
  "total_jobs": 1250,
  "completed_jobs": 1180,
  "failed_jobs": 25,
  "cancelled_jobs": 0,
  "average_wait_time_seconds": 125.5,
  "average_process_time_seconds": 450.2,
  "worker_count": 4,
  "healthy_workers": 4,
  "last_updated": "2025-01-17T10:45:00Z"
}
```

#### Get Worker Health
```http
GET /workers/health
Authorization: Bearer <jwt-token>

Response:
{
  "workers": [
    {
      "worker_id": "worker-1",
      "status": "healthy",
      "last_heartbeat": "2025-01-17T10:44:50Z",
      "current_job": "job-uuid",
      "processed_jobs": 125
    }
  ]
}
```

#### Get System Health
```http
GET /system/health
Authorization: Bearer <jwt-token>

Response:
{
  "status": "healthy",
  "alerts": []
}
```

Possible statuses:
- `healthy`: All systems operating normally
- `warning`: Some issues detected (high queue depth, some unhealthy workers)
- `critical`: Severe issues (high DLQ depth, majority of workers unhealthy)

#### Get Queue Statistics
```http
GET /queue/stats
Authorization: Bearer <jwt-token>

Response:
{
  "queue_depth": 45,
  "dlq_depth": 3
}
```

## Configuration

Update `config.yaml` with Phase 3 settings:

```yaml
server:
  port: 8080
  host: "0.0.0.0"
  readTimeout: "5m"      # Increased for large uploads
  writeTimeout: "5m"
  shutdownTimeout: "10s"

database:
  host: "postgres"
  port: 5432
  user: "postgres"
  password: "postgres"
  dbname: "transcode"
  sslmode: "disable"
  maxConns: 50          # Increased for higher concurrency
  minConns: 10

storage:
  endpoint: "minio:9000"
  accessKeyID: "minioadmin"
  secretAccessKey: "minioadmin"
  bucketName: "videos"
  region: "us-east-1"
  useSSL: false

queue:
  host: "rabbitmq"
  port: 5672
  user: "guest"
  password: "guest"
  vhost: "/"

transcoder:
  workerCount: 4
  tempDir: "/tmp/transcode"
  ffmpegPath: "ffmpeg"
  ffprobePath: "ffprobe"
  maxConcurrent: 10     # Max concurrent jobs across all workers
  chunkSize: 5242880    # 5MB for multipart uploads

redis:
  host: "redis"
  port: 6379
  password: ""
  db: 0

# Phase 3 specific settings
ratelimit:
  requestsPerSecond: 10
  burst: 20

quota:
  defaultDailyLimit: 100
  resetHour: 0          # Reset quotas at midnight UTC

monitoring:
  metricsInterval: 10   # Collect metrics every 10 seconds
  healthCheckInterval: 30

webhooks:
  maxRetries: 6
  timeout: 30           # Webhook request timeout in seconds
```

## Testing

### Run All Tests
```bash
go test ./...
```

### Run Specific Package Tests
```bash
# Authentication tests
go test ./internal/middleware -v

# Webhook tests
go test ./internal/webhook -v

# Scheduler tests
go test ./internal/scheduler -v

# Upload tests
go test ./internal/upload -v
```

### Test Coverage
```bash
go test -cover ./...
```

## Database Migrations

Run Phase 3 migration:

```bash
docker cp migrations/003_phase3_schema.up.sql transcode-postgres:/003_phase3_schema.up.sql
docker exec transcode-postgres psql -U postgres -d transcode -f /003_phase3_schema.up.sql
```

## Usage Examples

### Complete Workflow Example (cURL)

**1. Register and Login**
```bash
# Register
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"password123"}'

# Login
TOKEN=$(curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"password123"}' \
  | jq -r '.token')
```

**2. Upload Video**
```bash
# Small file - standard upload
curl -X POST http://localhost:8080/api/v1/videos/upload \
  -H "Authorization: Bearer $TOKEN" \
  -F "video=@video.mp4"

# Large file - multipart upload
UPLOAD_ID=$(curl -X POST http://localhost:8080/api/v1/uploads/initiate \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"filename":"large.mp4","total_size":1073741824}' \
  | jq -r '.id')

# Upload parts (example for part 1)
curl -X PUT "http://localhost:8080/api/v1/uploads/$UPLOAD_ID/parts/1" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/octet-stream" \
  --data-binary @part1.bin

# Complete upload
VIDEO_ID=$(curl -X POST "http://localhost:8080/api/v1/uploads/$UPLOAD_ID/complete" \
  -H "Authorization: Bearer $TOKEN" \
  | jq -r '.id')
```

**3. Create Transcode Job**
```bash
JOB_ID=$(curl -X POST "http://localhost:8080/api/v1/videos/$VIDEO_ID/transcode" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "resolution": "1080p",
    "output_format": "mp4",
    "codec": "libx264",
    "bitrate": 5000000,
    "priority": 10
  }' \
  | jq -r '.id')
```

**4. Setup Webhook**
```bash
curl -X POST http://localhost:8080/api/v1/webhooks \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://your-app.com/webhooks",
    "events": {
      "job_started": true,
      "job_completed": true,
      "job_failed": true
    },
    "secret": "your-secret"
  }'
```

**5. Monitor Progress**
```bash
# Check job status
curl "http://localhost:8080/api/v1/jobs/$JOB_ID" \
  -H "Authorization: Bearer $TOKEN"

# Check system metrics
curl http://localhost:8080/api/v1/metrics \
  -H "Authorization: Bearer $TOKEN"
```

## Best Practices

### Authentication
- Store JWT tokens securely (httpOnly cookies for web apps)
- Use API keys for server-to-server communication
- Rotate API keys periodically
- Never commit secrets to version control

### Rate Limiting
- Implement exponential backoff in clients
- Cache responses when possible
- Use webhooks instead of polling for status updates

### Multipart Uploads
- Use multipart for files > 100MB
- Implement retry logic for failed parts
- Clean up incomplete uploads
- Verify ETags for data integrity

### Webhooks
- Always verify webhook signatures
- Return 2xx status quickly (process async if needed)
- Implement idempotency (handle duplicate deliveries)
- Log all webhook attempts for debugging

### Job Management
- Use appropriate priority levels (0-10)
- Don't cancel jobs that are actively processing
- Monitor DLQ for recurring failures
- Implement proper error handling

## Troubleshooting

### Common Issues

**High Queue Depth**
- Scale up workers
- Increase `maxConcurrent` setting
- Check for slow jobs blocking the queue

**Webhook Delivery Failures**
- Verify webhook URL is accessible
- Check firewall/security group settings
- Ensure webhook endpoint returns 2xx status
- Review webhook delivery logs in database

**Rate Limiting**
- Reduce request frequency
- Implement request batching
- Use webhooks instead of polling
- Contact admin to increase quota

**Upload Failures**
- Check network connectivity
- Verify file size limits
- Ensure sufficient disk space
- Review nginx/proxy timeout settings

## Security Considerations

1. **Authentication**: All endpoints except `/auth/register` and `/auth/login` require authentication
2. **HTTPS**: Use HTTPS in production (configure reverse proxy)
3. **Secrets**: Store JWT secret and API keys in environment variables
4. **Input Validation**: All inputs are validated and sanitized
5. **Rate Limiting**: Prevents abuse and DoS attacks
6. **Webhook Signatures**: Verify all webhook payloads
7. **SQL Injection**: Using parameterized queries throughout
8. **File Upload**: Validate file types and sizes

## Performance Optimization

1. **Database Indexing**: All foreign keys and frequently queried columns are indexed
2. **Connection Pooling**: Configured for database, Redis, and RabbitMQ
3. **Caching**: Implement Redis caching for frequently accessed data
4. **CDN**: Use CDN for serving transcoded videos
5. **Monitoring**: Continuously monitor metrics and optimize bottlenecks

## Phase 3 Checklist

- [x] JWT authentication
- [x] API key authentication
- [x] User management with quotas
- [x] Rate limiting (per-user and per-IP)
- [x] Multipart upload support
- [x] Webhook notifications
- [x] Webhook retry with exponential backoff
- [x] Priority-based job scheduler
- [x] Job pause/resume/cancel
- [x] Dead letter queue
- [x] System monitoring
- [x] Worker health tracking
- [x] Comprehensive tests
- [x] API documentation

## Next Steps (Phase 4)

- GPU-accelerated transcoding
- Hardware encoder support (NVENC, QuickSync, VideoToolbox)
- Auto-scaling based on queue depth
- Advanced caching strategies
- CDN integration
- Performance profiling and optimization

---

**Version**: 3.0.0
**Last Updated**: 2025-01-17
**Status**: Complete ✅
