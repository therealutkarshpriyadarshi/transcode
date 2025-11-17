package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestRecordHTTPRequest(t *testing.T) {
	// Reset metrics
	HTTPRequestsTotal.Reset()
	HTTPRequestDuration.Reset()

	RecordHTTPRequest("GET", "/api/v1/videos", "200", 0.123)

	// Verify counter incremented
	counter := testutil.ToFloat64(HTTPRequestsTotal.WithLabelValues("GET", "/api/v1/videos", "200"))
	if counter != 1.0 {
		t.Errorf("Expected counter to be 1.0, got %f", counter)
	}
}

func TestRecordJobCreated(t *testing.T) {
	JobsCreatedTotal.Reset()

	RecordJobCreated("high")
	RecordJobCreated("medium")
	RecordJobCreated("high")

	highPriority := testutil.ToFloat64(JobsCreatedTotal.WithLabelValues("high"))
	if highPriority != 2.0 {
		t.Errorf("Expected high priority counter to be 2.0, got %f", highPriority)
	}

	mediumPriority := testutil.ToFloat64(JobsCreatedTotal.WithLabelValues("medium"))
	if mediumPriority != 1.0 {
		t.Errorf("Expected medium priority counter to be 1.0, got %f", mediumPriority)
	}
}

func TestRecordJobCompleted(t *testing.T) {
	JobsCompletedTotal.Reset()

	RecordJobCompleted("completed", 120.5, "1080p", "h264")
	RecordJobCompleted("failed", 30.2, "720p", "h264")

	completed := testutil.ToFloat64(JobsCompletedTotal.WithLabelValues("completed"))
	if completed != 1.0 {
		t.Errorf("Expected completed counter to be 1.0, got %f", completed)
	}

	failed := testutil.ToFloat64(JobsCompletedTotal.WithLabelValues("failed"))
	if failed != 1.0 {
		t.Errorf("Expected failed counter to be 1.0, got %f", failed)
	}
}

func TestUpdateJobMetrics(t *testing.T) {
	UpdateJobMetrics(5, 10)

	inProgress := testutil.ToFloat64(JobsInProgress)
	if inProgress != 5.0 {
		t.Errorf("Expected jobs in progress to be 5.0, got %f", inProgress)
	}

	queueDepth := testutil.ToFloat64(JobsQueueDepth)
	if queueDepth != 10.0 {
		t.Errorf("Expected queue depth to be 10.0, got %f", queueDepth)
	}
}

func TestRecordTranscodingSpeed(t *testing.T) {
	TranscodingSpeed.Reset()

	RecordTranscodingSpeed("h264", "1080p", "gpu", 4.5)
	RecordTranscodingSpeed("h264", "1080p", "cpu", 0.8)

	// Just verify no errors
	// Actual histogram values require more complex verification
}

func TestUpdateGPUMetrics(t *testing.T) {
	UpdateGPUMetrics("0", "worker-1", 85.5, 4096, 72.0)

	utilization := testutil.ToFloat64(GPUUtilization.WithLabelValues("0", "worker-1"))
	if utilization != 85.5 {
		t.Errorf("Expected GPU utilization to be 85.5, got %f", utilization)
	}

	memory := testutil.ToFloat64(GPUMemoryUsed.WithLabelValues("0", "worker-1"))
	if memory != 4096.0 {
		t.Errorf("Expected GPU memory to be 4096.0, got %f", memory)
	}

	temperature := testutil.ToFloat64(GPUTemperature.WithLabelValues("0", "worker-1"))
	if temperature != 72.0 {
		t.Errorf("Expected GPU temperature to be 72.0, got %f", temperature)
	}
}

func TestRecordStorageOperation(t *testing.T) {
	StorageOperationsTotal.Reset()
	StorageBytesTransferred.Reset()

	RecordStorageOperation("upload", "success", 1.234, 1048576)

	counter := testutil.ToFloat64(StorageOperationsTotal.WithLabelValues("upload", "success"))
	if counter != 1.0 {
		t.Errorf("Expected storage operation counter to be 1.0, got %f", counter)
	}

	bytes := testutil.ToFloat64(StorageBytesTransferred.WithLabelValues("upload"))
	if bytes != 1048576.0 {
		t.Errorf("Expected bytes transferred to be 1048576.0, got %f", bytes)
	}
}

func TestRecordDatabaseOperation(t *testing.T) {
	DatabaseOperationsTotal.Reset()

	RecordDatabaseOperation("select", "success", 0.05)
	RecordDatabaseOperation("insert", "error", 0.02)

	success := testutil.ToFloat64(DatabaseOperationsTotal.WithLabelValues("select", "success"))
	if success != 1.0 {
		t.Errorf("Expected select success counter to be 1.0, got %f", success)
	}

	error := testutil.ToFloat64(DatabaseOperationsTotal.WithLabelValues("insert", "error"))
	if error != 1.0 {
		t.Errorf("Expected insert error counter to be 1.0, got %f", error)
	}
}

func TestRecordVMAFScore(t *testing.T) {
	VMAFScore.Reset()

	RecordVMAFScore("1080p", "h264", 95.5)
	RecordVMAFScore("720p", "h265", 92.0)

	// Just verify no errors
	// Histogram values require more complex verification
}

func TestRecordCacheAccess(t *testing.T) {
	CacheHitsTotal.Reset()
	CacheMissesTotal.Reset()

	RecordCacheAccess("metadata", true)
	RecordCacheAccess("metadata", true)
	RecordCacheAccess("metadata", false)

	hits := testutil.ToFloat64(CacheHitsTotal.WithLabelValues("metadata"))
	if hits != 2.0 {
		t.Errorf("Expected cache hits to be 2.0, got %f", hits)
	}

	misses := testutil.ToFloat64(CacheMissesTotal.WithLabelValues("metadata"))
	if misses != 1.0 {
		t.Errorf("Expected cache misses to be 1.0, got %f", misses)
	}
}

func TestRecordError(t *testing.T) {
	ErrorsTotal.Reset()

	RecordError("api", "validation")
	RecordError("worker", "ffmpeg")
	RecordError("api", "validation")

	apiErrors := testutil.ToFloat64(ErrorsTotal.WithLabelValues("api", "validation"))
	if apiErrors != 2.0 {
		t.Errorf("Expected API validation errors to be 2.0, got %f", apiErrors)
	}

	workerErrors := testutil.ToFloat64(ErrorsTotal.WithLabelValues("worker", "ffmpeg"))
	if workerErrors != 1.0 {
		t.Errorf("Expected worker FFmpeg errors to be 1.0, got %f", workerErrors)
	}
}

func BenchmarkRecordHTTPRequest(b *testing.B) {
	for i := 0; i < b.N; i++ {
		RecordHTTPRequest("GET", "/api/v1/videos", "200", 0.123)
	}
}

func BenchmarkUpdateGPUMetrics(b *testing.B) {
	for i := 0; i < b.N; i++ {
		UpdateGPUMetrics("0", "worker-1", 85.5, 4096, 72.0)
	}
}
