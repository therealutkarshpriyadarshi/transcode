package livestream

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/therealutkarshpriyadarshi/transcode/internal/database"
	"github.com/therealutkarshpriyadarshi/transcode/internal/storage"
	"github.com/therealutkarshpriyadarshi/transcode/pkg/models"
)

// Transcoder handles real-time transcoding of live streams
type Transcoder struct {
	ffmpegPath string
	repo       *database.Repository
	storage    *storage.Storage
}

// NewTranscoder creates a new live stream transcoder
func NewTranscoder(ffmpegPath string, repo *database.Repository, storage *storage.Storage) *Transcoder {
	return &Transcoder{
		ffmpegPath: ffmpegPath,
		repo:       repo,
		storage:    storage,
	}
}

// TranscodeOptions holds options for live stream transcoding
type TranscodeOptions struct {
	LiveStreamID    string
	InputURL        string
	OutputDir       string
	Settings        models.LiveStreamSettings
	DVREnabled      bool
	DVRWindow       int // in seconds
	LowLatency      bool
}

// TranscodeResult contains the result of a transcode operation
type TranscodeResult struct {
	MasterPlaylistPath string
	VariantPlaylists   []string
	Error              error
}

// StartTranscoding begins transcoding a live stream to HLS
func (t *Transcoder) StartTranscoding(ctx context.Context, opts TranscodeOptions) (*TranscodeResult, error) {
	log.Printf("Starting transcode for live stream: %s", opts.LiveStreamID)

	// Create output directory
	if err := os.MkdirAll(opts.OutputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	// Build FFmpeg command for multi-variant HLS
	cmd := t.buildFFmpegCommand(ctx, opts)

	// Start FFmpeg process
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start FFmpeg: %w", err)
	}

	// Monitor FFmpeg output in background
	go t.monitorFFmpegOutput(ctx, opts.LiveStreamID, stderr)

	// Create stream variants in database
	if err := t.createStreamVariants(ctx, opts); err != nil {
		log.Printf("Failed to create stream variants: %v", err)
	}

	// Wait for master playlist to be created
	masterPlaylistPath := filepath.Join(opts.OutputDir, "master.m3u8")
	if err := t.waitForFile(masterPlaylistPath, 30*time.Second); err != nil {
		return nil, fmt.Errorf("master playlist not created: %w", err)
	}

	// Update live stream with master playlist URL
	if err := t.updateMasterPlaylist(ctx, opts.LiveStreamID, masterPlaylistPath); err != nil {
		log.Printf("Failed to update master playlist: %v", err)
	}

	// Monitor process completion
	go func() {
		if err := cmd.Wait(); err != nil {
			log.Printf("FFmpeg process error for stream %s: %v", opts.LiveStreamID, err)
		}
		log.Printf("Transcoding completed for stream: %s", opts.LiveStreamID)
	}()

	return &TranscodeResult{
		MasterPlaylistPath: masterPlaylistPath,
	}, nil
}

