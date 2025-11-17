package logging

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"
)

func TestNewLogger(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "JSON format to stdout",
			config: Config{
				Level:  "info",
				Format: "json",
				Output: "stdout",
			},
			wantErr: false,
		},
		{
			name: "Console format to stderr",
			config: Config{
				Level:  "debug",
				Format: "console",
				Output: "stderr",
			},
			wantErr: false,
		},
		{
			name: "Invalid log level defaults to info",
			config: Config{
				Level:  "invalid",
				Format: "json",
				Output: "stdout",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, err := NewLogger(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewLogger() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && logger == nil {
				t.Error("Expected non-nil logger")
			}
		})
	}
}

func TestLoggerMethods(t *testing.T) {
	var buf bytes.Buffer
	logger, err := NewLogger(Config{
		Level:  "debug",
		Format: "json",
		Output: "stdout",
	})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Test Info logging
	logger.Info("test info message")

	// Test Debug logging
	logger.Debug("test debug message")

	// Test Warn logging
	logger.Warn("test warn message")

	// Test Error logging
	logger.Error("test error message")

	// All methods should not panic
}

func TestLoggerWithFields(t *testing.T) {
	logger, err := NewLogger(Config{
		Level:  "info",
		Format: "json",
		Output: "stdout",
	})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Test WithField
	fieldLogger := logger.WithField("key", "value")
	if fieldLogger == nil {
		t.Error("Expected non-nil logger from WithField")
	}

	// Test WithFields
	fieldsLogger := logger.WithFields(map[string]interface{}{
		"key1": "value1",
		"key2": 123,
	})
	if fieldsLogger == nil {
		t.Error("Expected non-nil logger from WithFields")
	}

	// Test WithRequestID
	reqLogger := logger.WithRequestID("req-123")
	if reqLogger == nil {
		t.Error("Expected non-nil logger from WithRequestID")
	}

	// Test WithJobID
	jobLogger := logger.WithJobID("job-456")
	if jobLogger == nil {
		t.Error("Expected non-nil logger from WithJobID")
	}

	// Test WithVideoID
	videoLogger := logger.WithVideoID("video-789")
	if videoLogger == nil {
		t.Error("Expected non-nil logger from WithVideoID")
	}

	// Test WithWorkerID
	workerLogger := logger.WithWorkerID("worker-1")
	if workerLogger == nil {
		t.Error("Expected non-nil logger from WithWorkerID")
	}
}

func TestLogHTTPRequest(t *testing.T) {
	logger, err := NewLogger(Config{
		Level:  "info",
		Format: "json",
		Output: "stdout",
	})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	logger.LogHTTPRequest("GET", "/api/v1/videos", "192.168.1.1", 200, 100*time.Millisecond)
	// Should not panic
}

func TestLogJobEvent(t *testing.T) {
	logger, err := NewLogger(Config{
		Level:  "info",
		Format: "json",
		Output: "stdout",
	})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	logger.LogJobEvent("job-123", "started", "processing", map[string]interface{}{
		"resolution": "1080p",
		"codec":      "h264",
	})
	// Should not panic
}

func TestLogTranscodingProgress(t *testing.T) {
	logger, err := NewLogger(Config{
		Level:  "info",
		Format: "json",
		Output: "stdout",
	})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	logger.LogTranscodingProgress("job-123", 45.5, 30.0, 1.2)
	// Should not panic
}

func TestLogStorageOperation(t *testing.T) {
	logger, err := NewLogger(Config{
		Level:  "info",
		Format: "json",
		Output: "stdout",
	})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	logger.LogStorageOperation("upload", "videos", "test.mp4", 1048576, 2*time.Second, nil)
	// Should not panic
}

func TestLogDatabaseOperation(t *testing.T) {
	logger, err := NewLogger(Config{
		Level:  "info",
		Format: "json",
		Output: "stdout",
	})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	logger.LogDatabaseOperation("SELECT", 50*time.Millisecond, nil)
	// Should not panic
}

func TestLogGPUMetrics(t *testing.T) {
	logger, err := NewLogger(Config{
		Level:  "info",
		Format: "json",
		Output: "stdout",
	})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	logger.LogGPUMetrics("0", 85.5, 4096.0, 72.0)
	// Should not panic
}

func TestNewDefaultLogger(t *testing.T) {
	logger, err := NewDefaultLogger()
	if err != nil {
		t.Errorf("NewDefaultLogger() error = %v", err)
	}
	if logger == nil {
		t.Error("Expected non-nil logger from NewDefaultLogger")
	}
}

func TestNewConsoleLogger(t *testing.T) {
	logger, err := NewConsoleLogger()
	if err != nil {
		t.Errorf("NewConsoleLogger() error = %v", err)
	}
	if logger == nil {
		t.Error("Expected non-nil logger from NewConsoleLogger")
	}
}

func BenchmarkLogInfo(b *testing.B) {
	logger, _ := NewLogger(Config{
		Level:  "info",
		Format: "json",
		Output: "stdout",
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("benchmark message")
	}
}

func BenchmarkLogWithFields(b *testing.B) {
	logger, _ := NewLogger(Config{
		Level:  "info",
		Format: "json",
		Output: "stdout",
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.WithFields(map[string]interface{}{
			"key1": "value1",
			"key2": 123,
		}).Info("benchmark message")
	}
}
