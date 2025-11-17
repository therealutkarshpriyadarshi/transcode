package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/therealutkarshpriyadarshi/transcode/pkg/models"
)

// Cache provides caching functionality using Redis
type Cache struct {
	client *redis.Client
}

// NewCache creates a new cache instance
func NewCache(host string, port int, password string, db int) (*Cache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", host, port),
		Password: password,
		DB:       db,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &Cache{client: client}, nil
}

// Close closes the Redis connection
func (c *Cache) Close() error {
	return c.client.Close()
}

// Video Cache Operations

// SetVideo caches video metadata
func (c *Cache) SetVideo(ctx context.Context, video *models.Video, ttl time.Duration) error {
	data, err := json.Marshal(video)
	if err != nil {
		return fmt.Errorf("failed to marshal video: %w", err)
	}

	key := fmt.Sprintf("video:%s", video.ID)
	return c.client.Set(ctx, key, data, ttl).Err()
}

// GetVideo retrieves video metadata from cache
func (c *Cache) GetVideo(ctx context.Context, videoID string) (*models.Video, error) {
	key := fmt.Sprintf("video:%s", videoID)
	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Cache miss
		}
		return nil, fmt.Errorf("failed to get video from cache: %w", err)
	}

	var video models.Video
	if err := json.Unmarshal(data, &video); err != nil {
		return nil, fmt.Errorf("failed to unmarshal video: %w", err)
	}

	return &video, nil
}

// DeleteVideo removes video from cache
func (c *Cache) DeleteVideo(ctx context.Context, videoID string) error {
	key := fmt.Sprintf("video:%s", videoID)
	return c.client.Del(ctx, key).Err()
}

// Job Cache Operations

// SetJob caches job metadata
func (c *Cache) SetJob(ctx context.Context, job *models.Job, ttl time.Duration) error {
	data, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job: %w", err)
	}

	key := fmt.Sprintf("job:%s", job.ID)
	return c.client.Set(ctx, key, data, ttl).Err()
}

// GetJob retrieves job metadata from cache
func (c *Cache) GetJob(ctx context.Context, jobID string) (*models.Job, error) {
	key := fmt.Sprintf("job:%s", jobID)
	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Cache miss
		}
		return nil, fmt.Errorf("failed to get job from cache: %w", err)
	}

	var job models.Job
	if err := json.Unmarshal(data, &job); err != nil {
		return nil, fmt.Errorf("failed to unmarshal job: %w", err)
	}

	return &job, nil
}

// DeleteJob removes job from cache
func (c *Cache) DeleteJob(ctx context.Context, jobID string) error {
	key := fmt.Sprintf("job:%s", jobID)
	return c.client.Del(ctx, key).Err()
}

// SetJobProgress caches job progress for quick retrieval
func (c *Cache) SetJobProgress(ctx context.Context, jobID string, progress float64, ttl time.Duration) error {
	key := fmt.Sprintf("job:progress:%s", jobID)
	return c.client.Set(ctx, key, progress, ttl).Err()
}

// GetJobProgress retrieves job progress from cache
func (c *Cache) GetJobProgress(ctx context.Context, jobID string) (float64, error) {
	key := fmt.Sprintf("job:progress:%s", jobID)
	return c.client.Get(ctx, key).Float64()
}

// Thumbnail Cache Operations

// SetThumbnail caches thumbnail URL
func (c *Cache) SetThumbnail(ctx context.Context, videoID string, thumbnailType string, url string, ttl time.Duration) error {
	key := fmt.Sprintf("thumbnail:%s:%s", videoID, thumbnailType)
	return c.client.Set(ctx, key, url, ttl).Err()
}

// GetThumbnail retrieves thumbnail URL from cache
func (c *Cache) GetThumbnail(ctx context.Context, videoID string, thumbnailType string) (string, error) {
	key := fmt.Sprintf("thumbnail:%s:%s", videoID, thumbnailType)
	url, err := c.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return "", nil // Cache miss
		}
		return "", fmt.Errorf("failed to get thumbnail from cache: %w", err)
	}
	return url, nil
}

// Output Cache Operations

// SetOutputs caches video outputs
func (c *Cache) SetOutputs(ctx context.Context, videoID string, outputs []*models.Output, ttl time.Duration) error {
	data, err := json.Marshal(outputs)
	if err != nil {
		return fmt.Errorf("failed to marshal outputs: %w", err)
	}

	key := fmt.Sprintf("outputs:%s", videoID)
	return c.client.Set(ctx, key, data, ttl).Err()
}