// buildFFmpegCommand constructs the FFmpeg command for live transcoding
func (t *Transcoder) buildFFmpegCommand(ctx context.Context, opts TranscodeOptions) *exec.Cmd {
	settings := opts.Settings

	args := []string{
		"-i", opts.InputURL,
		"-y", // Overwrite output files
	}

	// Set up for multiple quality variants
	variants := t.getVariantConfigs(settings)

	// Video encoding settings for each variant
	for i, variant := range variants {
		// Video stream
		args = append(args,
			"-map", "0:v:0", // Map video stream
			fmt.Sprintf("-c:v:%d", i), t.getVideoCodec(settings.Codec, settings.GPUAcceleration),
			fmt.Sprintf("-b:v:%d", i), fmt.Sprintf("%dk", variant.Bitrate/1000),
			fmt.Sprintf("-maxrate:v:%d", i), fmt.Sprintf("%dk", variant.Bitrate/1000),
			fmt.Sprintf("-bufsize:v:%d", i), fmt.Sprintf("%dk", variant.Bitrate*2/1000),
			fmt.Sprintf("-s:v:%d", i), fmt.Sprintf("%dx%d", variant.Width, variant.Height),
			fmt.Sprintf("-r:%d", i), "30", // Frame rate
		)

		// GOP settings for better seeking
		if settings.KeyframeInterval > 0 {
			args = append(args,
				fmt.Sprintf("-g:%d", i), fmt.Sprintf("%d", settings.KeyframeInterval*30), // Keyframe every N seconds
				fmt.Sprintf("-keyint_min:%d", i), fmt.Sprintf("%d", settings.KeyframeInterval*30),
				fmt.Sprintf("-sc_threshold:%d", i), "0", // Disable scene detection for consistent GOPs
			)
		}

		// Audio stream
		args = append(args,
			"-map", "0:a:0", // Map audio stream
			fmt.Sprintf("-c:a:%d", i), settings.AudioCodec,
			fmt.Sprintf("-b:a:%d", i), fmt.Sprintf("%dk", settings.AudioBitrate),
		)
	}

	// HLS output settings
	segmentDuration := settings.SegmentDuration
	if segmentDuration == 0 {
		segmentDuration = 6 // Default 6 seconds
	}

	playlistLength := settings.PlaylistLength
	if playlistLength == 0 {
		playlistLength = 10 // Default 10 segments
	}

	args = append(args,
		"-f", "hls",
		"-hls_time", fmt.Sprintf("%d", segmentDuration),
		"-hls_list_size", fmt.Sprintf("%d", playlistLength),
		"-hls_flags", "delete_segments+independent_segments",
	)

	// Low-latency HLS settings
	if opts.LowLatency && settings.PartDuration > 0 {
		args = append(args,
			"-hls_flags", "delete_segments+independent_segments+program_date_time",
			"-hls_segment_type", "fmp4", // Use fragmented MP4 for LL-HLS
			"-hls_fmp4_init_filename", "init_%v.mp4",
			"-ldash", "1", // Enable low-latency mode
		)
	}

	// DVR settings
	if opts.DVREnabled {
		// Keep segments for DVR window
		dvrSegments := opts.DVRWindow / segmentDuration
		args = append(args,
			"-hls_list_size", fmt.Sprintf("%d", dvrSegments),
			"-hls_flags", "append_list+delete_segments",
		)
	}

	// Variant stream mapping
	var varStreamMap []string
	for i, variant := range variants {
		varStreamMap = append(varStreamMap,
			fmt.Sprintf("v:%d,a:%d,name:%s", i, i, variant.Resolution),
		)
	}
	args = append(args,
		"-var_stream_map", strings.Join(varStreamMap, " "),
		"-master_pl_name", "master.m3u8",
		"-hls_segment_filename", filepath.Join(opts.OutputDir, "%v_%03d.ts"),
		filepath.Join(opts.OutputDir, "%v.m3u8"),
	)

	return exec.CommandContext(ctx, t.ffmpegPath, args...)
}

// VariantConfig defines configuration for a streaming variant
type VariantConfig struct {
	Resolution string
	Width      int
	Height     int
	Bitrate    int64
	Codec      string
}

// getVariantConfigs returns the variant configurations based on settings
func (t *Transcoder) getVariantConfigs(settings models.LiveStreamSettings) []VariantConfig {
	variants := []VariantConfig{}

	for _, res := range settings.Resolutions {
		var config VariantConfig
		config.Resolution = res
		config.Codec = settings.Codec

		switch res {
		case "1080p":
			config.Width = 1920
			config.Height = 1080
			config.Bitrate = 5000000 // 5 Mbps
		case "720p":
			config.Width = 1280
			config.Height = 720
			config.Bitrate = 2800000 // 2.8 Mbps
		case "480p":
			config.Width = 854
			config.Height = 480
			config.Bitrate = 1400000 // 1.4 Mbps
		case "360p":
			config.Width = 640
			config.Height = 360
			config.Bitrate = 800000 // 800 Kbps
		case "240p":
			config.Width = 426
			config.Height = 240
			config.Bitrate = 400000 // 400 Kbps
		default:
			continue
		}

		variants = append(variants, config)
	}

	return variants
}

