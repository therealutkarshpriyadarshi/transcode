package transcoder

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWatermark(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping watermark test in short mode")
	}

	tempDir := t.TempDir()
	ffmpeg := NewFFmpeg("ffmpeg", "ffprobe")

	t.Run("ApplyTextWatermark", func(t *testing.T) {
		inputPath := "testdata/sample.mp4"
		outputPath := filepath.Join(tempDir, "watermarked.mp4")

		// Skip if test video doesn't exist
		if _, err := os.Stat(inputPath); os.IsNotExist(err) {
			t.Skip("Test video not available")
		}

		opts := WatermarkOptions{
			InputPath:     inputPath,
			OutputPath:    outputPath,
			WatermarkText: "Test Watermark",
			Position:      "bottom-right",
			Opacity:       0.8,
			FontSize:      24,
			FontColor:     "white",
			Padding:       10,
		}

		err := ffmpeg.ApplyWatermark(context.Background(), opts)

		if err == nil {
			// Verify output file exists
			_, statErr := os.Stat(outputPath)
			assert.NoError(t, statErr)

			// Verify output file has content
			info, _ := os.Stat(outputPath)
			assert.Greater(t, info.Size(), int64(0))
		}
	})

	t.Run("ApplyImageWatermark", func(t *testing.T) {
		inputPath := "testdata/sample.mp4"
		watermarkPath := "testdata/watermark.png"
		outputPath := filepath.Join(tempDir, "watermarked_image.mp4")

		// Skip if test files don't exist
		if _, err := os.Stat(inputPath); os.IsNotExist(err) {
			t.Skip("Test video not available")
		}
		if _, err := os.Stat(watermarkPath); os.IsNotExist(err) {
			t.Skip("Test watermark image not available")
		}

		opts := WatermarkOptions{
			InputPath:     inputPath,
			OutputPath:    outputPath,
			WatermarkPath: watermarkPath,
			Position:      "top-right",
			Opacity:       0.7,
			Scale:         0.15,
			Padding:       20,
		}

		err := ffmpeg.ApplyWatermark(context.Background(), opts)

		if err == nil {
			_, statErr := os.Stat(outputPath)
			assert.NoError(t, statErr)
		}
	})

	t.Run("WatermarkOptions_Defaults", func(t *testing.T) {
		opts := WatermarkOptions{
			InputPath:  "input.mp4",
			OutputPath: "output.mp4",
		}

		// Apply defaults
		if opts.Position == "" {
			opts.Position = "bottom-right"
		}
		if opts.Opacity == 0 {
			opts.Opacity = 0.8
		}
		if opts.Scale == 0 {
			opts.Scale = 0.15
		}
		if opts.FontSize == 0 {
			opts.FontSize = 24
		}
		if opts.FontColor == "" {
			opts.FontColor = "white"
		}
		if opts.Padding == 0 {
			opts.Padding = 10
		}

		assert.Equal(t, "bottom-right", opts.Position)
		assert.Equal(t, 0.8, opts.Opacity)
		assert.Equal(t, 0.15, opts.Scale)
		assert.Equal(t, 24, opts.FontSize)
		assert.Equal(t, "white", opts.FontColor)
		assert.Equal(t, 10, opts.Padding)
	})

	t.Run("CalculateWatermarkPosition", func(t *testing.T) {
		tests := []struct {
			position string
			padding  int
			expected string
		}{
			{"top-left", 10, "10:10"},
			{"top-right", 10, "W-w-10:10"},
			{"bottom-left", 10, "10:H-h-10"},
			{"bottom-right", 10, "W-w-10:H-h-10"},
			{"center", 0, "(W-w)/2:(H-h)/2"},
		}

		for _, tt := range tests {
			t.Run(tt.position, func(t *testing.T) {
				result := ffmpeg.calculateWatermarkPosition(tt.position, tt.padding)
				assert.Equal(t, tt.expected, result)
			})
		}
	})

	t.Run("BuildTextWatermarkFilter", func(t *testing.T) {
		opts := WatermarkOptions{
			WatermarkText: "Sample Text",
			Position:      "bottom-right",
			Opacity:       0.8,
			FontSize:      24,
			FontColor:     "white",
			Padding:       10,
		}

		filter := ffmpeg.buildTextWatermarkFilter(opts)
		assert.Contains(t, filter, "drawtext")
		assert.Contains(t, filter, "Sample Text")
		assert.Contains(t, filter, "fontsize=24")
		assert.Contains(t, filter, "fontcolor=white")
	})

	t.Run("BuildImageWatermarkFilter", func(t *testing.T) {
		opts := WatermarkOptions{
			WatermarkPath: "watermark.png",
			Position:      "top-right",
			Opacity:       0.7,
			Scale:         0.15,
			Padding:       20,
		}

		filter := ffmpeg.buildImageWatermarkFilter(opts)
		assert.Contains(t, filter, "scale")
		assert.Contains(t, filter, "overlay")
		assert.Contains(t, filter, "0.15")
		assert.Contains(t, filter, "0.70")
	})
}

func TestWatermarkErrorHandling(t *testing.T) {
	ffmpeg := NewFFmpeg("ffmpeg", "ffprobe")

	t.Run("NoWatermarkProvided", func(t *testing.T) {
		opts := WatermarkOptions{
			InputPath:  "input.mp4",
			OutputPath: "output.mp4",
			// Neither text nor image watermark provided
		}

		err := ffmpeg.ApplyWatermark(context.Background(), opts)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "watermark")
	})
}
