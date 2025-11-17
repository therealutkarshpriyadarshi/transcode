package cache

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/therealutkarshpriyadarshi/transcode/pkg/models"
)

func setupTestCache(t *testing.T) (*Cache, *miniredis.Miniredis) {
	// Create a mini Redis server for testing
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}

	// Parse host and port
	cache, err := NewCache(mr.Host(), mr.Server().Addr().Port, "", 0)
	if err != nil {
		mr.Close()
		t.Fatalf("Failed to create cache: %v", err)
	}

	return cache, mr
}

func TestNewCache(t *testing.T) {
	cache, mr := setupTestCache(t)
	defer mr.Close()
	defer cache.Close()

	if cache == nil {
		t.Fatal("Cache should not be nil")
	}

	// Test ping
	ctx := context.Background()
	if err := cache.Ping(ctx); err != nil {
		t.Errorf("Ping failed: %v", err)
	}
}

func TestCache_VideoOperations(t *testing.T) {
	cache, mr := setupTestCache(t)
	defer mr.Close()
	defer cache.Close()

	ctx := context.Background()

	// Create test video
	video := &models.Video{
		ID:       "test-video-1",
		Filename: "test.mp4",
		Size:     1024,
		Duration: 60.0,
		Width:    1920,
		Height:   1080,
		Status:   models.VideoStatusPending,
	}

	// Test SetVideo
	err := cache.SetVideo(ctx, video, 5*time.Minute)
	if err != nil {
		t.Fatalf("SetVideo failed: %v", err)
	}

	// Test GetVideo
	retrieved, err := cache.GetVideo(ctx, video.ID)
	if err != nil {
		t.Fatalf("GetVideo failed: %v", err)
	}

	if retrieved == nil {
		t.Fatal("Retrieved video should not be nil")
	}

	if retrieved.ID != video.ID {
		t.Errorf("Expected ID %s, got %s", video.ID, retrieved.ID)
	}

	if retrieved.Filename != video.Filename {
		t.Errorf("Expected filename %s, got %s", video.Filename, retrieved.Filename)
	}

	// Test GetVideo for non-existent video
	nonExistent, err := cache.GetVideo(ctx, "non-existent")
	if err != nil {
		t.Fatalf("GetVideo for non-existent should not error: %v", err)
	}

	if nonExistent != nil {
		t.Error("Non-existent video should return nil")
	}

	// Test DeleteVideo
	err = cache.DeleteVideo(ctx, video.ID)
	if err != nil {
		t.Fatalf("DeleteVideo failed: %v", err)
	}

	// Verify deletion
	deleted, err := cache.GetVideo(ctx, video.ID)
	if err != nil {
		t.Fatalf("GetVideo after delete failed: %v", err)
	}

	if deleted != nil {
		t.Error("Deleted video should return nil")
	}
}

func TestCache_JobOperations(t *testing.T) {
	cache, mr := setupTestCache(t)
	defer mr.Close()
	defer cache.Close()

	ctx := context.Background()

	// Create test job
	job := &models.Job{
		ID:      "test-job-1",
		VideoID: "test-video-1",
		Status:  models.JobStatusPending,
		Config: models.JobConfig{
			Resolution: "1080p",
			Codec:      "h264",
		},
	}

	// Test SetJob
	err := cache.SetJob(ctx, job, 5*time.Minute)
	if err != nil {
		t.Fatalf("SetJob failed: %v", err)
	}

	// Test GetJob
	retrieved, err := cache.GetJob(ctx, job.ID)
	if err != nil {
		t.Fatalf("GetJob failed: %v", err)
	}

	if retrieved == nil {
		t.Fatal("Retrieved job should not be nil")
	}

	if retrieved.ID != job.ID {
		t.Errorf("Expected ID %s, got %s", job.ID, retrieved.ID)
	}

	// Test DeleteJob
	err = cache.DeleteJob(ctx, job.ID)
	if err != nil {
		t.Fatalf("DeleteJob failed: %v", err)
	}
}

func TestCache_JobProgress(t *testing.T) {
	cache, mr := setupTestCache(t)
	defer mr.Close()
	defer cache.Close()

	ctx := context.Background()
	jobID := "test-job-1"

	// Test SetJobProgress
	err := cache.SetJobProgress(ctx, jobID, 50.5, 5*time.Minute)
	if err != nil {
		t.Fatalf("SetJobProgress failed: %v", err)
	}

	// Test GetJobProgress
	progress, err := cache.GetJobProgress(ctx, jobID)
	if err != nil {
		t.Fatalf("GetJobProgress failed: %v", err)
	}

	if progress != 50.5 {
		t.Errorf("Expected progress 50.5, got %f", progress)
	}
}

