package transcoder

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ConcatenationOptions holds options for video concatenation
type ConcatenationOptions struct {
	InputPaths      []string
	OutputPath      string
	Method          string // "concat" (fast, requires same format) or "filter" (slower, handles different formats)
	TransitionType  string // "none", "fade", "dissolve" (only for filter method)
	TransitionDuration float64 // Transition duration in seconds
	ReEncode        bool   // Force re-encoding (slower but more compatible)
	VideoCodec      string // Output video codec (default: libx264)
	AudioCodec      string // Output audio codec (default: aac)
	Preset          string // Encoding preset (default: medium)
}

// ConcatVideo concatenates multiple videos into one
func (f *FFmpeg) ConcatVideo(ctx context.Context, opts ConcatenationOptions) error {
	if len(opts.InputPaths) < 2 {
		return fmt.Errorf("at least 2 videos are required for concatenation")
	}

	// Set defaults
	if opts.Method == "" {
		opts.Method = "concat" // Use concat demuxer by default (faster)
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

	switch opts.Method {
	case "concat":
		return f.concatDemuxer(ctx, opts)
	case "filter":
		return f.concatFilter(ctx, opts)
	default:
		return fmt.Errorf("unknown concatenation method: %s", opts.Method)
	}
}

// concatDemuxer uses FFmpeg's concat demuxer (fast, no re-encoding unless forced)
func (f *FFmpeg) concatDemuxer(ctx context.Context, opts ConcatenationOptions) error {
	// Create concat file list
	concatFile, err := f.createConcatFile(opts.InputPaths)
	if err != nil {
		return fmt.Errorf("failed to create concat file: %w", err)
	}
	defer os.Remove(concatFile)

	// Build FFmpeg command
	args := []string{
		"-f", "concat",
		"-safe", "0",
		"-i", concatFile,
	}

	if opts.ReEncode {
		// Re-encode with specified codecs
		args = append(args,
			"-c:v", opts.VideoCodec,
			"-preset", opts.Preset,
			"-c:a", opts.AudioCodec,
		)
	} else {
		// Copy streams without re-encoding (fast)
		args = append(args,
			"-c", "copy",
		)
	}

	args = append(args, "-y", opts.OutputPath)

	cmd := exec.CommandContext(ctx, f.ffmpegPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("concatenation failed: %w, output: %s", err, string(output))
	}

	return nil
}

// concatFilter uses FFmpeg's concat filter (slower, handles different formats, supports transitions)
func (f *FFmpeg) concatFilter(ctx context.Context, opts ConcatenationOptions) error {
	// Build filter complex
	var filterComplex string

	if opts.TransitionType != "" && opts.TransitionType != "none" {
		filterComplex = f.buildTransitionFilter(len(opts.InputPaths), opts.TransitionType, opts.TransitionDuration)
	} else {
		filterComplex = f.buildSimpleConcatFilter(len(opts.InputPaths))
	}

	// Build FFmpeg command
	args := []string{}

	// Add all inputs
	for _, input := range opts.InputPaths {
		args = append(args, "-i", input)
	}

	// Add filter complex
	args = append(args,
		"-filter_complex", filterComplex,
		"-c:v", opts.VideoCodec,
		"-preset", opts.Preset,
		"-c:a", opts.AudioCodec,
		"-y",
		opts.OutputPath,
	)

	cmd := exec.CommandContext(ctx, f.ffmpegPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("concatenation with filter failed: %w, output: %s", err, string(output))
	}

	return nil
}

// createConcatFile creates a text file listing all input videos for concat demuxer
func (f *FFmpeg) createConcatFile(inputs []string) (string, error) {
	// Create temp file
	tempFile, err := os.CreateTemp("", "concat_*.txt")
	if err != nil {
		return "", err
	}
	defer tempFile.Close()

	// Write file list
	for _, input := range inputs {
		// Convert to absolute path
		absPath, err := filepath.Abs(input)
		if err != nil {
			return "", err
		}

		// Write to concat file
		// Format: file '/path/to/file.mp4'
		_, err = tempFile.WriteString(fmt.Sprintf("file '%s'\n", absPath))
		if err != nil {
			return "", err
		}
	}

	return tempFile.Name(), nil
}

// buildSimpleConcatFilter builds a simple concatenation filter without transitions
func (f *FFmpeg) buildSimpleConcatFilter(numInputs int) string {
	// Build concat filter: [0:v][0:a][1:v][1:a]...[n:v][n:a]concat=n=N:v=1:a=1[outv][outa]
	var inputs strings.Builder

	for i := 0; i < numInputs; i++ {
		inputs.WriteString(fmt.Sprintf("[%d:v][%d:a]", i, i))
	}

	filter := fmt.Sprintf("%sconcat=n=%d:v=1:a=1[outv][outa]", inputs.String(), numInputs)
	return filter
}

// buildTransitionFilter builds a filter with video transitions
func (f *FFmpeg) buildTransitionFilter(numInputs int, transitionType string, duration float64) string {
	if numInputs < 2 {
		return f.buildSimpleConcatFilter(numInputs)
	}

	var filter strings.Builder

	// For fade/dissolve transitions, we use xfade filter
	// This is more complex and requires building a chain of xfade filters

	switch transitionType {
	case "fade", "dissolve":
		// Build xfade chain
		// [0:v][1:v]xfade=transition=fade:duration=1:offset=10[v01];
		// [v01][2:v]xfade=transition=fade:duration=1:offset=20[v02];
		// ...

		var offset float64 = 0
		prevLabel := "0:v"

		for i := 1; i < numInputs; i++ {
			currentLabel := fmt.Sprintf("v%02d", i)

			if i == 1 {
				filter.WriteString(fmt.Sprintf("[%s][%d:v]xfade=transition=%s:duration=%.2f:offset=%.2f[%s];",
					prevLabel, i, transitionType, duration, offset, currentLabel))
			} else {
				filter.WriteString(fmt.Sprintf("[%s][%d:v]xfade=transition=%s:duration=%.2f:offset=%.2f[%s];",
					prevLabel, i, transitionType, duration, offset, currentLabel))
			}

			prevLabel = currentLabel
			offset += 10.0 // Assume each video is at least 10 seconds (this should be calculated from actual duration)
		}

		// Audio mixing (simple concat for now)
		var audioInputs strings.Builder
		for i := 0; i < numInputs; i++ {
			audioInputs.WriteString(fmt.Sprintf("[%d:a]", i))
		}
		filter.WriteString(fmt.Sprintf("%sconcat=n=%d:v=0:a=1[outa]", audioInputs.String(), numInputs))

		// Map final video output
		finalVideoLabel := fmt.Sprintf("v%02d", numInputs-1)
		filter.WriteString(fmt.Sprintf(";[%s]null[outv]", finalVideoLabel))

	default:
		// Fall back to simple concat
		return f.buildSimpleConcatFilter(numInputs)
	}

	return filter.String()
}

// GetVideosDuration returns the total duration of all input videos
func (f *FFmpeg) GetVideosDuration(ctx context.Context, inputs []string) (float64, error) {
	var totalDuration float64

	for _, input := range inputs {
		info, err := f.ExtractVideoInfo(ctx, input)
		if err != nil {
			return 0, fmt.Errorf("failed to get duration for %s: %w", input, err)
		}
		totalDuration += info.Duration
	}

	return totalDuration, nil
}

// ConcatWithIntros creates a video with intro/outro
func (f *FFmpeg) ConcatWithIntros(ctx context.Context, mainVideo, introVideo, outroVideo, outputPath string) error {
	inputs := []string{}

	if introVideo != "" {
		inputs = append(inputs, introVideo)
	}

	inputs = append(inputs, mainVideo)

	if outroVideo != "" {
		inputs = append(inputs, outroVideo)
	}

	opts := ConcatenationOptions{
		InputPaths: inputs,
		OutputPath: outputPath,
		Method:     "concat",
		ReEncode:   true, // Re-encode to ensure compatibility
	}

	return f.ConcatVideo(ctx, opts)
}
