package transcoder

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
)

// AudioNormalizationOptions holds options for audio normalization
type AudioNormalizationOptions struct {
	InputPath     string
	OutputPath    string
	TargetLevel   float64 // Target loudness level in LUFS (default: -16.0)
	TruePeak      float64 // True peak in dBTP (default: -1.5)
	LoudnessRange float64 // Loudness range in LU (default: 11.0)
	DualPass      bool    // Use two-pass normalization for better results
}

// AudioInfo holds audio stream information
type AudioInfo struct {
	Codec         string
	Channels      int
	SampleRate    int
	Bitrate       int64
	Duration      float64
	Language      string
	Title         string
}

// ExtractAudioInfo extracts audio stream information from a video
func (f *FFmpeg) ExtractAudioInfo(ctx context.Context, inputPath string) ([]AudioInfo, error) {
	metadata, err := f.ProbeVideo(ctx, inputPath)
	if err != nil {
		return nil, err
	}

	var audioStreams []AudioInfo
	for _, stream := range metadata.Streams {
		if stream.CodecType == "audio" {
			info := AudioInfo{
				Codec: stream.CodecName,
			}
			// Parse bitrate if available
			fmt.Sscanf(stream.BitRate, "%d", &info.Bitrate)
			audioStreams = append(audioStreams, info)
		}
	}

	return audioStreams, nil
}

// NormalizeAudio normalizes audio levels using loudnorm filter
func (f *FFmpeg) NormalizeAudio(ctx context.Context, opts AudioNormalizationOptions) error {
	// Set defaults
	if opts.TargetLevel == 0 {
		opts.TargetLevel = -16.0
	}
	if opts.TruePeak == 0 {
		opts.TruePeak = -1.5
	}
	if opts.LoudnessRange == 0 {
		opts.LoudnessRange = 11.0
	}

	if opts.DualPass {
		return f.normalizeTwoPass(ctx, opts)
	}

	return f.normalizeSinglePass(ctx, opts)
}

// normalizeSinglePass performs single-pass audio normalization
func (f *FFmpeg) normalizeSinglePass(ctx context.Context, opts AudioNormalizationOptions) error {
	loudnormFilter := fmt.Sprintf("loudnorm=I=%.1f:TP=%.1f:LRA=%.1f",
		opts.TargetLevel, opts.TruePeak, opts.LoudnessRange)

	args := []string{
		"-i", opts.InputPath,
		"-af", loudnormFilter,
		"-c:v", "copy", // Copy video stream without re-encoding
		"-c:a", "aac",  // Re-encode audio
		"-b:a", "192k",
		"-y",
		opts.OutputPath,
	}

	cmd := exec.CommandContext(ctx, f.ffmpegPath, args...)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("audio normalization failed: %w, stderr: %s", err, stderr.String())
	}

	return nil
}

// normalizeTwoPass performs two-pass audio normalization for better results
func (f *FFmpeg) normalizeTwoPass(ctx context.Context, opts AudioNormalizationOptions) error {
	// First pass: measure loudness
	loudnormFilter := fmt.Sprintf("loudnorm=I=%.1f:TP=%.1f:LRA=%.1f:print_format=json",
		opts.TargetLevel, opts.TruePeak, opts.LoudnessRange)

	args := []string{
		"-i", opts.InputPath,
		"-af", loudnormFilter,
		"-f", "null",
		"-",
	}

	cmd := exec.CommandContext(ctx, f.ffmpegPath, args...)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("first pass failed: %w", err)
	}

	// Parse loudness measurements from stderr
	measurements := parseLoudnormMeasurements(stderr.String())
	if measurements == nil {
		// Fall back to single pass if parsing failed
		return f.normalizeSinglePass(ctx, opts)
	}

	// Second pass: apply normalization with measured values
	loudnormFilter2 := fmt.Sprintf("loudnorm=I=%.1f:TP=%.1f:LRA=%.1f:measured_I=%.2f:measured_TP=%.2f:measured_LRA=%.2f:measured_thresh=%.2f:offset=%.2f",
		opts.TargetLevel, opts.TruePeak, opts.LoudnessRange,
		measurements.InputI, measurements.InputTP, measurements.InputLRA,
		measurements.InputThresh, measurements.TargetOffset)

	args2 := []string{
		"-i", opts.InputPath,
		"-af", loudnormFilter2,
		"-c:v", "copy",
		"-c:a", "aac",
		"-b:a", "192k",
		"-y",
		opts.OutputPath,
	}

	cmd2 := exec.CommandContext(ctx, f.ffmpegPath, args2...)

	var stderr2 bytes.Buffer
	cmd2.Stderr = &stderr2

	if err := cmd2.Run(); err != nil {
		return fmt.Errorf("second pass failed: %w, stderr: %s", err, stderr2.String())
	}

	return nil
}

// LoudnormMeasurements holds loudness measurements from first pass
type LoudnormMeasurements struct {
	InputI       float64 `json:"input_i"`
	InputTP      float64 `json:"input_tp"`
	InputLRA     float64 `json:"input_lra"`
	InputThresh  float64 `json:"input_thresh"`
	TargetOffset float64 `json:"target_offset"`
}

// parseLoudnormMeasurements extracts loudness measurements from FFmpeg output
func parseLoudnormMeasurements(output string) *LoudnormMeasurements {
	// Find JSON block in output
	startIdx := -1
	endIdx := -1
	braceCount := 0

	for i, char := range output {
		if char == '{' {
			if startIdx == -1 {
				startIdx = i
			}
			braceCount++
		} else if char == '}' {
			braceCount--
			if braceCount == 0 && startIdx != -1 {
				endIdx = i + 1
				break
			}
		}
	}

	if startIdx == -1 || endIdx == -1 {
		return nil
	}

	jsonStr := output[startIdx:endIdx]

	var measurements LoudnormMeasurements
	if err := json.Unmarshal([]byte(jsonStr), &measurements); err != nil {
		return nil
	}

	return &measurements
}

// ExtractAudioTrack extracts an audio track to a separate file
func (f *FFmpeg) ExtractAudioTrack(ctx context.Context, inputPath, outputPath string, trackIndex int, codec string, bitrate int) error {
	if codec == "" {
		codec = "aac"
	}
	if bitrate <= 0 {
		bitrate = 128000
	}

	args := []string{
		"-i", inputPath,
		"-map", fmt.Sprintf("0:a:%d", trackIndex),
		"-c:a", codec,
		"-b:a", fmt.Sprintf("%d", bitrate),
		"-vn", // No video
		"-y",
		outputPath,
	}

	cmd := exec.CommandContext(ctx, f.ffmpegPath, args...)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("audio extraction failed: %w, stderr: %s", err, stderr.String())
	}

	return nil
}
