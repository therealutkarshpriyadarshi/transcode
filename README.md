# Video Transcoding Pipeline

A production-grade distributed video transcoding service that converts uploaded videos into multiple resolutions with adaptive bitrate streaming support (HLS/DASH). Built with Go, FFmpeg, and a microservices architecture.

## Features

### Phase 1 (Foundation) ✅

- **Video Upload & Storage**: Upload videos with automatic metadata extraction
- **FFmpeg Integration**: Full FFmpeg wrapper with progress tracking
- **Basic Transcoding**: Single-resolution transcoding with configurable codecs and bitrates
- **Distributed Architecture**:
  - RESTful API server
  - Worker service for job processing
  - Message queue (RabbitMQ) for job distribution
  - Object storage (MinIO/S3) for video files
  - PostgreSQL database for metadata
  - Redis for caching
- **Job Management**: Create, monitor, and track transcoding jobs
- **Progress Tracking**: Real-time transcoding progress updates
- **Error Handling**: Comprehensive error handling and retry logic

### Phase 2 (Multi-Resolution & Adaptive Streaming) ✅

- **Multi-Resolution Transcoding**: Parallel transcoding to multiple resolutions (144p to 4K)
- **Intelligent Resolution Selection**: Automatic resolution ladder based on source video
- **HLS Streaming**: HTTP Live Streaming with master playlists and variant streams
- **DASH Streaming**: Dynamic Adaptive Streaming with MPD manifests
- **Thumbnail Generation**: Single thumbnails, sprite sheets, and animated previews
- **Audio Processing**: Normalization using loudnorm filter, multi-track support
- **Subtitle Support**: Extract, convert, and burn-in subtitles (VTT, SRT, ASS)
- **Multiple Codec Support**: H.264, H.265/HEVC, VP9, with optimized encoding profiles
- **Advanced Database Schema**: Support for thumbnails, subtitles, streaming profiles, audio tracks

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         Client Applications                      │
└────────────────────────────┬────────────────────────────────────┘
                             │
                    ┌────────▼────────┐
                    │   API Server    │
                    │  (Port 8080)    │
                    └────────┬────────┘
                             │
         ┌───────────────────┼───────────────────┐
         │                   │                   │
    ┌────▼─────┐      ┌─────▼──────┐     ┌─────▼──────┐
    │PostgreSQL│      │   Redis    │     │  MinIO/S3  │
    │(Metadata)│      │  (Cache)   │     │  (Videos)  │
    └──────────┘      └────────────┘     └────────────┘
                             │
                    ┌────────▼────────┐
                    │    RabbitMQ     │
                    │   (Job Queue)   │
                    └────────┬────────┘
                             │
              ┌──────────────┼──────────────┐
              │              │              │
        ┌─────▼─────┐  ┌─────▼─────┐  ┌────▼──────┐
        │  Worker 1 │  │  Worker 2 │  │  Worker N │
        │ (FFmpeg)  │  │ (FFmpeg)  │  │ (FFmpeg)  │
        └───────────┘  └───────────┘  └───────────┘
```

## Tech Stack

- **Language**: Go 1.21+
- **Transcoding**: FFmpeg 7.0+
- **Message Queue**: RabbitMQ
- **Storage**: MinIO (local) / S3 (production)
- **Database**: PostgreSQL 15+
- **Cache**: Redis 7+
- **API Framework**: Gin
- **Containerization**: Docker & Docker Compose

## Project Structure

```
transcode/
├── cmd/
│   ├── api/          # API server entry point
│   └── worker/       # Worker service entry point
├── internal/
│   ├── config/       # Configuration management
│   ├── database/     # Database connection and repository
│   ├── queue/        # RabbitMQ queue operations
│   ├── storage/      # MinIO/S3 storage operations
│   └── transcoder/   # FFmpeg wrapper and transcoding logic
├── pkg/
│   └── models/       # Data models
├── migrations/       # Database migrations
├── test/            # Integration tests
├── docker-compose.yml
├── config.yaml
├── Makefile
└── README.md
```

## Prerequisites

- Docker and Docker Compose
- Go 1.21+ (for local development)
- FFmpeg 7.0+ (installed automatically in Docker)
- Make (optional, for convenience commands)

## Quick Start

### 1. Clone the Repository

```bash
git clone https://github.com/therealutkarshpriyadarshi/transcode.git
cd transcode
```

### 2. Start All Services

```bash
# Using Docker Compose
docker-compose up -d

