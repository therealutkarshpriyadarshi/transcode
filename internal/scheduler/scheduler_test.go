package scheduler

import (
	"container/heap"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/therealutkarshpriyadarshi/transcode/pkg/models"
)

func TestPriorityQueue(t *testing.T) {
	pq := &PriorityQueue{}
	heap.Init(pq)

	// Create jobs with different priorities
	jobs := []*models.Job{
		{ID: "job-1", Priority: 5},
		{ID: "job-2", Priority: 10},
		{ID: "job-3", Priority: 1},
		{ID: "job-4", Priority: 7},
	}

	// Push jobs to queue
	for _, job := range jobs {
		item := &QueueItem{
			Job:       job,
			Priority:  job.Priority,
			Timestamp: time.Now(),
		}
		heap.Push(pq, item)
	}

	assert.Equal(t, 4, pq.Len())

	// Pop jobs and verify they come out in priority order
	expectedOrder := []string{"job-2", "job-4", "job-1", "job-3"}
	for i, expectedID := range expectedOrder {
		item := heap.Pop(pq).(*QueueItem)
		assert.Equal(t, expectedID, item.Job.ID, "Job order mismatch at position %d", i)
	}

	assert.Equal(t, 0, pq.Len())
}

func TestPriorityQueueFIFO(t *testing.T) {
	pq := &PriorityQueue{}
	heap.Init(pq)

	baseTime := time.Now()

	// Create jobs with same priority but different timestamps
	jobs := []*QueueItem{
		{Job: &models.Job{ID: "job-1", Priority: 5}, Priority: 5, Timestamp: baseTime},
		{Job: &models.Job{ID: "job-2", Priority: 5}, Priority: 5, Timestamp: baseTime.Add(1 * time.Second)},
		{Job: &models.Job{ID: "job-3", Priority: 5}, Priority: 5, Timestamp: baseTime.Add(2 * time.Second)},
	}

	// Push jobs
	for _, item := range jobs {
		heap.Push(pq, item)
	}

	// Jobs with same priority should come out in FIFO order (earliest first)
	expectedOrder := []string{"job-1", "job-2", "job-3"}
	for i, expectedID := range expectedOrder {
		item := heap.Pop(pq).(*QueueItem)
		assert.Equal(t, expectedID, item.Job.ID, "FIFO order mismatch at position %d", i)
	}
}