// getVideoCodec returns the appropriate video codec based on settings
func (t *Transcoder) getVideoCodec(codec string, gpuAccel bool) string {
	if gpuAccel {
		switch codec {
		case "h264":
			return "h264_nvenc"
		case "h265", "hevc":
			return "hevc_nvenc"
		}
	}

	switch codec {
	case "h264":
		return "libx264"
	case "h265", "hevc":
		return "libx265"
	case "vp9":
		return "libvpx-vp9"
	default:
		return "libx264"
	}
}

// monitorFFmpegOutput monitors FFmpeg output for progress and errors
func (t *Transcoder) monitorFFmpegOutput(ctx context.Context, streamID string, stderr bufio.Reader) {
	scanner := bufio.NewScanner(&stderr)
	frameRegex := regexp.MustCompile(`frame=\s*(\d+)`)
	bitrateRegex := regexp.MustCompile(`bitrate=\s*([\d.]+)kbits/s`)

	var lastUpdate time.Time

	for scanner.Scan() {
		line := scanner.Text()

		// Parse progress information
		if matches := frameRegex.FindStringSubmatch(line); len(matches) > 1 {
			frames, _ := strconv.Atoi(matches[1])

			// Update analytics periodically (every 10 seconds)
			if time.Since(lastUpdate) >= 10*time.Second {
				lastUpdate = time.Now()

				var bitrate int64
				if bitrateMatches := bitrateRegex.FindStringSubmatch(line); len(bitrateMatches) > 1 {
					bitrateFloat, _ := strconv.ParseFloat(bitrateMatches[1], 64)
					bitrate = int64(bitrateFloat * 1000) // Convert to bps
				}

				analytics := &models.LiveStreamAnalytics{
					LiveStreamID:  streamID,
					Timestamp:     time.Now(),
					IngestBitrate: bitrate,
					BufferHealth:  100,
					QualityScore:  95,
				}

				if err := t.repo.CreateLiveStreamAnalytics(ctx, analytics); err != nil {
					log.Printf("Failed to create analytics: %v", err)
				}

				log.Printf("Stream %s: frames=%d, bitrate=%dkbps", streamID, frames, bitrate/1000)
			}
		}

		// Check for errors
		if strings.Contains(line, "Error") || strings.Contains(line, "error") {
			log.Printf("FFmpeg error for stream %s: %s", streamID, line)

			event := &models.LiveStreamEvent{
				LiveStreamID: streamID,
				EventType:    models.LiveStreamEventError,
				Severity:     models.SeverityError,
				Message:      "FFmpeg error detected",
				Details:      models.Metadata{"error": line},
				Timestamp:    time.Now(),
			}

			if err := t.repo.CreateLiveStreamEvent(ctx, event); err != nil {
				log.Printf("Failed to log event: %v", err)
			}
		}
	}
}

// createStreamVariants creates variant records in the database
func (t *Transcoder) createStreamVariants(ctx context.Context, opts TranscodeOptions) error {
	variants := t.getVariantConfigs(opts.Settings)

	for _, variant := range variants {
		dbVariant := &models.LiveStreamVariant{
			LiveStreamID:   opts.LiveStreamID,
			Resolution:     variant.Resolution,
			Width:          variant.Width,
			Height:         variant.Height,
			Bitrate:        variant.Bitrate,
			FrameRate:      30.0,
			Codec:          variant.Codec,
			AudioBitrate:   opts.Settings.AudioBitrate,
			PlaylistURL:    fmt.Sprintf("%s/%s.m3u8", opts.OutputDir, variant.Resolution),
			SegmentPattern: fmt.Sprintf("%s/%s_%%03d.ts", opts.OutputDir, variant.Resolution),
		}

		if err := t.repo.CreateLiveStreamVariant(ctx, dbVariant); err != nil {
			return fmt.Errorf("failed to create variant %s: %w", variant.Resolution, err)
		}
	}

	return nil
}

// updateMasterPlaylist updates the live stream with the master playlist URL
func (t *Transcoder) updateMasterPlaylist(ctx context.Context, streamID, playlistPath string) error {
	// In production, you would upload this to storage and get a URL
	// For now, we'll use the local path
	return t.repo.UpdateLiveStreamMasterPlaylist(ctx, streamID, playlistPath)
}

// waitForFile waits for a file to be created
func (t *Transcoder) waitForFile(path string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("file %s not created within timeout", path)
}