func TestCache_ThumbnailOperations(t *testing.T) {
	cache, mr := setupTestCache(t)
	defer mr.Close()
	defer cache.Close()

	ctx := context.Background()
	videoID := "test-video-1"
	thumbnailType := "poster"
	url := "https://example.com/thumbnail.jpg"

	// Test SetThumbnail
	err := cache.SetThumbnail(ctx, videoID, thumbnailType, url, 10*time.Minute)
	if err != nil {
		t.Fatalf("SetThumbnail failed: %v", err)
	}

	// Test GetThumbnail
	retrieved, err := cache.GetThumbnail(ctx, videoID, thumbnailType)
	if err != nil {
		t.Fatalf("GetThumbnail failed: %v", err)
	}

	if retrieved != url {
		t.Errorf("Expected URL %s, got %s", url, retrieved)
	}

	// Test non-existent thumbnail
	nonExistent, err := cache.GetThumbnail(ctx, videoID, "non-existent")
	if err != nil {
		t.Fatalf("GetThumbnail for non-existent should not error: %v", err)
	}

	if nonExistent != "" {
		t.Error("Non-existent thumbnail should return empty string")
	}
}

func TestCache_OutputOperations(t *testing.T) {
	cache, mr := setupTestCache(t)
	defer mr.Close()
	defer cache.Close()

	ctx := context.Background()
	videoID := "test-video-1"

	outputs := []*models.Output{
		{
			ID:         "output-1",
			VideoID:    videoID,
			Resolution: "1080p",
			Format:     "mp4",
		},
		{
			ID:         "output-2",
			VideoID:    videoID,
			Resolution: "720p",
			Format:     "mp4",
		},
	}

	// Test SetOutputs
	err := cache.SetOutputs(ctx, videoID, outputs, 10*time.Minute)
	if err != nil {
		t.Fatalf("SetOutputs failed: %v", err)
	}

	// Test GetOutputs
	retrieved, err := cache.GetOutputs(ctx, videoID)
	if err != nil {
		t.Fatalf("GetOutputs failed: %v", err)
	}

	if len(retrieved) != len(outputs) {
		t.Errorf("Expected %d outputs, got %d", len(outputs), len(retrieved))
	}

	// Test DeleteOutputs
	err = cache.DeleteOutputs(ctx, videoID)
	if err != nil {
		t.Fatalf("DeleteOutputs failed: %v", err)
	}

	// Verify deletion
	deleted, err := cache.GetOutputs(ctx, videoID)
	if err != nil {
		t.Fatalf("GetOutputs after delete failed: %v", err)
	}

	if deleted != nil {
		t.Error("Deleted outputs should return nil")
	}
}

func TestCache_StatOperations(t *testing.T) {
	cache, mr := setupTestCache(t)
	defer mr.Close()
	defer cache.Close()

	ctx := context.Background()
	stat := "videos_processed"

	// Test IncrementStat
	err := cache.IncrementStat(ctx, stat)
	if err != nil {
		t.Fatalf("IncrementStat failed: %v", err)
	}

	// Increment again
	err = cache.IncrementStat(ctx, stat)
	if err != nil {
		t.Fatalf("IncrementStat failed: %v", err)
	}

	// Test GetStat
	value, err := cache.GetStat(ctx, stat)
	if err != nil {
		t.Fatalf("GetStat failed: %v", err)
	}

	if value != 2 {
		t.Errorf("Expected stat value 2, got %d", value)
	}

	// Test SetStat
	err = cache.SetStat(ctx, stat, 100, 5*time.Minute)
	if err != nil {
		t.Fatalf("SetStat failed: %v", err)
	}

	value, err = cache.GetStat(ctx, stat)
	if err != nil {
		t.Fatalf("GetStat failed: %v", err)
	}

	if value != 100 {
		t.Errorf("Expected stat value 100, got %d", value)
	}
}

