package scheduler

import (
	"container/heap"
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/therealutkarshpriyadarshi/transcode/pkg/models"
)

// JobScheduler manages job scheduling with priority and resource awareness
type JobScheduler struct {
	queue          *PriorityQueue
	mu             sync.RWMutex
	maxConcurrent  int
	activeJobs     int
	repo           Repository
	publisher      JobPublisher
	ctx            context.Context
	cancel         context.CancelFunc
}

// Repository defines the interface for job persistence
type Repository interface {
	GetPendingJobs(ctx context.Context, limit int) ([]*models.Job, error)
	UpdateJobStatus(ctx context.Context, jobID, status string) error
	GetJobByID(ctx context.Context, jobID string) (*models.Job, error)
}

// JobPublisher defines the interface for publishing jobs to queue
type JobPublisher interface {
	PublishJob(ctx context.Context, job *models.Job) error
}

// NewScheduler creates a new job scheduler
func NewScheduler(repo Repository, publisher JobPublisher, maxConcurrent int) *JobScheduler {
	ctx, cancel := context.WithCancel(context.Background())

	return &JobScheduler{
		queue:         &PriorityQueue{},
		maxConcurrent: maxConcurrent,
		activeJobs:    0,
		repo:          repo,
		publisher:     publisher,
		ctx:           ctx,
		cancel:        cancel,
	}
}

// Start begins the scheduler
func (s *JobScheduler) Start() error {
	heap.Init(s.queue)

	// Load pending jobs from database
	if err := s.loadPendingJobs(); err != nil {
		return fmt.Errorf("failed to load pending jobs: %w", err)
	}

	// Start scheduler loop
	go s.scheduleLoop()

	log.Println("Job scheduler started")
	return nil
}

// Stop stops the scheduler
func (s *JobScheduler) Stop() {
	s.cancel()
	log.Println("Job scheduler stopped")
}

// ScheduleJob adds a job to the scheduling queue
func (s *JobScheduler) ScheduleJob(job *models.Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	item := &QueueItem{
		Job:       job,
		Priority:  job.Priority,
		Timestamp: time.Now(),
	}

	heap.Push(s.queue, item)
	return nil
}

// loadPendingJobs loads pending jobs from the database
func (s *JobScheduler) loadPendingJobs() error {
	jobs, err := s.repo.GetPendingJobs(s.ctx, 1000)
	if err != nil {
		return err
	}

	for _, job := range jobs {
		if err := s.ScheduleJob(job); err != nil {
			log.Printf("Failed to schedule job %s: %v", job.ID, err)
		}
	}

	log.Printf("Loaded %d pending jobs", len(jobs))
	return nil
}

// scheduleLoop is the main scheduling loop
func (s *JobScheduler) scheduleLoop() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.processQueue()
		}
	}
}

// processQueue processes jobs from the priority queue
func (s *JobScheduler) processQueue() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if we can process more jobs
	for s.activeJobs < s.maxConcurrent && s.queue.Len() > 0 {
		item := heap.Pop(s.queue).(*QueueItem)

		// Publish job to worker queue
		if err := s.publisher.PublishJob(s.ctx, item.Job); err != nil {
			log.Printf("Failed to publish job %s: %v", item.Job.ID, err)
			// Re-queue the job
			heap.Push(s.queue, item)
			break
		}

		// Update job status to queued
		if err := s.repo.UpdateJobStatus(s.ctx, item.Job.ID, models.JobStatusQueued); err != nil {
			log.Printf("Failed to update job status %s: %v", item.Job.ID, err)
		}

		s.activeJobs++
		log.Printf("Scheduled job %s (priority: %d, active: %d/%d)",
			item.Job.ID, item.Priority, s.activeJobs, s.maxConcurrent)
	}
}

// JobCompleted notifies the scheduler that a job has completed
func (s *JobScheduler) JobCompleted(jobID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.activeJobs > 0 {
		s.activeJobs--
	}

	log.Printf("Job %s completed (active: %d/%d)", jobID, s.activeJobs, s.maxConcurrent)
}

// GetQueueDepth returns the current queue depth
func (s *JobScheduler) GetQueueDepth() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.queue.Len()
}

// GetActiveJobs returns the number of active jobs
func (s *JobScheduler) GetActiveJobs() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.activeJobs
}

// PriorityQueue implements a priority queue for jobs
type PriorityQueue []*QueueItem

// QueueItem represents a job in the priority queue
type QueueItem struct {
	Job       *models.Job
	Priority  int
	Timestamp time.Time
	Index     int
}

func (pq PriorityQueue) Len() int { return len(pq) }

func (pq PriorityQueue) Less(i, j int) bool {
	// Higher priority first
	if pq[i].Priority != pq[j].Priority {
		return pq[i].Priority > pq[j].Priority
	}
	// If same priority, FIFO (earlier timestamp first)
	return pq[i].Timestamp.Before(pq[j].Timestamp)
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].Index = i
	pq[j].Index = j
}

func (pq *PriorityQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*QueueItem)
	item.Index = n
	*pq = append(*pq, item)
}

func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.Index = -1
	*pq = old[0 : n-1]
	return item
}
