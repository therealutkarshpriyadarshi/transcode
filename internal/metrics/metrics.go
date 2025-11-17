package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// API Metrics
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "transcode_http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "transcode_http_request_duration_seconds",
			Help:    "HTTP request latency in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)

	// Upload Metrics
	VideoUploadsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "transcode_video_uploads_total",
			Help: "Total number of video uploads",
		},
	)

	VideoUploadSizeBytes = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "transcode_video_upload_size_bytes",
			Help:    "Size of uploaded videos in bytes",
			Buckets: prometheus.ExponentialBuckets(1024*1024, 2, 15), // 1MB to 16GB
		},
	)

	// Job Metrics
	JobsCreatedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "transcode_jobs_created_total",
			Help: "Total number of transcoding jobs created",
		},
		[]string{"priority"},
	)

	JobsCompletedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "transcode_jobs_completed_total",
			Help: "Total number of completed transcoding jobs",
		},
		[]string{"status"},
	)

	JobsInProgress = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "transcode_jobs_in_progress",
			Help: "Number of jobs currently being processed",
		},
	)

	JobsQueueDepth = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "transcode_jobs_queue_depth",
			Help: "Number of jobs waiting in queue",
		},
	)

	JobDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "transcode_job_duration_seconds",
			Help:    "Job processing duration in seconds",
			Buckets: prometheus.ExponentialBuckets(1, 2, 12), // 1s to ~1 hour
		},
		[]string{"resolution", "codec"},
	)

	JobQueueTime = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "transcode_job_queue_time_seconds",
			Help:    "Time jobs spend waiting in queue",
			Buckets: prometheus.ExponentialBuckets(1, 2, 10),
		},
	)

	// Worker Metrics
	WorkerActive = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "transcode_worker_active",
			Help: "Number of active workers",
		},
		[]string{"worker_type"},
	)

	WorkerJobsProcessed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "transcode_worker_jobs_processed_total",
			Help: "Total number of jobs processed by workers",
		},
		[]string{"worker_id", "worker_type"},
	)

	// Transcoding Metrics
	TranscodingSpeed = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "transcode_speed_ratio",
			Help:    "Transcoding speed ratio (output duration / processing time)",
			Buckets: []float64{0.1, 0.25, 0.5, 1.0, 2.0, 4.0, 8.0, 16.0},
		},
		[]string{"codec", "resolution", "worker_type"},
	)

	TranscodingBitrate = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "transcode_output_bitrate_bps",
			Help:    "Output bitrate in bits per second",
			Buckets: prometheus.ExponentialBuckets(100000, 2, 15), // 100kbps to 3.2Gbps
		},
		[]string{"resolution", "codec"},
	)

	// GPU Metrics
	GPUUtilization = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "transcode_gpu_utilization_percent",
			Help: "GPU utilization percentage",
		},
		[]string{"gpu_id", "worker_id"},
	)

	GPUMemoryUsed = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "transcode_gpu_memory_used_bytes",
			Help: "GPU memory used in bytes",
		},
		[]string{"gpu_id", "worker_id"},
	)

	GPUTemperature = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "transcode_gpu_temperature_celsius",
			Help: "GPU temperature in Celsius",
		},
		[]string{"gpu_id", "worker_id"},
	)

	// Storage Metrics
	StorageOperationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "transcode_storage_operations_total",
			Help: "Total number of storage operations",
		},
		[]string{"operation", "status"},
	)

	StorageOperationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "transcode_storage_operation_duration_seconds",
			Help:    "Storage operation duration in seconds",
			Buckets: prometheus.ExponentialBuckets(0.01, 2, 12),
		},
		[]string{"operation"},
	)

	StorageBytesTransferred = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "transcode_storage_bytes_transferred_total",
			Help: "Total bytes transferred to/from storage",
		},
		[]string{"operation"},
	)

	// Database Metrics
	DatabaseOperationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "transcode_database_operations_total",
			Help: "Total number of database operations",
		},
		[]string{"operation", "status"},
	)

	DatabaseOperationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "transcode_database_operation_duration_seconds",
			Help:    "Database operation duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation"},
	)

	DatabaseConnectionsActive = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "transcode_database_connections_active",
			Help: "Number of active database connections",
		},
	)

	// Quality Metrics
	VMAFScore = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "transcode_vmaf_score",
			Help:    "VMAF quality score",
			Buckets: []float64{50, 60, 70, 75, 80, 85, 90, 92, 94, 96, 98, 100},
		},
		[]string{"resolution", "codec"},
	)

	BitrateOptimizationSavings = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "transcode_bitrate_optimization_savings_percent",
			Help:    "Percentage of bitrate savings from optimization",
			Buckets: []float64{0, 5, 10, 15, 20, 25, 30, 35, 40, 50},
		},
		[]string{"content_type"},
	)

	// Cache Metrics
	CacheHitsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "transcode_cache_hits_total",
			Help: "Total number of cache hits",
		},
		[]string{"cache_type"},
	)

	CacheMissesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "transcode_cache_misses_total",
			Help: "Total number of cache misses",
		},
		[]string{"cache_type"},
	)

	// Error Metrics
	ErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "transcode_errors_total",
			Help: "Total number of errors",
		},
		[]string{"component", "error_type"},
	)

	// Business Metrics
	VideoDurationProcessed = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "transcode_video_duration_processed_seconds_total",
			Help: "Total duration of video processed in seconds",
		},
	)

	EstimatedCostPerJob = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "transcode_estimated_cost_per_job_dollars",
			Help:    "Estimated cost per transcoding job in dollars",
			Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0, 5.0},
		},
		[]string{"worker_type"},
	)
)

