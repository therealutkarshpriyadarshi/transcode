package transcoder

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/therealutkarshpriyadarshi/transcode/pkg/models"
)

// DASHOptions holds options for DASH streaming
type DASHOptions struct {
	InputPath      string
	OutputDir      string
	Resolutions    []models.ResolutionProfile
	SegmentTime    int    // Segment duration in seconds (default: 4)
	VideoCodec     string
	AudioCodec     string
	Preset         string
	UseSingleFile  bool   // Use single file mode vs segment files
}

// DASHResult holds the result of DASH generation
type DASHResult struct {
	ManifestPath   string
	SegmentDir     string
	Representations []DASHRepresentation
}

// DASHRepresentation represents a single DASH representation (resolution)
type DASHRepresentation struct {
	Resolution models.ResolutionProfile
	ID         string
	Bandwidth  int64
	InitSegment string
	MediaTemplate string
}

// GenerateDASH generates DASH manifest (MPD) and segments for adaptive streaming
func (f *FFmpeg) GenerateDASH(ctx context.Context, opts DASHOptions, progressCB ProgressCallback) (*DASHResult, error) {
	if len(opts.Resolutions) == 0 {
		return nil, fmt.Errorf("no resolutions specified for DASH")
	}

	// Set defaults
	if opts.SegmentTime <= 0 {
		opts.SegmentTime = 4
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

	// Create output directory
	if err := os.MkdirAll(opts.OutputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	result := &DASHResult{
		SegmentDir:      opts.OutputDir,
		Representations: make([]DASHRepresentation, 0),
	}

	// Build FFmpeg command
	args := []string{
		"-i", opts.InputPath,
		"-y",
	}

	// Add mapping and encoding options for each resolution
	for i, res := range opts.Resolutions {
		// Video encoding
		args = append(args,
			"-map", "0:v:0",
			"-map", "0:a:0",
		)

		// Video settings for this output
		args = append(args,
			fmt.Sprintf("-c:v:%d", i), opts.VideoCodec,
			fmt.Sprintf("-b:v:%d", i), fmt.Sprintf("%d", res.VideoBitrate),
			fmt.Sprintf("-s:v:%d", i), fmt.Sprintf("%dx%d", res.Width, res.Height),
			fmt.Sprintf("-preset:v:%d", i), opts.Preset,
			fmt.Sprintf("-maxrate:v:%d", i), fmt.Sprintf("%d", res.MaxBitrate),
			fmt.Sprintf("-bufsize:v:%d", i), fmt.Sprintf("%d", res.MaxBitrate*2),
		)

		// Audio settings
		args = append(args,
			fmt.Sprintf("-c:a:%d", i), opts.AudioCodec,
			fmt.Sprintf("-b:a:%d", i), fmt.Sprintf("%d", res.AudioBitrate),
		)

		// Profile and level for H.264
		if opts.VideoCodec == "libx264" {
			if res.Height <= 480 {
				args = append(args, fmt.Sprintf("-profile:v:%d", i), "main")
			} else {
				args = append(args, fmt.Sprintf("-profile:v:%d", i), "high")
			}
			args = append(args, fmt.Sprintf("-level:v:%d", i), "4.0")
		}

		// Add to representations list
		result.Representations = append(result.Representations, DASHRepresentation{
			Resolution: res,
			ID:         fmt.Sprintf("video_%s", res.Name),
			Bandwidth:  res.VideoBitrate + int64(res.AudioBitrate),
		})
	}

	// DASH-specific options
	manifestPath := filepath.Join(opts.OutputDir, "manifest.mpd")

	args = append(args,
		"-f", "dash",
		"-seg_duration", fmt.Sprintf("%d", opts.SegmentTime),
		"-use_template", "1",
		"-use_timeline", "1",
		"-init_seg_name", "init-stream$RepresentationID$.m4s",
		"-media_seg_name", "chunk-stream$RepresentationID$-$Number%05d$.m4s",
	)

	if opts.UseSingleFile {
		args = append(args, "-single_file", "1")
	}

	args = append(args,
		"-adaptation_sets", "id=0,streams=v id=1,streams=a",
		manifestPath,
	)

	// Execute FFmpeg command
	cmd := exec.CommandContext(ctx, f.ffmpegPath, args...)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if progressCB != nil {
		progressCB(50) // Midway progress
	}

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ffmpeg DASH generation failed: %w, stderr: %s", err, stderr.String())
	}

	if progressCB != nil {
		progressCB(100)
	}

	result.ManifestPath = manifestPath

	return result, nil
}
