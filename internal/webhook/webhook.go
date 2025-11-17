package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/therealutkarshpriyadarshi/transcode/pkg/models"
)

// Service handles webhook delivery and retry logic
type Service struct {
	client *http.Client
	repo   Repository
}

// Repository defines the interface for webhook persistence
type Repository interface {
	GetWebhooksByEvent(ctx context.Context, event string) ([]*models.Webhook, error)
	CreateDelivery(ctx context.Context, delivery *models.WebhookDelivery) error
	UpdateDelivery(ctx context.Context, delivery *models.WebhookDelivery) error
	GetPendingDeliveries(ctx context.Context, limit int) ([]*models.WebhookDelivery, error)
}

// NewService creates a new webhook service
func NewService(repo Repository) *Service {
	return &Service{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		repo: repo,
	}
}

// Notify sends a webhook notification for an event
func (s *Service) Notify(ctx context.Context, event string, data interface{}) error {
	webhooks, err := s.repo.GetWebhooksByEvent(ctx, event)
	if err != nil {
		return fmt.Errorf("failed to get webhooks: %w", err)
	}

	payload := models.WebhookEvent{
		Event:     event,
		Timestamp: time.Now(),
		Data:      data,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	for _, webhook := range webhooks {
		if !webhook.IsActive {
			continue
		}

		delivery := &models.WebhookDelivery{
			ID:         uuid.New().String(),
			WebhookID:  webhook.ID,
			Event:      event,
			Payload:    string(payloadBytes),
			Status:     models.WebhookDeliveryStatusPending,
			RetryCount: 0,
			CreatedAt:  time.Now(),
		}

		if err := s.repo.CreateDelivery(ctx, delivery); err != nil {
			log.Printf("Failed to create delivery: %v", err)
			continue
		}

		// Attempt immediate delivery in background
		go s.deliver(context.Background(), webhook, delivery, payloadBytes)
	}

	return nil
}

// deliver attempts to deliver a webhook
func (s *Service) deliver(ctx context.Context, webhook *models.Webhook, delivery *models.WebhookDelivery, payload []byte) {
	req, err := http.NewRequestWithContext(ctx, "POST", webhook.URL, bytes.NewReader(payload))
	if err != nil {
		s.markDeliveryFailed(ctx, delivery, 0, fmt.Sprintf("Failed to create request: %v", err))
		return
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Transcode-Webhook/1.0")
	req.Header.Set("X-Webhook-Event", delivery.Event)
	req.Header.Set("X-Webhook-Delivery", delivery.ID)

	// Add HMAC signature if secret is configured
	if webhook.Secret != "" {
		signature := s.generateSignature(payload, webhook.Secret)
		req.Header.Set("X-Webhook-Signature", signature)
	}

	// Send request
	resp, err := s.client.Do(req)
	if err != nil {
		s.markDeliveryFailed(ctx, delivery, 0, fmt.Sprintf("Failed to send request: %v", err))
		return
	}
	defer resp.Body.Close()

	// Read response body
	body, _ := io.ReadAll(resp.Body)

	// Check if delivery was successful
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		delivery.Status = models.WebhookDeliveryStatusDelivered
		delivery.StatusCode = resp.StatusCode
		delivery.ResponseBody = string(body)
		now := time.Now()
		delivery.CompletedAt = &now

		if err := s.repo.UpdateDelivery(ctx, delivery); err != nil {
			log.Printf("Failed to update delivery: %v", err)
		}
	} else {
		s.markDeliveryFailed(ctx, delivery, resp.StatusCode, string(body))
	}
}

// markDeliveryFailed marks a delivery as failed and schedules retry
func (s *Service) markDeliveryFailed(ctx context.Context, delivery *models.WebhookDelivery, statusCode int, responseBody string) {
	delivery.StatusCode = statusCode
	delivery.ResponseBody = responseBody
	delivery.RetryCount++

	// Calculate next retry time with exponential backoff
	// Retry delays: 1min, 5min, 15min, 1hr, 4hr, 12hr
	retryDelays := []time.Duration{
		1 * time.Minute,
		5 * time.Minute,
		15 * time.Minute,
		1 * time.Hour,
		4 * time.Hour,
		12 * time.Hour,
	}

	if delivery.RetryCount <= len(retryDelays) {
		nextRetry := time.Now().Add(retryDelays[delivery.RetryCount-1])
		delivery.NextRetryAt = &nextRetry
		delivery.Status = models.WebhookDeliveryStatusPending
	} else {
		// Max retries exceeded
		delivery.Status = models.WebhookDeliveryStatusFailed
		now := time.Now()
		delivery.CompletedAt = &now
	}

	if err := s.repo.UpdateDelivery(ctx, delivery); err != nil {
		log.Printf("Failed to update delivery: %v", err)
	}
}

// generateSignature generates HMAC-SHA256 signature for webhook payload
func (s *Service) generateSignature(payload []byte, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(payload)
	return "sha256=" + hex.EncodeToString(h.Sum(nil))
}

// RetryWorker processes pending webhook deliveries
func (s *Service) RetryWorker(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.retryPendingDeliveries(ctx)
		}
	}
}

// retryPendingDeliveries retries pending webhook deliveries
func (s *Service) retryPendingDeliveries(ctx context.Context) {
	deliveries, err := s.repo.GetPendingDeliveries(ctx, 100)
	if err != nil {
		log.Printf("Failed to get pending deliveries: %v", err)
		return
	}

	for _, delivery := range deliveries {
		// Skip if not ready for retry
		if delivery.NextRetryAt != nil && time.Now().Before(*delivery.NextRetryAt) {
			continue
		}

		// Get webhook configuration
		webhooks, err := s.repo.GetWebhooksByEvent(ctx, delivery.Event)
		if err != nil {
			log.Printf("Failed to get webhook for delivery %s: %v", delivery.ID, err)
			continue
		}

		var webhook *models.Webhook
		for _, wh := range webhooks {
			if wh.ID == delivery.WebhookID {
				webhook = wh
				break
			}
		}

		if webhook == nil || !webhook.IsActive {
			continue
		}

		// Retry delivery
		go s.deliver(context.Background(), webhook, delivery, []byte(delivery.Payload))
	}
}

// NotifyJobStarted sends notification when a job starts
func (s *Service) NotifyJobStarted(ctx context.Context, job *models.Job) error {
	return s.Notify(ctx, models.WebhookEventJobStarted, job)
}

// NotifyJobCompleted sends notification when a job completes
func (s *Service) NotifyJobCompleted(ctx context.Context, job *models.Job) error {
	return s.Notify(ctx, models.WebhookEventJobCompleted, job)
}

// NotifyJobFailed sends notification when a job fails
func (s *Service) NotifyJobFailed(ctx context.Context, job *models.Job) error {
	return s.Notify(ctx, models.WebhookEventJobFailed, job)
}

// NotifyJobProgress sends notification for job progress updates
func (s *Service) NotifyJobProgress(ctx context.Context, job *models.Job) error {
	return s.Notify(ctx, models.WebhookEventJobProgress, job)
}

// NotifyVideoUploaded sends notification when a video is uploaded
func (s *Service) NotifyVideoUploaded(ctx context.Context, video *models.Video) error {
	return s.Notify(ctx, models.WebhookEventVideoUploaded, video)
}
