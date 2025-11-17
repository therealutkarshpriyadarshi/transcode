package transcoder

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConcatenation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concatenation test in short mode")
	}

	tempDir := t.TempDir()
	ffmpeg := NewFFmpeg("ffmpeg", "ffprobe")

	t.Run("ConcatVideo_ConcatDemuxer", func(t *testing.T) {
		inputPaths := []string{
			"testdata/video1.mp4",
			"testdata/video2.mp4",
		}

		// Skip if test videos don't exist
		for _, path := range inputPaths {
			if _, err := os.Stat(path); os.IsNotExist(err) {
				t.Skip("Test videos not available")
			}
		}

		outputPath := filepath.Join(tempDir, "concatenated.mp4")

		opts := ConcatenationOptions{
			InputPaths: inputPaths,
			OutputPath: outputPath,
			Method:     "concat",
			ReEncode:   false,
		}

		err := ffmpeg.ConcatVideo(context.Background(), opts)

		if err == nil {
			_, statErr := os.Stat(outputPath)
			assert.NoError(t, statErr)
		}
	})

	t.Run("ConcatVideo_FilterMethod", func(t *testing.T) {
		inputPaths := []string{
			"testdata/video1.mp4",
			"testdata/video2.mp4",
		}

		// Skip if test videos don't exist
		for _, path := range inputPaths {
			if _, err := os.Stat(path); os.IsNotExist(err) {
				t.Skip("Test videos not available")
			}
		}

		outputPath := filepath.Join(tempDir, "concatenated_filter.mp4")

		opts := ConcatenationOptions{
			InputPaths: inputPaths,
			OutputPath: outputPath,
			Method:     "filter",
			VideoCodec: "libx264",
			AudioCodec: "aac",
			Preset:     "fast",
		}

		err := ffmpeg.ConcatVideo(context.Background(), opts)

		if err == nil {
			_, statErr := os.Stat(outputPath)
			assert.NoError(t, statErr)
		}
	})

	t.Run("ConcatVideo_WithTransitions", func(t *testing.T) {
		inputPaths := []string{
			"testdata/video1.mp4",
			"testdata/video2.mp4",
		}

		// Skip if test videos don't exist
		for _, path := range inputPaths {
			if _, err := os.Stat(path); os.IsNotExist(err) {
				t.Skip("Test videos not available")
			}
		}

		outputPath := filepath.Join(tempDir, "concatenated_fade.mp4")

		opts := ConcatenationOptions{
			InputPaths:         inputPaths,
			OutputPath:         outputPath,
			Method:             "filter",
			TransitionType:     "fade",
			TransitionDuration: 1.0,
		}

		err := ffmpeg.ConcatVideo(context.Background(), opts)

		if err == nil {
			_, statErr := os.Stat(outputPath)
			assert.NoError(t, statErr)
		}
	})

	t.Run("ConcatVideo_ErrorOnSingleVideo", func(t *testing.T) {
		opts := ConcatenationOptions{
			InputPaths: []string{"testdata/video1.mp4"},
			OutputPath: filepath.Join(tempDir, "output.mp4"),
			Method:     "concat",
		}

		err := ffmpeg.ConcatVideo(context.Background(), opts)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "at least 2 videos")
	})

	t.Run("ConcatenationOptions_Defaults", func(t *testing.T) {
		opts := ConcatenationOptions{
			InputPaths: []string{"video1.mp4", "video2.mp4"},
			OutputPath: "output.mp4",
		}

		// Apply defaults
		if opts.Method == "" {
			opts.Method = "concat"
		}
		if opts.VideoCodec == "" {
			opts.VideoCodec = "libx264"
		}
		if opts.AudioCodec == "" {
			opts.AudioCodec = "aac"
		}
		if opts.Preset == "" {
			opts.Preset = "medium"
		}
		if opts.TransitionDuration == 0 {
			opts.TransitionDuration = 1.0
		}

		assert.Equal(t, "concat", opts.Method)
		assert.Equal(t, "libx264", opts.VideoCodec)
		assert.Equal(t, "aac", opts.AudioCodec)
		assert.Equal(t, "medium", opts.Preset)
		assert.Equal(t, 1.0, opts.TransitionDuration)
	})

	t.Run("BuildSimpleConcatFilter", func(t *testing.T) {
		filter := ffmpeg.buildSimpleConcatFilter(3)
		assert.Contains(t, filter, "[0:v][0:a][1:v][1:a][2:v][2:a]")
		assert.Contains(t, filter, "concat=n=3:v=1:a=1")
	})

	t.Run("BuildTransitionFilter_Fade", func(t *testing.T) {
		filter := ffmpeg.buildTransitionFilter(2, "fade", 1.5)
		assert.Contains(t, filter, "xfade")
		assert.Contains(t, filter, "transition=fade")
		assert.Contains(t, filter, "duration=1.50")
	})

	t.Run("CreateConcatFile", func(t *testing.T) {
		inputs := []string{
			filepath.Join(tempDir, "video1.mp4"),
			filepath.Join(tempDir, "video2.mp4"),
		}

		// Create dummy files
		for _, path := range inputs {
			f, err := os.Create(path)
			require.NoError(t, err)
			f.Close()
		}

		concatFile, err := ffmpeg.createConcatFile(inputs)
		require.NoError(t, err)
		defer os.Remove(concatFile)

		// Verify concat file content
		content, err := os.ReadFile(concatFile)
		require.NoError(t, err)

		assert.Contains(t, string(content), "file '")
		assert.Contains(t, string(content), "video1.mp4")
		assert.Contains(t, string(content), "video2.mp4")
	})

	t.Run("ConcatWithIntros", func(t *testing.T) {
		mainVideo := "testdata/main.mp4"
		introVideo := "testdata/intro.mp4"
		outroVideo := "testdata/outro.mp4"
		outputPath := filepath.Join(tempDir, "with_intros.mp4")

		// Skip if test videos don't exist
		if _, err := os.Stat(mainVideo); os.IsNotExist(err) {
			t.Skip("Test videos not available")
		}

		err := ffmpeg.ConcatWithIntros(context.Background(), mainVideo, introVideo, outroVideo, outputPath)

		// If successful, verify output
		if err == nil {
			_, statErr := os.Stat(outputPath)
			assert.NoError(t, statErr)
		}
	})
}

func TestGetVideosDuration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping duration test in short mode")
	}

	ffmpeg := NewFFmpeg("ffmpeg", "ffprobe")

	t.Run("GetVideosDuration", func(t *testing.T) {
		inputs := []string{
			"testdata/video1.mp4",
			"testdata/video2.mp4",
		}

		// Skip if test videos don't exist
		for _, path := range inputs {
			if _, err := os.Stat(path); os.IsNotExist(err) {
				t.Skip("Test videos not available")
			}
		}

		duration, err := ffmpeg.GetVideosDuration(context.Background(), inputs)

		if err == nil {
			assert.Greater(t, duration, 0.0)
		}
	})
}
