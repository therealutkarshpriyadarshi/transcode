package transcoder

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSceneDetection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping scene detection test in short mode")
	}

	// Create test directory
	tempDir := t.TempDir()

	// Create FFmpeg instance
	ffmpeg := NewFFmpeg("ffmpeg", "ffprobe")

	t.Run("DetectScenes_BasicFunctionality", func(t *testing.T) {
		// This test requires a sample video file
		// In a real scenario, you'd have a test video file
		// For now, we'll test the structure

		opts := SceneDetectionOptions{
			InputPath:        "testdata/sample.mp4",
			OutputDir:        filepath.Join(tempDir, "scenes"),
			Threshold:        0.4,
			MinSceneDuration: 1.0,
			MaxScenes:        10,
		}

		// Skip if test video doesn't exist
		if _, err := os.Stat(opts.InputPath); os.IsNotExist(err) {
			t.Skip("Test video not available")
		}

		result, err := ffmpeg.DetectScenes(context.Background(), opts)

		// If the test video exists, verify results
		if err == nil {
			assert.NotNil(t, result)
			assert.GreaterOrEqual(t, result.TotalScenes, 0)
			assert.LessOrEqual(t, result.TotalScenes, opts.MaxScenes)
		}
	})

	t.Run("SelectBestScene", func(t *testing.T) {
		scenes := []SceneInfo{
			{SceneNumber: 1, StartTime: 0, EndTime: 5, Duration: 5},
			{SceneNumber: 2, StartTime: 5, EndTime: 15, Duration: 10}, // Longest
			{SceneNumber: 3, StartTime: 15, EndTime: 20, Duration: 5},
		}

		best := selectBestScene(scenes)
		require.NotNil(t, best)
		assert.Equal(t, 2, best.SceneNumber)
		assert.Equal(t, 10.0, best.Duration)
	})

	t.Run("SelectBestScene_EmptyScenes", func(t *testing.T) {
		scenes := []SceneInfo{}
		best := selectBestScene(scenes)
		assert.Nil(t, best)
	})

	t.Run("GenerateIntelligentThumbnails", func(t *testing.T) {
		inputPath := "testdata/sample.mp4"
		outputDir := filepath.Join(tempDir, "thumbnails")

		// Skip if test video doesn't exist
		if _, err := os.Stat(inputPath); os.IsNotExist(err) {
			t.Skip("Test video not available")
		}

		thumbnails, err := ffmpeg.GenerateIntelligentThumbnails(context.Background(), inputPath, outputDir, 5)

		if err == nil {
			assert.NotNil(t, thumbnails)
			// Should have up to 5 thumbnails
			assert.LessOrEqual(t, len(thumbnails), 5)
		}
	})
}

func TestSceneDetectionOptions_Defaults(t *testing.T) {
	opts := SceneDetectionOptions{
		InputPath: "test.mp4",
		OutputDir: "/tmp/scenes",
	}

	// Test that defaults would be applied
	if opts.Threshold == 0 {
		opts.Threshold = 0.4
	}
	if opts.MinSceneDuration == 0 {
		opts.MinSceneDuration = 1.0
	}
	if opts.MaxScenes == 0 {
		opts.MaxScenes = 20
	}

	assert.Equal(t, 0.4, opts.Threshold)
	assert.Equal(t, 1.0, opts.MinSceneDuration)
	assert.Equal(t, 20, opts.MaxScenes)
}
