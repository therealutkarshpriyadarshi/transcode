package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/therealutkarshpriyadarshi/transcode/internal/config"
	"github.com/therealutkarshpriyadarshi/transcode/pkg/models"
)

const (
	TranscodeQueueName = "transcode_jobs"
	ExchangeName       = "transcode"
)

// Queue provides message queue operations
type Queue struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

// New creates a new queue client
func New(cfg config.QueueConfig) (*Queue, error) {
	url := fmt.Sprintf("amqp://%s:%s@%s:%d%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Vhost)

	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	channel, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	// Declare exchange
	err = channel.ExchangeDeclare(
		ExchangeName,
		"direct",
		true,  // durable
		false, // auto-deleted
		false, // internal
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		channel.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare exchange: %w", err)
	}

	// Declare queue
	_, err = channel.QueueDeclare(
		TranscodeQueueName,
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		channel.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare queue: %w", err)
	}

	// Bind queue to exchange
	err = channel.QueueBind(
		TranscodeQueueName,
		TranscodeQueueName,
		ExchangeName,
		false,
		nil,
	)
	if err != nil {
		channel.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to bind queue: %w", err)
	}

	return &Queue{
		conn:    conn,
		channel: channel,
	}, nil
}

// Close closes the queue connection
func (q *Queue) Close() error {
	if q.channel != nil {
		q.channel.Close()
	}
	if q.conn != nil {
		return q.conn.Close()
	}
	return nil
}

// PublishJob publishes a transcoding job to the queue
func (q *Queue) PublishJob(ctx context.Context, job *models.Job) error {
	body, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job: %w", err)
	}

	// Set priority based on job priority
	priority := uint8(job.Priority)
	if priority > 10 {
		priority = 10
	}

	err = q.channel.PublishWithContext(ctx,
		ExchangeName,
		TranscodeQueueName,
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "application/json",
			Body:         body,
			Timestamp:    time.Now(),
			Priority:     priority,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to publish job: %w", err)
	}

	return nil
}

// ConsumeJobs starts consuming jobs from the queue
func (q *Queue) ConsumeJobs(ctx context.Context, handler func(*models.Job) error) error {
	// Set QoS to limit concurrent processing
	err := q.channel.Qos(
		1,     // prefetch count
		0,     // prefetch size
		false, // global
	)
	if err != nil {
		return fmt.Errorf("failed to set QoS: %w", err)
	}

	msgs, err := q.channel.Consume(
		TranscodeQueueName,
		"",    // consumer
		false, // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)
	if err != nil {
		return fmt.Errorf("failed to register consumer: %w", err)
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

				if err := handler(&job); err != nil {
					// Requeue the message with a delay
					msg.Nack(false, true)
				} else {
					msg.Ack(false)
				}
			}
		}
	}()

	return nil
}

// GetQueueDepth returns the number of messages in the queue
func (q *Queue) GetQueueDepth() (int, error) {
	info, err := q.channel.QueueInspect(TranscodeQueueName)
	if err != nil {
		return 0, fmt.Errorf("failed to inspect queue: %w", err)
	}

	return info.Messages, nil
}