// RecordHTTPRequest records an HTTP request
func RecordHTTPRequest(method, endpoint, status string, duration float64) {
	HTTPRequestsTotal.WithLabelValues(method, endpoint, status).Inc()
	HTTPRequestDuration.WithLabelValues(method, endpoint).Observe(duration)
}

// RecordJobCreated records a job creation
func RecordJobCreated(priority string) {
	JobsCreatedTotal.WithLabelValues(priority).Inc()
}

// RecordJobCompleted records a job completion
func RecordJobCompleted(status string, duration float64, resolution, codec string) {
	JobsCompletedTotal.WithLabelValues(status).Inc()
	JobDuration.WithLabelValues(resolution, codec).Observe(duration)
}

// UpdateJobMetrics updates current job metrics
func UpdateJobMetrics(inProgress, queueDepth int) {
	JobsInProgress.Set(float64(inProgress))
	JobsQueueDepth.Set(float64(queueDepth))
}

// RecordTranscodingSpeed records transcoding speed ratio
func RecordTranscodingSpeed(codec, resolution, workerType string, speed float64) {
	TranscodingSpeed.WithLabelValues(codec, resolution, workerType).Observe(speed)
}

// UpdateGPUMetrics updates GPU metrics
func UpdateGPUMetrics(gpuID, workerID string, utilization float64, memoryUsed int64, temperature float64) {
	GPUUtilization.WithLabelValues(gpuID, workerID).Set(utilization)
	GPUMemoryUsed.WithLabelValues(gpuID, workerID).Set(float64(memoryUsed))
	GPUTemperature.WithLabelValues(gpuID, workerID).Set(temperature)
}

// RecordStorageOperation records a storage operation
func RecordStorageOperation(operation, status string, duration float64, bytesTransferred int64) {
	StorageOperationsTotal.WithLabelValues(operation, status).Inc()
	StorageOperationDuration.WithLabelValues(operation).Observe(duration)
	StorageBytesTransferred.WithLabelValues(operation).Add(float64(bytesTransferred))
}

// RecordDatabaseOperation records a database operation
func RecordDatabaseOperation(operation, status string, duration float64) {
	DatabaseOperationsTotal.WithLabelValues(operation, status).Inc()
	DatabaseOperationDuration.WithLabelValues(operation).Observe(duration)
}

// RecordVMAFScore records a VMAF quality score
func RecordVMAFScore(resolution, codec string, score float64) {
	VMAFScore.WithLabelValues(resolution, codec).Observe(score)
}

// RecordCacheAccess records cache hit or miss
func RecordCacheAccess(cacheType string, hit bool) {
	if hit {
		CacheHitsTotal.WithLabelValues(cacheType).Inc()
	} else {
		CacheMissesTotal.WithLabelValues(cacheType).Inc()
	}
}

// RecordError records an error
func RecordError(component, errorType string) {
	ErrorsTotal.WithLabelValues(component, errorType).Inc()
}