func TestCache_RateLimit(t *testing.T) {
	cache, mr := setupTestCache(t)
	defer mr.Close()
	defer cache.Close()

	ctx := context.Background()
	key := "user:123"
	limit := int64(5)
	window := 1 * time.Minute

	// Should allow first 5 requests
	for i := 0; i < 5; i++ {
		allowed, err := cache.CheckRateLimit(ctx, key, limit, window)
		if err != nil {
			t.Fatalf("CheckRateLimit failed: %v", err)
		}

		if !allowed {
			t.Errorf("Request %d should be allowed", i+1)
		}
	}

	// Should deny 6th request
	allowed, err := cache.CheckRateLimit(ctx, key, limit, window)
	if err != nil {
		t.Fatalf("CheckRateLimit failed: %v", err)
	}

	if allowed {
		t.Error("Request beyond limit should be denied")
	}
}

func TestCache_Locking(t *testing.T) {
	cache, mr := setupTestCache(t)
	defer mr.Close()
	defer cache.Close()

	ctx := context.Background()
	resource := "video:test-123"

	// Test AcquireLock
	acquired, err := cache.AcquireLock(ctx, resource, 1*time.Minute)
	if err != nil {
		t.Fatalf("AcquireLock failed: %v", err)
	}

	if !acquired {
		t.Error("First lock acquisition should succeed")
	}

	// Test acquiring same lock again (should fail)
	acquired, err = cache.AcquireLock(ctx, resource, 1*time.Minute)
	if err != nil {
		t.Fatalf("Second AcquireLock failed: %v", err)
	}

	if acquired {
		t.Error("Second lock acquisition should fail")
	}

	// Test ReleaseLock
	err = cache.ReleaseLock(ctx, resource)
	if err != nil {
		t.Fatalf("ReleaseLock failed: %v", err)
	}

	// Should be able to acquire again
	acquired, err = cache.AcquireLock(ctx, resource, 1*time.Minute)
	if err != nil {
		t.Fatalf("AcquireLock after release failed: %v", err)
	}

	if !acquired {
		t.Error("Lock acquisition after release should succeed")
	}
}

func TestCache_Exists(t *testing.T) {
	cache, mr := setupTestCache(t)
	defer mr.Close()
	defer cache.Close()

	ctx := context.Background()
	key := "test:key"

	// Key should not exist initially
	exists, err := cache.Exists(ctx, key)
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}

	if exists {
		t.Error("Key should not exist initially")
	}

	// Set a value
	err = cache.SetWithJSON(ctx, key, map[string]string{"test": "value"}, 5*time.Minute)
	if err != nil {
		t.Fatalf("SetWithJSON failed: %v", err)
	}

	// Key should exist now
	exists, err = cache.Exists(ctx, key)
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}

	if !exists {
		t.Error("Key should exist after setting")
	}
}

func TestCache_SetGetWithJSON(t *testing.T) {
	cache, mr := setupTestCache(t)
	defer mr.Close()
	defer cache.Close()

	ctx := context.Background()
	key := "test:json"

	type TestData struct {
		Name  string
		Count int
	}

	original := TestData{
		Name:  "test",
		Count: 42,
	}

	// Test SetWithJSON
	err := cache.SetWithJSON(ctx, key, original, 5*time.Minute)
	if err != nil {
		t.Fatalf("SetWithJSON failed: %v", err)
	}

	// Test GetWithJSON
	var retrieved TestData
	err = cache.GetWithJSON(ctx, key, &retrieved)
	if err != nil {
		t.Fatalf("GetWithJSON failed: %v", err)
	}

	if retrieved.Name != original.Name {
		t.Errorf("Expected Name %s, got %s", original.Name, retrieved.Name)
	}

	if retrieved.Count != original.Count {
		t.Errorf("Expected Count %d, got %d", original.Count, retrieved.Count)
	}
}

// Benchmark tests
func BenchmarkCache_SetVideo(b *testing.B) {
	mr, _ := miniredis.Run()
	defer mr.Close()

	cache, _ := NewCache(mr.Host(), mr.Server().Addr().Port, "", 0)
	defer cache.Close()

	ctx := context.Background()
	video := &models.Video{
		ID:       "benchmark-video",
		Filename: "test.mp4",
		Size:     1024,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.SetVideo(ctx, video, 5*time.Minute)
	}
}

func BenchmarkCache_GetVideo(b *testing.B) {
	mr, _ := miniredis.Run()
	defer mr.Close()

	cache, _ := NewCache(mr.Host(), mr.Server().Addr().Port, "", 0)
	defer cache.Close()

	ctx := context.Background()
	video := &models.Video{
		ID:       "benchmark-video",
		Filename: "test.mp4",
		Size:     1024,
	}

	cache.SetVideo(ctx, video, 5*time.Minute)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.GetVideo(ctx, video.ID)
	}
}
