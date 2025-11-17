# Deployment Guide

This guide covers deploying the Video Transcoding Pipeline in various environments.

## Table of Contents

1. [Local Development](#local-development)
2. [Docker Compose](#docker-compose)
3. [Production Considerations](#production-considerations)
4. [Monitoring](#monitoring)
5. [Troubleshooting](#troubleshooting)

## Local Development

### Prerequisites

- Go 1.21+
- Docker & Docker Compose
- PostgreSQL 15+
- Redis 7+
- RabbitMQ
- MinIO or AWS S3
- FFmpeg 7.0+

### Setup Steps

1. **Install Dependencies**
   ```bash
   # Install Go dependencies
   go mod download

   # Install FFmpeg (macOS)
   brew install ffmpeg

   # Install FFmpeg (Ubuntu/Debian)
   sudo apt-get install ffmpeg
   ```

2. **Start Infrastructure Services**
   ```bash
   # Start only the infrastructure services
   docker-compose up -d postgres redis minio rabbitmq
   ```

3. **Run Database Migrations**
   ```bash
   # Apply migrations
   docker cp migrations/001_init_schema.up.sql transcode-postgres:/001_init_schema.up.sql
   docker exec transcode-postgres psql -U postgres -d transcode -f /001_init_schema.up.sql
   ```

4. **Configure Environment**
   ```bash
   # Copy example config
   cp .env.example .env

   # Update config for local development
   # Edit config.yaml to use localhost instead of service names
   ```

5. **Run Services Locally**
   ```bash
   # Terminal 1: Run API
   make run-api

   # Terminal 2: Run Worker
   make run-worker
   ```

## Docker Compose

### Quick Start

```bash
# Start all services
docker-compose up -d

# Run migrations
docker cp migrations/001_init_schema.up.sql transcode-postgres:/001_init_schema.up.sql
docker exec transcode-postgres psql -U postgres -d transcode -f /001_init_schema.up.sql

# Check service status
docker-compose ps

# View logs
docker-compose logs -f
```

### Scaling Workers

```bash
# Scale workers to 4 replicas
docker-compose up -d --scale worker=4

# Check worker status
docker-compose ps worker
```

### Production-Like Setup

For a more production-like setup with Docker Compose:

1. **Use External Services**
   - Managed PostgreSQL (AWS RDS, GCP Cloud SQL)
   - Managed Redis (AWS ElastiCache, Redis Cloud)
   - AWS S3 instead of MinIO
   - Managed RabbitMQ (CloudAMQP)

2. **Update docker-compose.override.yml**
   ```yaml
   version: '3.8'

   services:
     api:
       environment:
         - DB_HOST=your-rds-endpoint
         - REDIS_HOST=your-redis-endpoint
         - STORAGE_ENDPOINT=s3.amazonaws.com
         - STORAGE_BUCKET=your-bucket-name
         - QUEUE_HOST=your-rabbitmq-endpoint
   ```

3. **Enable HTTPS**
   - Add nginx reverse proxy
   - Configure SSL certificates
   - Update API to use HTTPS

## Production Considerations

### Infrastructure

1. **Database**
   - Use managed PostgreSQL (RDS, Cloud SQL)
   - Enable automated backups
   - Set up read replicas for scaling
   - Configure connection pooling

2. **Message Queue**
   - Use managed RabbitMQ or migrate to Kafka
   - Configure clustering for high availability
   - Set up dead letter queues
   - Monitor queue depth

3. **Storage**
   - Use S3 or compatible object storage
   - Enable versioning
   - Configure lifecycle policies
   - Set up CloudFront CDN

4. **Caching**
   - Use managed Redis (ElastiCache)
   - Configure persistence
   - Set up clustering
   - Monitor memory usage

### Security

1. **API Security**
   ```go
   // Implement authentication middleware
   // Add rate limiting
   // Enable CORS properly
   // Validate all inputs
   // Use HTTPS only
   ```

2. **Database Security**
   - Use strong passwords
   - Enable SSL connections
   - Restrict network access
   - Regular security updates

3. **Storage Security**
   - Use IAM roles instead of access keys
   - Enable bucket encryption
   - Set up bucket policies
   - Enable access logging

4. **Secrets Management**
   - Use AWS Secrets Manager or HashiCorp Vault
   - Never commit secrets to git
   - Rotate credentials regularly
   - Use environment variables

### Performance

1. **API Optimization**
   - Enable response compression
   - Implement caching
   - Use connection pooling
   - Add request timeouts

2. **Worker Optimization**
   - Adjust worker count based on CPU cores
   - Configure FFmpeg presets
   - Use GPU acceleration (Phase 4)
   - Implement batch processing

3. **Database Optimization**
   - Add appropriate indexes
   - Optimize queries
   - Use prepared statements
   - Regular VACUUM operations

### Scaling

1. **Horizontal Scaling**
   - Deploy multiple API instances behind load balancer
   - Scale workers independently based on queue depth
   - Use auto-scaling groups

2. **Load Balancing**
   ```nginx
   upstream api {
       server api1:8080;
       server api2:8080;
       server api3:8080;
   }

   server {
       listen 80;
       location / {
           proxy_pass http://api;
       }
   }
   ```

3. **Auto-Scaling**
   - Scale based on CPU/memory usage
   - Scale based on queue depth
   - Set minimum and maximum replicas

## Monitoring

### Metrics to Track

1. **System Metrics**
   - CPU usage
   - Memory usage
   - Disk usage
   - Network I/O

2. **Application Metrics**
   - Request rate
   - Response time
   - Error rate
   - Queue depth

3. **Business Metrics**
   - Videos processed
   - Processing time
   - Success rate
   - Cost per video

### Logging

1. **Structured Logging**
   ```go
   log.WithFields(log.Fields{
       "job_id": job.ID,
       "video_id": job.VideoID,
       "status": job.Status,
   }).Info("Processing job")
   ```

2. **Log Aggregation**
   - Use ELK stack or Loki
   - Set up log retention policies
   - Create dashboards
   - Set up alerts

### Health Checks

1. **API Health Check**
   ```bash
   curl http://localhost:8080/health
   ```

2. **Worker Health Check**
   - Monitor job processing
   - Check FFmpeg availability
   - Verify queue connectivity

## Troubleshooting

### Common Issues

1. **API Won't Start**
   ```bash
   # Check logs
   docker logs transcode-api

   # Verify database connectivity
   docker exec transcode-postgres pg_isready

   # Check configuration
   cat config.yaml
   ```

2. **Worker Not Processing Jobs**
   ```bash
   # Check worker logs
   docker logs transcode-worker

   # Check queue
   curl -u guest:guest http://localhost:15672/api/queues

   # Check RabbitMQ connectivity
   docker logs transcode-rabbitmq
   ```

3. **FFmpeg Errors**
   ```bash
   # Test FFmpeg directly
   docker exec transcode-worker ffmpeg -version

   # Check temp directory permissions
   docker exec transcode-worker ls -la /tmp/transcode

   # View worker logs for FFmpeg errors
   docker logs transcode-worker | grep ffmpeg
   ```

4. **Storage Issues**
   ```bash
   # Check MinIO
   curl http://localhost:9000/minio/health/live

   # List buckets
   docker exec transcode-minio mc ls local

   # Check bucket exists
   docker exec transcode-minio mc ls local/videos
   ```

### Performance Issues

1. **Slow Transcoding**
   - Check CPU usage
   - Verify FFmpeg preset settings
   - Consider GPU acceleration
   - Optimize video parameters

2. **High Memory Usage**
   - Adjust worker count
   - Check for memory leaks
   - Optimize chunk size
   - Monitor temp directory cleanup

3. **Database Slow Queries**
   - Check query execution plans
   - Add missing indexes
   - Optimize queries
   - Increase connection pool size

### Debugging

1. **Enable Debug Logging**
   ```yaml
   # In config.yaml
   logging:
     level: debug
   ```

2. **Trace Requests**
   - Add request IDs
   - Log at key points
   - Use distributed tracing

3. **Profile Application**
   ```bash
   # CPU profiling
   go test -cpuprofile cpu.prof

   # Memory profiling
   go test -memprofile mem.prof

   # Analyze profiles
   go tool pprof cpu.prof
   ```

## Backup and Recovery

### Database Backup

```bash
# Backup database
docker exec transcode-postgres pg_dump -U postgres transcode > backup.sql

# Restore database
docker exec -i transcode-postgres psql -U postgres transcode < backup.sql
```

### Storage Backup

```bash
# Sync to backup location
aws s3 sync s3://videos s3://videos-backup

# Enable versioning
aws s3api put-bucket-versioning \
  --bucket videos \
  --versioning-configuration Status=Enabled
```

## Disaster Recovery

1. **Regular Backups**
   - Automated daily database backups
   - Replicate storage to multiple regions
   - Test restore procedures

2. **High Availability**
   - Multi-AZ deployment
   - Load balancer with health checks
   - Auto-healing with health monitoring

3. **Recovery Procedures**
   - Document recovery steps
   - Test recovery process regularly
   - Maintain runbooks

## Cost Optimization

1. **Compute**
   - Use spot instances for workers
   - Scale down during off-peak hours
   - Right-size instances

2. **Storage**
   - Use S3 Intelligent Tiering
   - Implement lifecycle policies
   - Delete temporary files
   - Compress outputs

3. **Network**
   - Use CloudFront for delivery
   - Minimize cross-region transfers
   - Optimize file sizes

---

**Last Updated**: 2025-01-17
