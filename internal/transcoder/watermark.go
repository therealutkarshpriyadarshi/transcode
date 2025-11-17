package transcoder

import (
	"context"
	"fmt"
	"os/exec"
)

// WatermarkOptions holds options for watermarking
type WatermarkOptions struct {
	InputPath      string
	OutputPath     string
	WatermarkPath  string  // Path to watermark image (PNG with transparency recommended)
	WatermarkText  string  // Text watermark (alternative to image)
	Position       string  // Position: "top-left", "top-right", "bottom-left", "bottom-right", "center"
	Opacity        float64 // Opacity: 0.0 (transparent) to 1.0 (opaque)
	Scale          float64 // Scale of watermark relative to video (0.1 to 1.0)
	FontSize       int     // Font size for text watermark
	FontColor      string  // Font color for text watermark (e.g., "white", "black")
	Padding        int     // Padding from edges in pixels
}

// ApplyWatermark adds a watermark (image or text) to a video
func (f *FFmpeg) ApplyWatermark(ctx context.Context, opts WatermarkOptions) error {
	// Set defaults
	if opts.Position == "" {
		opts.Position = "bottom-right"
	}
	if opts.Opacity == 0 {
		opts.Opacity = 0.8
	}
	if opts.Scale == 0 {
		opts.Scale = 0.15 // 15% of video width by default
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

	var filterComplex string

	if opts.WatermarkPath != "" {
		// Image watermark
		filterComplex = f.buildImageWatermarkFilter(opts)
	} else if opts.WatermarkText != "" {
		// Text watermark
		filterComplex = f.buildTextWatermarkFilter(opts)
	} else {
		return fmt.Errorf("either watermark image or text must be provided")
	}

	// Build FFmpeg command
	args := []string{
		"-i", opts.InputPath,
	}

	// Add watermark image as second input if using image watermark
	if opts.WatermarkPath != "" {
		args = append(args, "-i", opts.WatermarkPath)
	}

	args = append(args,
		"-filter_complex", filterComplex,
		"-c:a", "copy", // Copy audio without re-encoding
		"-y",
		opts.OutputPath,
	)

	cmd := exec.CommandContext(ctx, f.ffmpegPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("watermarking failed: %w, output: %s", err, string(output))
	}

	return nil
}

// buildImageWatermarkFilter builds the filter string for image watermark
func (f *FFmpeg) buildImageWatermarkFilter(opts WatermarkOptions) string {
	// Calculate position overlay
	position := f.calculateWatermarkPosition(opts.Position, opts.Padding)

	// Build filter:
	// 1. Scale watermark to appropriate size
	// 2. Adjust opacity
	// 3. Overlay on video at specified position

	filter := fmt.Sprintf(
		"[1:v]scale=iw*%.2f:-1,format=rgba,colorchannelmixer=aa=%.2f[wm];[0:v][wm]overlay=%s",
		opts.Scale,
		opts.Opacity,
		position,
	)

	return filter
}

// buildTextWatermarkFilter builds the filter string for text watermark
func (f *FFmpeg) buildTextWatermarkFilter(opts WatermarkOptions) string {
	// Calculate position
	var x, y string
	padding := opts.Padding

	switch opts.Position {
	case "top-left":
		x = fmt.Sprintf("%d", padding)
		y = fmt.Sprintf("%d", padding)
	case "top-right":
		x = fmt.Sprintf("w-tw-%d", padding)
		y = fmt.Sprintf("%d", padding)
	case "bottom-left":
		x = fmt.Sprintf("%d", padding)
		y = fmt.Sprintf("h-th-%d", padding)
	case "bottom-right":
		x = fmt.Sprintf("w-tw-%d", padding)
		y = fmt.Sprintf("h-th-%d", padding)
	case "center":
		x = "(w-tw)/2"
		y = "(h-th)/2"
	default:
		x = fmt.Sprintf("w-tw-%d", padding)
		y = fmt.Sprintf("h-th-%d", padding)
	}

	// Build drawtext filter
	// Note: alpha is 0-1 in drawtext, so we convert opacity
	alpha := opts.Opacity

	filter := fmt.Sprintf(
		"drawtext=text='%s':fontsize=%d:fontcolor=%s@%.2f:x=%s:y=%s",
		opts.WatermarkText,
		opts.FontSize,
		opts.FontColor,
		alpha,
		x,
		y,
	)

	return filter
}

// calculateWatermarkPosition returns the overlay position string for FFmpeg
func (f *FFmpeg) calculateWatermarkPosition(position string, padding int) string {
	switch position {
	case "top-left":
		return fmt.Sprintf("%d:%d", padding, padding)
	case "top-right":
		return fmt.Sprintf("W-w-%d:%d", padding, padding)
	case "bottom-left":
		return fmt.Sprintf("%d:H-h-%d", padding, padding)
	case "bottom-right":
		return fmt.Sprintf("W-w-%d:H-h-%d", padding, padding)
	case "center":
		return "(W-w)/2:(H-h)/2"
	default:
		return fmt.Sprintf("W-w-%d:H-h-%d", padding, padding) // Default to bottom-right
	}
}

// BatchWatermark applies watermark to multiple videos
func (f *FFmpeg) BatchWatermark(ctx context.Context, inputs []string, baseOpts WatermarkOptions, outputDir string) ([]string, error) {
	outputs := make([]string, 0, len(inputs))

	for i, input := range inputs {
		// Create output path
		outputPath := fmt.Sprintf("%s/watermarked_%d.mp4", outputDir, i)

		// Copy base options and set input/output
		opts := baseOpts
		opts.InputPath = input
		opts.OutputPath = outputPath

		// Apply watermark
		if err := f.ApplyWatermark(ctx, opts); err != nil {
			return outputs, fmt.Errorf("failed to watermark video %s: %w", input, err)
		}

		outputs = append(outputs, outputPath)
	}

	return outputs, nil
}
