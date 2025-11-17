package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/therealutkarshpriyadarshi/transcode/pkg/models"
)

const (
	DeadLetterQueueName     = "transcode_jobs_dlq"
	DeadLetterExchangeName  = "transcode_dlq"
	RetryQueueName          = "transcode_jobs_retry"
	MaxRetries              = 5
)

// SetupDeadLetterQueue sets up the dead letter queue infrastructure
func (q *Queue) SetupDeadLetterQueue() error {
	// Declare dead letter exchange
	err := q.channel.ExchangeDeclare(
		DeadLetterExchangeName,
		"direct",
		true,  // durable
		false, // auto-deleted
		false, // internal
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare DLQ exchange: %w", err)
	}

	// Declare dead letter queue
	_, err = q.channel.QueueDeclare(
		DeadLetterQueueName,
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare DLQ: %w", err)
	}

	// Bind DLQ to exchange
	err = q.channel.QueueBind(
		DeadLetterQueueName,
		DeadLetterQueueName,
		DeadLetterExchangeName,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to bind DLQ: %w", err)
	}

	// Declare retry queue with TTL
	retryArgs := amqp.Table{
		"x-dead-letter-exchange":    ExchangeName,
		"x-dead-letter-routing-key": TranscodeQueueName,
		"x-message-ttl":             60000, // 1 minute TTL
	}

	_, err = q.channel.QueueDeclare(
		RetryQueueName,
		true,
		false,
		false,
		false,
		retryArgs,
	)
	if err != nil {
		return fmt.Errorf("failed to declare retry queue: %w", err)
	}

	log.Println("Dead letter queue infrastructure set up successfully")
	return nil
}

// PublishJobWithRetry publishes a job with retry support
func (q *Queue) PublishJobWithRetry(ctx context.Context, job *models.Job, retryCount int) error {
	body, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job: %w", err)
	}

	// Set priority based on job priority
	priority := uint8(job.Priority)
	if priority > 10 {
		priority = 10
	}

	headers := amqp.Table{
		"x-retry-count": retryCount,
	}

	err = q.channel.PublishWithContext(ctx,
		ExchangeName,
		TranscodeQueueName,
		false,
		false,
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "application/json",
			Body:         body,
			Timestamp:    time.Now(),
			Priority:     priority,
			Headers:      headers,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to publish job: %w", err)
	}

	return nil
}

// PublishToRetryQueue publishes a job to the retry queue
func (q *Queue) PublishToRetryQueue(ctx context.Context, job *models.Job, retryCount int) error {
	if retryCount >= MaxRetries {
		return q.PublishToDeadLetterQueue(ctx, job, "max retries exceeded")
	}

	body, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job: %w", err)
	}

	headers := amqp.Table{
		"x-retry-count": retryCount + 1,
	}

	// Calculate exponential backoff delay
	delay := calculateBackoffDelay(retryCount)

	err = q.channel.PublishWithContext(ctx,
		"",
		RetryQueueName,
		false,
		false,
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "application/json",
			Body:         body,
			Timestamp:    time.Now(),
			Headers:      headers,
			Expiration:   fmt.Sprintf("%d", delay.Milliseconds()),
		},
	)
	if err != nil {
		return fmt.Errorf("failed to publish to retry queue: %w", err)
	}

	log.Printf("Job %s queued for retry #%d in %v", job.ID, retryCount+1, delay)
	return nil
}

// PublishToDeadLetterQueue publishes a failed job to the dead letter queue
func (q *Queue) PublishToDeadLetterQueue(ctx context.Context, job *models.Job, reason string) error {
	body, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job: %w", err)
	}

	headers := amqp.Table{
		"x-failure-reason": reason,
		"x-failed-at":      time.Now().Format(time.RFC3339),
	}

	err = q.channel.PublishWithContext(ctx,
		DeadLetterExchangeName,
		DeadLetterQueueName,
		false,
		false,
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "application/json",
			Body:         body,
			Timestamp:    time.Now(),
			Headers:      headers,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to publish to DLQ: %w", err)
	}

	log.Printf("Job %s moved to dead letter queue: %s", job.ID, reason)
	return nil
}

// ConsumeDLQ consumes messages from the dead letter queue for manual processing
func (q *Queue) ConsumeDLQ(ctx context.Context, handler func(*models.Job, string) error) error {
	msgs, err := q.channel.Consume(
		DeadLetterQueueName,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to register DLQ consumer: %w", err)
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-msgs:
				if !ok {
					return
				}

				var job models.Job
				if err := json.Unmarshal(msg.Body, &job); err != nil {
					msg.Nack(false, false)
					continue
				}

				reason := ""
				if val, ok := msg.Headers["x-failure-reason"].(string); ok {
					reason = val
				}

				if err := handler(&job, reason); err != nil {
					msg.Nack(false, true)
				} else {
					msg.Ack(false)
				}
			}
		}
	}()

	return nil
}

// RetryFromDLQ retries a job from the dead letter queue
func (q *Queue) RetryFromDLQ(ctx context.Context, job *models.Job) error {
	return q.PublishJobWithRetry(ctx, job, 0)
}

// calculateBackoffDelay calculates exponential backoff delay
func calculateBackoffDelay(retryCount int) time.Duration {
	// Exponential backoff: 1min, 2min, 4min, 8min, 16min
	baseDelay := 1 * time.Minute
	delay := baseDelay * (1 << retryCount) // 2^retryCount

	// Cap at 1 hour
	if delay > 1*time.Hour {
		delay = 1 * time.Hour
	}

	return delay
}

// GetDLQDepth returns the number of messages in the dead letter queue
func (q *Queue) GetDLQDepth() (int, error) {
	info, err := q.channel.QueueInspect(DeadLetterQueueName)
	if err != nil {
		return 0, fmt.Errorf("failed to inspect DLQ: %w", err)
	}

	return info.Messages, nil
}
