package transcoder

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/therealutkarshpriyadarshi/transcode/pkg/models"
)

// HLSOptions holds options for HLS streaming
type HLSOptions struct {
	InputPath      string
	OutputDir      string
	Resolutions    []models.ResolutionProfile
	SegmentTime    int    // Segment duration in seconds (default: 6)
	PlaylistType   string // "vod" or "event"
	VideoCodec     string
	AudioCodec     string
	Preset         string
}

// HLSResult holds the result of HLS generation
type HLSResult struct {
	MasterPlaylistPath string
	VariantPlaylists   []HLSVariant
	SegmentDir         string
}

// HLSVariant represents a single HLS variant (resolution)
type HLSVariant struct {
	Resolution     models.ResolutionProfile
	PlaylistPath   string
	SegmentPattern string
	Bandwidth      int64
}

// GenerateHLS generates HLS manifests and segments for adaptive streaming
func (f *FFmpeg) GenerateHLS(ctx context.Context, opts HLSOptions, progressCB ProgressCallback) (*HLSResult, error) {
	if len(opts.Resolutions) == 0 {
		return nil, fmt.Errorf("no resolutions specified for HLS")
	}

	// Set defaults
	if opts.SegmentTime <= 0 {
		opts.SegmentTime = 6
	}
	if opts.PlaylistType == "" {
		opts.PlaylistType = "vod"
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

	result := &HLSResult{
		SegmentDir:       opts.OutputDir,
		VariantPlaylists: make([]HLSVariant, 0),
	}

	// Build FFmpeg command with multiple outputs
	args := []string{
		"-i", opts.InputPath,
		"-y",
	}

	// Add mapping and encoding options for each resolution
	for i, res := range opts.Resolutions {
		// Video encoding
		args = append(args,
			fmt.Sprintf("-map"), "0:v:0",
			fmt.Sprintf("-map"), "0:a:0",
			fmt.Sprintf("-c:v:%d", i), opts.VideoCodec,
			fmt.Sprintf("-c:a:%d", i), opts.AudioCodec,
			fmt.Sprintf("-b:v:%d", i), fmt.Sprintf("%d", res.VideoBitrate),
			fmt.Sprintf("-b:a:%d", i), fmt.Sprintf("%d", res.AudioBitrate),
			fmt.Sprintf("-s:v:%d", i), fmt.Sprintf("%dx%d", res.Width, res.Height),
			fmt.Sprintf("-preset:v:%d", i), opts.Preset,
			fmt.Sprintf("-maxrate:v:%d", i), fmt.Sprintf("%d", res.MaxBitrate),
			fmt.Sprintf("-bufsize:v:%d", i), fmt.Sprintf("%d", res.MaxBitrate*2),
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
	}

	// HLS-specific options
	args = append(args,
		"-f", "hls",
		"-hls_time", fmt.Sprintf("%d", opts.SegmentTime),
		"-hls_playlist_type", opts.PlaylistType,
		"-hls_flags", "independent_segments+temp_file",
		"-hls_segment_type", "mpegts",
	)

	// Master playlist and variant streams
	var varStreamMap []string
	for i, res := range opts.Resolutions {
		variantName := fmt.Sprintf("v:%d,a:%d,name:%s", i, i, res.Name)
		varStreamMap = append(varStreamMap, variantName)

		// Create variant playlist path
		playlistFilename := fmt.Sprintf("stream_%s.m3u8", res.Name)
		playlistPath := filepath.Join(opts.OutputDir, playlistFilename)

		result.VariantPlaylists = append(result.VariantPlaylists, HLSVariant{
			Resolution:     res,
			PlaylistPath:   playlistPath,
			SegmentPattern: fmt.Sprintf("stream_%s_%%03d.ts", res.Name),
			Bandwidth:      res.VideoBitrate + int64(res.AudioBitrate),
		})
	}

	args = append(args,
		"-var_stream_map", strings.Join(varStreamMap, " "),
		"-master_pl_name", "master.m3u8",
		"-hls_segment_filename", filepath.Join(opts.OutputDir, "stream_%v_%03d.ts"),
		filepath.Join(opts.OutputDir, "stream_%v.m3u8"),
	)

	// Execute FFmpeg command
	cmd := exec.CommandContext(ctx, f.ffmpegPath, args...)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if progressCB != nil {
		// Get total duration for progress tracking
		metadata, _ := f.ProbeVideo(ctx, opts.InputPath)
		if metadata != nil {
			// Could parse stderr for progress, but simplified for now
			progressCB(50) // Midway progress
		}
	}

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ffmpeg HLS generation failed: %w, stderr: %s", err, stderr.String())
	}

	if progressCB != nil {
		progressCB(100)
	}

	result.MasterPlaylistPath = filepath.Join(opts.OutputDir, "master.m3u8")

	return result, nil
}

// GenerateMasterPlaylist creates an HLS master playlist manually
func GenerateMasterPlaylist(variants []HLSVariant, outputPath string) error {
	var content strings.Builder

	content.WriteString("#EXTM3U\n")
	content.WriteString("#EXT-X-VERSION:3\n\n")

	for _, variant := range variants {
		// Stream info
		content.WriteString(fmt.Sprintf("#EXT-X-STREAM-INF:BANDWIDTH=%d,RESOLUTION=%dx%d,NAME=\"%s\"\n",
			variant.Bandwidth,
			variant.Resolution.Width,
			variant.Resolution.Height,
			variant.Resolution.Name,
		))
		content.WriteString(filepath.Base(variant.PlaylistPath) + "\n\n")
	}

	return os.WriteFile(outputPath, []byte(content.String()), 0644)
}
