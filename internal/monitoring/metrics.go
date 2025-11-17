package monitoring

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// Metrics holds system metrics
type Metrics struct {
	QueueDepth        int       `json:"queue_depth"`
	DLQDepth          int       `json:"dlq_depth"`
	ActiveJobs        int       `json:"active_jobs"`
	TotalJobs         int64     `json:"total_jobs"`
	CompletedJobs     int64     `json:"completed_jobs"`
	FailedJobs        int64     `json:"failed_jobs"`
	CancelledJobs     int64     `json:"cancelled_jobs"`
	AverageWaitTime   float64   `json:"average_wait_time_seconds"`
	AverageProcessTime float64  `json:"average_process_time_seconds"`
	WorkerCount       int       `json:"worker_count"`
	HealthyWorkers    int       `json:"healthy_workers"`
	LastUpdated       time.Time `json:"last_updated"`
}

// WorkerHealth holds worker health information
type WorkerHealth struct {
	WorkerID      string    `json:"worker_id"`
	Status        string    `json:"status"`
	LastHeartbeat time.Time `json:"last_heartbeat"`
	CurrentJob    string    `json:"current_job,omitempty"`
	ProcessedJobs int64     `json:"processed_jobs"`
}

// Monitor provides system monitoring and health checks
type Monitor struct {
	metrics       *Metrics
	workers       map[string]*WorkerHealth
	mu            sync.RWMutex
	repo          MetricsRepository
	queueProvider QueueProvider
}

// MetricsRepository defines the interface for metrics data
type MetricsRepository interface {
	GetJobStats(ctx context.Context) (total, completed, failed, cancelled int64, err error)
	GetAverageWaitTime(ctx context.Context) (float64, error)
	GetAverageProcessTime(ctx context.Context) (float64, error)
	GetActiveWorkers(ctx context.Context) (int, error)
}

// QueueProvider defines the interface for queue metrics
type QueueProvider interface {
	GetQueueDepth() (int, error)
	GetDLQDepth() (int, error)
}

// NewMonitor creates a new monitoring service
func NewMonitor(repo MetricsRepository, queueProvider QueueProvider) *Monitor {
	return &Monitor{
		metrics: &Metrics{
			LastUpdated: time.Now(),
		},
		workers:       make(map[string]*WorkerHealth),
		repo:          repo,
		queueProvider: queueProvider,
	}
}

// Start begins the monitoring service
func (m *Monitor) Start(ctx context.Context) {
	go m.collectMetrics(ctx)
	go m.checkWorkerHealth(ctx)
}

// collectMetrics periodically collects system metrics
func (m *Monitor) collectMetrics(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := m.updateMetrics(ctx); err != nil {
				log.Printf("Failed to update metrics: %v", err)
			}
		}
	}
}

// updateMetrics updates the current metrics
func (m *Monitor) updateMetrics(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Get queue depths
	queueDepth, err := m.queueProvider.GetQueueDepth()
	if err != nil {
		return fmt.Errorf("failed to get queue depth: %w", err)
	}
	m.metrics.QueueDepth = queueDepth

	dlqDepth, err := m.queueProvider.GetDLQDepth()
	if err != nil {
		return fmt.Errorf("failed to get DLQ depth: %w", err)
	}
	m.metrics.DLQDepth = dlqDepth

	// Get job statistics
	total, completed, failed, cancelled, err := m.repo.GetJobStats(ctx)
	if err != nil {
		return fmt.Errorf("failed to get job stats: %w", err)
	}
	m.metrics.TotalJobs = total
	m.metrics.CompletedJobs = completed
	m.metrics.FailedJobs = failed
	m.metrics.CancelledJobs = cancelled

	// Calculate active jobs
	m.metrics.ActiveJobs = int(total - completed - failed - cancelled)

	// Get average times
	avgWait, err := m.repo.GetAverageWaitTime(ctx)
	if err == nil {
		m.metrics.AverageWaitTime = avgWait
	}

	avgProcess, err := m.repo.GetAverageProcessTime(ctx)
	if err == nil {
		m.metrics.AverageProcessTime = avgProcess
	}

	// Get worker count
	workerCount, err := m.repo.GetActiveWorkers(ctx)
	if err == nil {
		m.metrics.WorkerCount = workerCount
	}

	// Count healthy workers
	healthyCount := 0
	for _, worker := range m.workers {
		if worker.Status == "healthy" {
			healthyCount++
		}
	}
	m.metrics.HealthyWorkers = healthyCount

	m.metrics.LastUpdated = time.Now()

	return nil
}