// GetOutputs retrieves video outputs from cache
func (c *Cache) GetOutputs(ctx context.Context, videoID string) ([]*models.Output, error) {
	key := fmt.Sprintf("outputs:%s", videoID)
	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Cache miss
		}
		return nil, fmt.Errorf("failed to get outputs from cache: %w", err)
	}

	var outputs []*models.Output
	if err := json.Unmarshal(data, &outputs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal outputs: %w", err)
	}

	return outputs, nil
}

// DeleteOutputs removes outputs from cache
func (c *Cache) DeleteOutputs(ctx context.Context, videoID string) error {
	key := fmt.Sprintf("outputs:%s", videoID)
	return c.client.Del(ctx, key).Err()
}

// Stats Cache Operations

// IncrementStat increments a statistic counter
func (c *Cache) IncrementStat(ctx context.Context, stat string) error {
	key := fmt.Sprintf("stats:%s", stat)
	return c.client.Incr(ctx, key).Err()
}

// GetStat retrieves a statistic value
func (c *Cache) GetStat(ctx context.Context, stat string) (int64, error) {
	key := fmt.Sprintf("stats:%s", stat)
	return c.client.Get(ctx, key).Int64()
}

// SetStat sets a statistic value
func (c *Cache) SetStat(ctx context.Context, stat string, value int64, ttl time.Duration) error {
	key := fmt.Sprintf("stats:%s", stat)
	return c.client.Set(ctx, key, value, ttl).Err()
}

// GPU Status Cache

// SetGPUStatus caches GPU status
func (c *Cache) SetGPUStatus(ctx context.Context, gpuIndex int, status interface{}, ttl time.Duration) error {
	data, err := json.Marshal(status)
	if err != nil {
		return fmt.Errorf("failed to marshal GPU status: %w", err)
	}

	key := fmt.Sprintf("gpu:status:%d", gpuIndex)
	return c.client.Set(ctx, key, data, ttl).Err()
}

// GetGPUStatus retrieves GPU status from cache
func (c *Cache) GetGPUStatus(ctx context.Context, gpuIndex int) ([]byte, error) {
	key := fmt.Sprintf("gpu:status:%d", gpuIndex)
	return c.client.Get(ctx, key).Bytes()
}

// Rate Limiting Operations

// CheckRateLimit checks if a rate limit has been exceeded
func (c *Cache) CheckRateLimit(ctx context.Context, key string, limit int64, window time.Duration) (bool, error) {
	rateLimitKey := fmt.Sprintf("ratelimit:%s", key)

	// Increment counter
	count, err := c.client.Incr(ctx, rateLimitKey).Result()
	if err != nil {
		return false, fmt.Errorf("failed to increment rate limit: %w", err)
	}

	// Set expiry on first request
	if count == 1 {
		if err := c.client.Expire(ctx, rateLimitKey, window).Err(); err != nil {
			return false, fmt.Errorf("failed to set expiry: %w", err)
		}
	}

	// Check if limit exceeded
	return count <= limit, nil
}

// Locking Operations for Distributed Systems

// AcquireLock attempts to acquire a distributed lock
func (c *Cache) AcquireLock(ctx context.Context, resource string, ttl time.Duration) (bool, error) {
	key := fmt.Sprintf("lock:%s", resource)
	return c.client.SetNX(ctx, key, "locked", ttl).Result()
}

// ReleaseLock releases a distributed lock
func (c *Cache) ReleaseLock(ctx context.Context, resource string) error {
	key := fmt.Sprintf("lock:%s", resource)
	return c.client.Del(ctx, key).Err()
}

// Batch Operations

// DeletePattern deletes all keys matching a pattern
func (c *Cache) DeletePattern(ctx context.Context, pattern string) error {
	iter := c.client.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		if err := c.client.Del(ctx, iter.Val()).Err(); err != nil {
			return fmt.Errorf("failed to delete key %s: %w", iter.Val(), err)
		}
	}
	return iter.Err()
}

// Exists checks if a key exists
func (c *Cache) Exists(ctx context.Context, key string) (bool, error) {
	result, err := c.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return result > 0, nil
}

// SetWithJSON sets a value with JSON marshaling
func (c *Cache) SetWithJSON(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}
	return c.client.Set(ctx, key, data, ttl).Err()
}

// GetWithJSON gets a value with JSON unmarshaling
func (c *Cache) GetWithJSON(ctx context.Context, key string, dest interface{}) error {
	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil // Cache miss
		}
		return fmt.Errorf("failed to get value from cache: %w", err)
	}

	if err := json.Unmarshal(data, dest); err != nil {
		return fmt.Errorf("failed to unmarshal value: %w", err)
	}

	return nil
}

// Health check
func (c *Cache) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}