# Or using Make
make docker-up
```

This will start:
- PostgreSQL (port 5432)
- Redis (port 6379)
- MinIO (port 9000, console 9001)
- RabbitMQ (port 5672, management 15672)
- API Server (port 8080)
- Worker (2 replicas)

### 3. Run Database Migrations

```bash
# Copy migrations to PostgreSQL container
docker cp migrations/001_init_schema.up.sql transcode-postgres:/001_init_schema.up.sql

# Run migration
docker exec transcode-postgres psql -U postgres -d transcode -f /001_init_schema.up.sql
```

### 4. Verify Services

```bash
# Check API health
curl http://localhost:8080/health

# Check MinIO console
open http://localhost:9001
# Login: minioadmin / minioadmin

# Check RabbitMQ management
open http://localhost:15672
# Login: guest / guest
```

## API Documentation

### Base URL
```
http://localhost:8080/api/v1
```

### Endpoints

#### Upload Video
```bash
POST /videos/upload

# Example
curl -X POST http://localhost:8080/api/v1/videos/upload \
  -F "video=@/path/to/video.mp4"

# Response
{
  "id": "uuid",
  "filename": "video.mp4",
  "size": 1234567,
  "duration": 120.5,
  "width": 1920,
  "height": 1080,
  "codec": "h264",
  "bitrate": 5000000,
  "status": "pending",
  "created_at": "2025-01-17T10:00:00Z"
}
```

#### Get Video Details
```bash
GET /videos/:id

# Example
curl http://localhost:8080/api/v1/videos/{video-id}
```

#### List Videos
```bash
GET /videos

# Example
curl http://localhost:8080/api/v1/videos
```

#### Create Transcode Job
```bash
POST /videos/:id/transcode

# Example
curl -X POST http://localhost:8080/api/v1/videos/{video-id}/transcode \
  -H "Content-Type: application/json" \
  -d '{
    "resolution": "720p",
    "output_format": "mp4",
    "codec": "libx264",
    "bitrate": 2500000,
    "preset": "medium",
    "priority": 5
  }'

# Response
{
  "id": "job-uuid",
  "video_id": "video-uuid",
  "status": "queued",
  "priority": 5,
  "progress": 0,
  "config": {
    "output_format": "mp4",
    "resolution": "720p",
    "codec": "libx264",
    "bitrate": 2500000
  }
}
```

#### Get Job Status
```bash
GET /jobs/:id

# Example
curl http://localhost:8080/api/v1/jobs/{job-id}
```

#### Get Video Jobs
```bash
GET /videos/:id/jobs

# Example
curl http://localhost:8080/api/v1/videos/{video-id}/jobs
```

#### Get Video Outputs
```bash
GET /videos/:id/outputs

# Example
curl http://localhost:8080/api/v1/videos/{video-id}/outputs
```

## Configuration

Configuration is managed via `config.yaml`:

```yaml
server:
  port: 8080
  host: "0.0.0.0"

database:
  host: "postgres"
  port: 5432
  user: "postgres"
  password: "postgres"
  dbname: "transcode"

storage:
  endpoint: "minio:9000"
  accessKeyID: "minioadmin"
  secretAccessKey: "minioadmin"
  bucketName: "videos"

queue:
  host: "rabbitmq"
  port: 5672
  user: "guest"
  password: "guest"

transcoder:
  workerCount: 2
  tempDir: "/tmp/transcode"
  ffmpegPath: "ffmpeg"
  ffprobePath: "ffprobe"
```

## Development

### Local Development Setup

```bash
# Install dependencies
go mod download

# Run tests
make test

# Run API locally (requires services running)
make run-api

# Run worker locally
make run-worker

# Build binaries
make build
```

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
make test

# Run specific package tests
go test ./internal/config -v
```