// checkWorkerHealth checks worker health status
func (m *Monitor) checkWorkerHealth(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.updateWorkerHealth()
		}
	}
}

// updateWorkerHealth updates worker health status
func (m *Monitor) updateWorkerHealth() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	for workerID, worker := range m.workers {
		// Mark worker as unhealthy if no heartbeat in 2 minutes
		if now.Sub(worker.LastHeartbeat) > 2*time.Minute {
			worker.Status = "unhealthy"
			log.Printf("Worker %s marked as unhealthy (no heartbeat)", workerID)
		}
	}
}

// RegisterWorkerHeartbeat registers a worker heartbeat
func (m *Monitor) RegisterWorkerHeartbeat(workerID, currentJob string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	worker, exists := m.workers[workerID]
	if !exists {
		worker = &WorkerHealth{
			WorkerID: workerID,
			Status:   "healthy",
		}
		m.workers[workerID] = worker
	}

	worker.LastHeartbeat = time.Now()
	worker.CurrentJob = currentJob
	worker.Status = "healthy"
}

// IncrementWorkerJobCount increments processed job count for a worker
func (m *Monitor) IncrementWorkerJobCount(workerID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if worker, exists := m.workers[workerID]; exists {
		worker.ProcessedJobs++
	}
}

// GetMetrics returns current system metrics
func (m *Monitor) GetMetrics() *Metrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Create a copy to avoid race conditions
	metrics := *m.metrics
	return &metrics
}

// GetWorkerHealth returns health status of all workers
func (m *Monitor) GetWorkerHealth() []*WorkerHealth {
	m.mu.RLock()
	defer m.mu.RUnlock()

	workers := make([]*WorkerHealth, 0, len(m.workers))
	for _, worker := range m.workers {
		// Create a copy
		w := *worker
		workers = append(workers, &w)
	}

	return workers
}

// GetSystemHealth returns overall system health
func (m *Monitor) GetSystemHealth() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Check various health indicators
	if m.metrics.DLQDepth > 100 {
		return "critical"
	}

	if m.metrics.QueueDepth > 1000 {
		return "warning"
	}

	healthyRatio := float64(m.metrics.HealthyWorkers) / float64(m.metrics.WorkerCount)
	if healthyRatio < 0.5 {
		return "critical"
	} else if healthyRatio < 0.8 {
		return "warning"
	}

	failureRate := float64(m.metrics.FailedJobs) / float64(m.metrics.TotalJobs)
	if failureRate > 0.1 {
		return "warning"
	}

	return "healthy"
}

// GetAlerts returns current system alerts
func (m *Monitor) GetAlerts() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var alerts []string

	if m.metrics.DLQDepth > 100 {
		alerts = append(alerts, fmt.Sprintf("High DLQ depth: %d messages", m.metrics.DLQDepth))
	}

	if m.metrics.QueueDepth > 1000 {
		alerts = append(alerts, fmt.Sprintf("High queue depth: %d jobs pending", m.metrics.QueueDepth))
	}

	if m.metrics.WorkerCount > 0 {
		healthyRatio := float64(m.metrics.HealthyWorkers) / float64(m.metrics.WorkerCount)
		if healthyRatio < 0.8 {
			alerts = append(alerts, fmt.Sprintf("Unhealthy workers: %d/%d",
				m.metrics.WorkerCount-m.metrics.HealthyWorkers, m.metrics.WorkerCount))
		}
	}

	if m.metrics.TotalJobs > 0 {
		failureRate := float64(m.metrics.FailedJobs) / float64(m.metrics.TotalJobs)
		if failureRate > 0.1 {
			alerts = append(alerts, fmt.Sprintf("High failure rate: %.1f%%", failureRate*100))
		}
	}

	return alerts
}
