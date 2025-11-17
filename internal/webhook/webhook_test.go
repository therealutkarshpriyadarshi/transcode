package webhook

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/therealutkarshpriyadarshi/transcode/pkg/models"
)

type mockRepository struct {
	webhooks   []*models.Webhook
	deliveries []*models.WebhookDelivery
}

func (m *mockRepository) GetWebhooksByEvent(ctx context.Context, event string) ([]*models.Webhook, error) {
	return m.webhooks, nil
}

func (m *mockRepository) CreateDelivery(ctx context.Context, delivery *models.WebhookDelivery) error {
	m.deliveries = append(m.deliveries, delivery)
	return nil
}

func (m *mockRepository) UpdateDelivery(ctx context.Context, delivery *models.WebhookDelivery) error {
	for i, d := range m.deliveries {
		if d.ID == delivery.ID {
			m.deliveries[i] = delivery
			return nil
		}
	}
	return nil
}

func (m *mockRepository) GetPendingDeliveries(ctx context.Context, limit int) ([]*models.WebhookDelivery, error) {
	return m.deliveries, nil
}

func TestWebhookNotify(t *testing.T) {
	// Create a test HTTP server
	receivedPayload := ""
	server := httptest.NewServer(http.HandlerFunc(func(w *http.ResponseWriter, r *http.Request) {
		body := make([]byte, r.ContentLength)
		r.Body.Read(body)
		receivedPayload = string(body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	repo := &mockRepository{
		webhooks: []*models.Webhook{
			{
				ID:     "webhook-1",
				UserID: "user-1",
				URL:    server.URL,
				Events: models.WebhookEvents{
					JobStarted: true,
				},
				IsActive: true,
			},
		},
		deliveries: []*models.WebhookDelivery{},
	}

	service := NewService(repo)

	job := &models.Job{
		ID:      "job-1",
		VideoID: "video-1",
		Status:  models.JobStatusProcessing,
	}

	err := service.NotifyJobStarted(context.Background(), job)
	assert.NoError(t, err)

	// Wait a bit for async delivery
	time.Sleep(100 * time.Millisecond)

	// Verify delivery was created
	assert.Len(t, repo.deliveries, 1)
}

func TestWebhookSignature(t *testing.T) {
	service := NewService(&mockRepository{})

	payload := []byte(`{"event":"test"}`)
	secret := "test-secret"

	signature := service.generateSignature(payload, secret)
	assert.NotEmpty(t, signature)
	assert.Contains(t, signature, "sha256=")
}

func TestWebhookEventMarshaling(t *testing.T) {
	event := models.WebhookEvent{
		Event:     models.WebhookEventJobStarted,
		Timestamp: time.Now(),
		Data: map[string]string{
			"job_id": "test-job",
		},
	}

	data, err := json.Marshal(event)
	assert.NoError(t, err)
	assert.NotEmpty(t, data)

	var unmarshaled models.WebhookEvent
	err = json.Unmarshal(data, &unmarshaled)
	assert.NoError(t, err)
	assert.Equal(t, event.Event, unmarshaled.Event)
}