### Code Formatting

```bash
# Format code
go fmt ./...

# Run linters
make lint
```

## Deployment

### Docker Compose (Development/Testing)

```bash
# Start all services
docker-compose up -d

# View logs
docker-compose logs -f

# Stop services
docker-compose down
```

### Production Deployment

For production deployment:

1. Use environment variables for sensitive configuration
2. Set up proper S3 bucket with appropriate IAM roles
3. Use managed PostgreSQL (RDS) and Redis (ElastiCache)
4. Deploy to Kubernetes (Phase 6 of roadmap)
5. Set up monitoring and alerting

## Troubleshooting

### FFmpeg Not Found
```bash
# Check FFmpeg installation in container
docker exec transcode-worker ffmpeg -version
```

### Database Connection Issues
```bash
# Check PostgreSQL is running
docker exec transcode-postgres pg_isready

# View PostgreSQL logs
docker logs transcode-postgres
```

### Worker Not Processing Jobs
```bash
# Check RabbitMQ queue depth
curl -u guest:guest http://localhost:15672/api/queues/%2F/transcode_jobs

# View worker logs
docker logs transcode-worker
```

## Performance Benchmarks

Based on 1080p video transcoding to 720p:

- **Processing Speed**: 0.5-1x real-time on CPU
- **Memory Usage**: ~200-500MB per worker
- **Concurrent Jobs**: Up to 4 jobs per worker (configurable)

## Roadmap

See [roadmap.md](roadmap.md) for the complete project roadmap.

### Completed ✅
- **Phase 1**: Foundation (Weeks 1-2)
  - Project setup and infrastructure
  - FFmpeg integration and basic transcoding
  - Worker and API services

- **Phase 2**: Multi-Resolution & Adaptive Streaming (Weeks 3-5)
  - Multi-resolution transcoding with resolution ladder
  - HLS/DASH manifest generation and segmented streaming
  - Thumbnail and sprite sheet generation
  - Audio normalization and multi-track support
  - Subtitle extraction and processing

- **Phase 3**: API & Job Management (Weeks 6-7)
  - JWT and API key authentication
  - Rate limiting and user quotas
  - Multipart upload for large files
  - Webhook notifications with retry logic
  - Priority-based job scheduler
  - Dead letter queue for failed jobs
  - Comprehensive monitoring and metrics
  - Job pause/resume/cancel functionality

- **Phase 4**: GPU Acceleration & Performance Optimization (Weeks 8-9) ✅
  - NVIDIA NVENC hardware encoding (H.264, H.265)
  - Automatic GPU detection and multi-GPU support
  - Intelligent GPU selection based on memory and utilization
  - Automatic CPU fallback on GPU failure
  - Two-pass encoding for better quality (CPU)
  - Redis caching for metadata and thumbnails
  - Optimized parallel multipart uploads/downloads
  - GPU resource monitoring and management

### Coming Next 🚀
- Phase 5: AI-Powered Per-Title Encoding
- Phase 6: Kubernetes & Production Readiness

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Make your changes with tests
4. Submit a pull request

## License

MIT License - see LICENSE file for details

## Contact

For questions or issues, please open an issue on GitHub.

---

**Status**: Phase 4 Complete ✅
**Version**: 4.0.0
**Last Updated**: 2025-01-17

## Phase Documentation

- **Phase 1**: [Foundation](roadmap.md#phase-1-foundation-weeks-1-2) - Basic transcoding pipeline ✅
- **Phase 2**: [Multi-Resolution & Adaptive Streaming](PHASE2.md) - HLS/DASH, thumbnails, audio/subtitle processing ✅
- **Phase 3**: [API & Job Management](PHASE3.md) - Authentication, webhooks, monitoring, advanced scheduling ✅
- **Phase 4**: [GPU Acceleration & Performance Optimization](PHASE4.md) - NVIDIA NVENC, caching, optimized storage ✅

For comprehensive Phase 4 documentation including GPU acceleration, automatic fallback, caching strategies, and performance optimizations, see [PHASE4.md](PHASE4.md).
