package models

import (
	"encoding/json"
	"testing"
)

func TestMetadataValue(t *testing.T) {
	meta := Metadata{
		"key1": "value1",
		"key2": 123,
	}

	value, err := meta.Value()
	if err != nil {
		t.Fatalf("Failed to get value: %v", err)
	}

	// Value should be JSON
	var result map[string]interface{}
	if err := json.Unmarshal(value.([]byte), &result); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if result["key1"] != "value1" {
		t.Errorf("Expected key1=value1, got %v", result["key1"])
	}
}

func TestMetadataScan(t *testing.T) {
	jsonData := []byte(`{"key1":"value1","key2":123}`)

	var meta Metadata
	if err := meta.Scan(jsonData); err != nil {
		t.Fatalf("Failed to scan: %v", err)
	}

	if meta["key1"] != "value1" {
		t.Errorf("Expected key1=value1, got %v", meta["key1"])
	}

	if val, ok := meta["key2"].(float64); !ok || val != 123 {
		t.Errorf("Expected key2=123, got %v", meta["key2"])
	}
}

func TestMetadataScanNil(t *testing.T) {
	var meta Metadata
	if err := meta.Scan(nil); err != nil {
		t.Fatalf("Failed to scan nil: %v", err)
	}

	if len(meta) != 0 {
		t.Error("Expected empty metadata after scanning nil")
	}
}

func TestTranscodeConfigValue(t *testing.T) {
	config := TranscodeConfig{
		OutputFormat: "mp4",
		Resolution:   "1080p",
		Bitrate:      5000000,
		Codec:        "h264",
	}

	value, err := config.Value()
	if err != nil {
		t.Fatalf("Failed to get value: %v", err)
	}

	// Value should be JSON
	var result TranscodeConfig
	if err := json.Unmarshal(value.([]byte), &result); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if result.OutputFormat != "mp4" {
		t.Errorf("Expected mp4, got %s", result.OutputFormat)
	}
}

func TestJobStatusConstants(t *testing.T) {
	statuses := []string{
		JobStatusPending,
		JobStatusQueued,
		JobStatusProcessing,
		JobStatusCompleted,
		JobStatusFailed,
		JobStatusCancelled,
	}

	for _, status := range statuses {
		if status == "" {
			t.Error("Job status constant is empty")
		}
	}
}

func TestVideoStatusConstants(t *testing.T) {
	statuses := []string{
		VideoStatusPending,
		VideoStatusProcessing,
		VideoStatusCompleted,
		VideoStatusFailed,
	}

	for _, status := range statuses {
		if status == "" {
			t.Error("Video status constant is empty")
		}
	}
}
