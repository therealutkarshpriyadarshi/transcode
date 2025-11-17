package transcoder

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// ThumbnailOptions holds options for thumbnail generation
type ThumbnailOptions struct {
	InputPath    string
	OutputDir    string
	Width        int     // Thumbnail width (0 to maintain aspect ratio)
	Height       int     // Thumbnail height (0 to maintain aspect ratio)
	Count        int     // Number of thumbnails to generate
	Interval     float64 // Time interval between thumbnails (seconds)
	Quality      int     // JPEG quality (2-31, lower is better quality)
	Timestamps   []float64 // Specific timestamps to extract (optional)
}

// SpriteOptions holds options for sprite sheet generation
type SpriteOptions struct {
	InputPath    string
	OutputPath   string
	Width        int     // Individual thumbnail width
	Height       int     // Individual thumbnail height
	Columns      int     // Number of columns in sprite
	Rows         int     // Number of rows in sprite
	Interval     float64 // Time interval between thumbnails (seconds)
	Quality      int     // JPEG quality
}

// AnimatedPreviewOptions holds options for animated preview generation
type AnimatedPreviewOptions struct {
	InputPath    string
	OutputPath   string
	Width        int     // Preview width
	Height       int     // Preview height
	Duration     float64 // Duration of animated preview (seconds)
	FPS          int     // Frames per second
	StartTime    float64 // Start time in source video
}

// ThumbnailResult holds the result of thumbnail generation
type ThumbnailResult struct {
	Thumbnails []string
	Width      int
	Height     int
}

// GenerateThumbnails generates multiple thumbnails from a video
func (f *FFmpeg) GenerateThumbnails(ctx context.Context, opts ThumbnailOptions) (*ThumbnailResult, error) {
	if err := os.MkdirAll(opts.OutputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	// Set defaults
	if opts.Quality <= 0 {
		opts.Quality = 2
	}
	if opts.Width == 0 && opts.Height == 0 {
		opts.Width = 320
	}

	result := &ThumbnailResult{
		Thumbnails: make([]string, 0),
		Width:      opts.Width,
		Height:     opts.Height,
	}

	// If specific timestamps are provided, use them
	if len(opts.Timestamps) > 0 {
		for i, timestamp := range opts.Timestamps {
			outputPath := filepath.Join(opts.OutputDir, fmt.Sprintf("thumb_%04d.jpg", i))
			if err := f.ExtractThumbnail(ctx, opts.InputPath, outputPath, timestamp); err != nil {
				return nil, fmt.Errorf("failed to extract thumbnail at %.2f: %w", timestamp, err)
			}
			result.Thumbnails = append(result.Thumbnails, outputPath)
		}
		return result, nil
	}

	// Otherwise, generate thumbnails at intervals
	if opts.Count <= 0 {
		opts.Count = 10
	}

	// Get video duration to calculate intervals
	metadata, err := f.ProbeVideo(ctx, opts.InputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to probe video: %w", err)
	}

	duration := 0.0
	fmt.Sscanf(metadata.Format.Duration, "%f", &duration)

	if opts.Interval <= 0 {
		opts.Interval = duration / float64(opts.Count+1)
	}

	// Generate thumbnails at calculated intervals
	for i := 0; i < opts.Count; i++ {
		timestamp := opts.Interval * float64(i+1)
		if timestamp >= duration {
			break
		}

		outputPath := filepath.Join(opts.OutputDir, fmt.Sprintf("thumb_%04d.jpg", i))

		args := []string{
			"-ss", fmt.Sprintf("%.2f", timestamp),
			"-i", opts.InputPath,
			"-vframes", "1",
			"-q:v", fmt.Sprintf("%d", opts.Quality),
		}

		if opts.Width > 0 || opts.Height > 0 {
			scale := fmt.Sprintf("%d:%d", opts.Width, opts.Height)
			args = append(args, "-vf", fmt.Sprintf("scale=%s", scale))
		}

		args = append(args, "-y", outputPath)

		cmd := exec.CommandContext(ctx, f.ffmpegPath, args...)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr

		if err := cmd.Run(); err != nil {
			return nil, fmt.Errorf("failed to generate thumbnail: %w, stderr: %s", err, stderr.String())
		}

		result.Thumbnails = append(result.Thumbnails, outputPath)
	}

	return result, nil
}

// GenerateSpriteSheet generates a sprite sheet of thumbnails
func (f *FFmpeg) GenerateSpriteSheet(ctx context.Context, opts SpriteOptions) error {
	if opts.Columns <= 0 {
		opts.Columns = 5
	}
	if opts.Rows <= 0 {
		opts.Rows = 5
	}
	if opts.Quality <= 0 {
		opts.Quality = 2
	}
	if opts.Width <= 0 {
		opts.Width = 160
	}
	if opts.Height <= 0 {
		opts.Height = 90
	}

	// Calculate FPS for sprite generation
	// If interval is 10 seconds, fps should be 1/10 = 0.1
	fps := 1.0 / opts.Interval
	if fps <= 0 {
		fps = 0.1 // Default to one frame every 10 seconds
	}

	// Build tile layout string
	tile := fmt.Sprintf("%dx%d", opts.Columns, opts.Rows)

	args := []string{
		"-i", opts.InputPath,
		"-vf", fmt.Sprintf("fps=%f,scale=%d:%d,tile=%s", fps, opts.Width, opts.Height, tile),
		"-frames:v", "1",
		"-q:v", fmt.Sprintf("%d", opts.Quality),
		"-y",
		opts.OutputPath,
	}

	cmd := exec.CommandContext(ctx, f.ffmpegPath, args...)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to generate sprite sheet: %w, stderr: %s", err, stderr.String())
	}

	return nil
}

// GenerateAnimatedPreview generates an animated GIF preview
func (f *FFmpeg) GenerateAnimatedPreview(ctx context.Context, opts AnimatedPreviewOptions) error {
	if opts.Duration <= 0 {
		opts.Duration = 5.0
	}
	if opts.FPS <= 0 {
		opts.FPS = 10
	}
	if opts.Width <= 0 {
		opts.Width = 480
	}

	args := []string{
		"-ss", fmt.Sprintf("%.2f", opts.StartTime),
		"-i", opts.InputPath,
		"-t", fmt.Sprintf("%.2f", opts.Duration),
		"-vf", fmt.Sprintf("fps=%d,scale=%d:-1:flags=lanczos", opts.FPS, opts.Width),
		"-y",
		opts.OutputPath,
	}

	cmd := exec.CommandContext(ctx, f.ffmpegPath, args...)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to generate animated preview: %w, stderr: %s", err, stderr.String())
	}

	return nil
}
