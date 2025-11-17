package transcoder

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/therealutkarshpriyadarshi/transcode/pkg/models"
)

// FFmpeg wraps FFmpeg operations
type FFmpeg struct {
	ffmpegPath  string
	ffprobePath string
}

// NewFFmpeg creates a new FFmpeg instance
func NewFFmpeg(ffmpegPath, ffprobePath string) *FFmpeg {
	return &FFmpeg{
		ffmpegPath:  ffmpegPath,
		ffprobePath: ffprobePath,
	}
}

// VideoMetadata holds video metadata extracted from ffprobe
type VideoMetadata struct {
	Format   FormatInfo   `json:"format"`
	Streams  []StreamInfo `json:"streams"`
}

// FormatInfo holds format information
type FormatInfo struct {
	Filename   string `json:"filename"`
	FormatName string `json:"format_name"`
	Duration   string `json:"duration"`
	Size       string `json:"size"`
	BitRate    string `json:"bit_rate"`
}

// StreamInfo holds stream information
type StreamInfo struct {
	CodecType    string `json:"codec_type"`
	CodecName    string `json:"codec_name"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	BitRate      string `json:"bit_rate"`
	FrameRate    string `json:"r_frame_rate"`
	AvgFrameRate string `json:"avg_frame_rate"`
}

// ProbeVideo extracts metadata from a video file
func (f *FFmpeg) ProbeVideo(ctx context.Context, inputPath string) (*VideoMetadata, error) {
	args := []string{
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		inputPath,
	}

	cmd := exec.CommandContext(ctx, f.ffprobePath, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ffprobe failed: %w, stderr: %s", err, stderr.String())
	}

	var metadata VideoMetadata
	if err := json.Unmarshal(stdout.Bytes(), &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse ffprobe output: %w", err)
	}

	return &metadata, nil
}

// ExtractVideoInfo extracts basic video information
func (f *FFmpeg) ExtractVideoInfo(ctx context.Context, inputPath string) (*models.Video, error) {
	metadata, err := f.ProbeVideo(ctx, inputPath)
	if err != nil {
		return nil, err
	}

	video := &models.Video{
		Filename: metadata.Format.Filename,
		Metadata: make(models.Metadata),
	}

	// Parse duration
	if duration, err := strconv.ParseFloat(metadata.Format.Duration, 64); err == nil {
		video.Duration = duration
	}

	// Parse size
	if size, err := strconv.ParseInt(metadata.Format.Size, 10, 64); err == nil {
		video.Size = size
	}

	// Parse bitrate
	if bitrate, err := strconv.ParseInt(metadata.Format.BitRate, 10, 64); err == nil {
		video.Bitrate = bitrate
	}

	// Extract video stream information
	for _, stream := range metadata.Streams {
		if stream.CodecType == "video" {
			video.Width = stream.Width
			video.Height = stream.Height
			video.Codec = stream.CodecName

			// Parse frame rate
			if stream.AvgFrameRate != "" {
				parts := strings.Split(stream.AvgFrameRate, "/")
				if len(parts) == 2 {
					num, _ := strconv.ParseFloat(parts[0], 64)
					den, _ := strconv.ParseFloat(parts[1], 64)
					if den != 0 {
						video.FrameRate = num / den
					}
				}
			}
			break
		}
	}

	return video, nil
}

// TranscodeOptions holds transcoding options
type TranscodeOptions struct {
	InputPath    string
	OutputPath   string
	Width        int
	Height       int
	VideoBitrate string
	AudioBitrate string
	VideoCodec   string
	AudioCodec   string
	Preset       string
	Format       string
	ExtraArgs    []string
}

// ProgressCallback is called with progress updates
type ProgressCallback func(progress float64)

// Transcode transcodes a video file with progress tracking
func (f *FFmpeg) Transcode(ctx context.Context, opts TranscodeOptions, progressCB ProgressCallback) error {
	// Get total duration for progress calculation
	metadata, err := f.ProbeVideo(ctx, opts.InputPath)
	if err != nil {
		return fmt.Errorf("failed to probe video: %w", err)
	}

	totalDuration, _ := strconv.ParseFloat(metadata.Format.Duration, 64)

	// Build FFmpeg command
	args := []string{
		"-i", opts.InputPath,
		"-y", // overwrite output
	}

	// Video codec
	if opts.VideoCodec != "" {
		args = append(args, "-c:v", opts.VideoCodec)
	} else {
		args = append(args, "-c:v", "libx264")
	}

	// Video bitrate
	if opts.VideoBitrate != "" {
		args = append(args, "-b:v", opts.VideoBitrate)
	}

	// Resolution
	if opts.Width > 0 && opts.Height > 0 {
		args = append(args, "-s", fmt.Sprintf("%dx%d", opts.Width, opts.Height))
	}

	// Preset
	if opts.Preset != "" {
		args = append(args, "-preset", opts.Preset)
	} else {
		args = append(args, "-preset", "medium")
	}

	// Audio codec
	if opts.AudioCodec != "" {
		args = append(args, "-c:a", opts.AudioCodec)
	} else {
		args = append(args, "-c:a", "aac")
	}

	// Audio bitrate
	if opts.AudioBitrate != "" {
		args = append(args, "-b:a", opts.AudioBitrate)
	}

	// Extra arguments
	args = append(args, opts.ExtraArgs...)

	// Progress tracking
	args = append(args, "-progress", "pipe:1")

	// Output
	args = append(args, opts.OutputPath)

	cmd := exec.CommandContext(ctx, f.ffmpegPath, args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start ffmpeg: %w", err)
	}

	// Parse progress
	progressRegex := regexp.MustCompile(`out_time_ms=(\d+)`)
	scanner := bufio.NewScanner(stdout)

	go func() {
		for scanner.Scan() {
			line := scanner.Text()
			if matches := progressRegex.FindStringSubmatch(line); len(matches) > 1 {
				if timeMs, err := strconv.ParseFloat(matches[1], 64); err == nil {
					currentTime := timeMs / 1000000.0 // Convert to seconds
					if totalDuration > 0 {
						progress := (currentTime / totalDuration) * 100
						if progress > 100 {
							progress = 100
						}
						if progressCB != nil {
							progressCB(progress)
						}
					}
				}
			}
		}
	}()

	// Capture stderr for error reporting
	var stderrBuf bytes.Buffer
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			stderrBuf.WriteString(scanner.Text() + "\n")
		}
	}()

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("ffmpeg failed: %w, stderr: %s", err, stderrBuf.String())
	}

	// Final progress update
	if progressCB != nil {
		progressCB(100)
	}

	return nil
}

// ExtractThumbnail extracts a thumbnail from a video at a specific time
func (f *FFmpeg) ExtractThumbnail(ctx context.Context, inputPath, outputPath string, timeSeconds float64) error {
	args := []string{
		"-i", inputPath,
		"-ss", fmt.Sprintf("%.2f", timeSeconds),
		"-vframes", "1",
		"-q:v", "2",
		"-y",
		outputPath,
	}

	cmd := exec.CommandContext(ctx, f.ffmpegPath, args...)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to extract thumbnail: %w, stderr: %s", err, stderr.String())
	}

	return nil
}
