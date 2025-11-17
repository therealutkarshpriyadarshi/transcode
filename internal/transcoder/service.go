package transcoder

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/therealutkarshpriyadarshi/transcode/internal/config"
	"github.com/therealutkarshpriyadarshi/transcode/internal/database"
	"github.com/therealutkarshpriyadarshi/transcode/internal/storage"
	"github.com/therealutkarshpriyadarshi/transcode/pkg/models"
)

// Service orchestrates transcoding operations
type Service struct {
	ffmpeg     *FFmpeg
	storage    *storage.Storage
	repo       *database.Repository
	cfg        config.TranscoderConfig
	workerID   string
}

// NewService creates a new transcoder service
func NewService(
	cfg config.TranscoderConfig,
	storage *storage.Storage,
	repo *database.Repository,
) *Service {
	return &Service{
		ffmpeg:   NewFFmpeg(cfg.FFmpegPath, cfg.FFprobePath),
		storage:  storage,
		repo:     repo,
		cfg:      cfg,
		workerID: uuid.New().String(),
	}
}

// ProcessJob processes a transcoding job
func (s *Service) ProcessJob(ctx context.Context, job *models.Job) error {
	// Update job status to processing
	job.Status = models.JobStatusProcessing
	job.WorkerID = s.workerID
	now := time.Now()
	job.StartedAt = &now
	job.Progress = 0

	if err := s.repo.UpdateJob(ctx, job); err != nil {
		return fmt.Errorf("failed to update job status: %w", err)
	}

	// Get video information
	video, err := s.repo.GetVideo(ctx, job.VideoID)
	if err != nil {
		return s.failJob(ctx, job, fmt.Errorf("failed to get video: %w", err))
	}

	// Create temporary directory
	tempDir := filepath.Join(s.cfg.TempDir, job.ID)
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return s.failJob(ctx, job, fmt.Errorf("failed to create temp directory: %w", err))
	}
	defer os.RemoveAll(tempDir)

	// Download source video
	inputPath := filepath.Join(tempDir, "input"+filepath.Ext(video.Filename))
	if err := s.storage.DownloadFile(ctx, video.OriginalURL, inputPath); err != nil {
		return s.failJob(ctx, job, fmt.Errorf("failed to download video: %w", err))
	}

	// Determine output format
	format := job.Config.OutputFormat
	if format == "" {
		format = "mp4"
	}

	outputFilename := fmt.Sprintf("output_%s.%s", job.Config.Resolution, format)
	outputPath := filepath.Join(tempDir, outputFilename)

	// Build transcode options
	opts := TranscodeOptions{
		InputPath:    inputPath,
		OutputPath:   outputPath,
		VideoCodec:   job.Config.Codec,
		AudioCodec:   job.Config.AudioCodec,
		Preset:       job.Config.Preset,
		Format:       format,
		ExtraArgs:    []string{},
	}

	// Set resolution if specified
	if job.Config.Resolution != "" {
		width, height := parseResolution(job.Config.Resolution)
		if width > 0 && height > 0 {
			opts.Width = width
			opts.Height = height
		}
	}

	// Set bitrates
	if job.Config.Bitrate > 0 {
		opts.VideoBitrate = fmt.Sprintf("%dk", job.Config.Bitrate/1000)
	}
	if job.Config.AudioBitrate > 0 {
		opts.AudioBitrate = fmt.Sprintf("%dk", job.Config.AudioBitrate)
	}

	// Transcode with progress tracking
	progressCallback := func(progress float64) {
		job.Progress = progress
		s.repo.UpdateJob(ctx, job)
	}

	if err := s.ffmpeg.Transcode(ctx, opts, progressCallback); err != nil {
		return s.failJob(ctx, job, fmt.Errorf("transcoding failed: %w", err))
	}

	// Get output file info
	outputInfo, err := os.Stat(outputPath)
	if err != nil {
		return s.failJob(ctx, job, fmt.Errorf("failed to stat output file: %w", err))
	}

	// Extract metadata from output
	outputMetadata, err := s.ffmpeg.ExtractVideoInfo(ctx, outputPath)
	if err != nil {
		return s.failJob(ctx, job, fmt.Errorf("failed to extract output metadata: %w", err))
	}

	// Upload output to storage
	storageKey := fmt.Sprintf("videos/%s/outputs/%s", video.ID, outputFilename)
	if err := s.storage.UploadFile(ctx, storageKey, outputPath); err != nil {
		return s.failJob(ctx, job, fmt.Errorf("failed to upload output: %w", err))
	}

	// Get URL for output
	url, err := s.storage.GetURL(ctx, storageKey)
	if err != nil {
		return s.failJob(ctx, job, fmt.Errorf("failed to get output URL: %w", err))
	}

	// Create output record
	output := &models.Output{
		JobID:      job.ID,
		VideoID:    video.ID,
		Format:     format,
		Resolution: job.Config.Resolution,
		Width:      outputMetadata.Width,
		Height:     outputMetadata.Height,
		Codec:      outputMetadata.Codec,
		Bitrate:    outputMetadata.Bitrate,
		Size:       outputInfo.Size(),
		Duration:   outputMetadata.Duration,
		URL:        url,
		Path:       storageKey,
	}

	if err := s.repo.CreateOutput(ctx, output); err != nil {
		return s.failJob(ctx, job, fmt.Errorf("failed to create output record: %w", err))
	}

	// Update job as completed
	job.Status = models.JobStatusCompleted
	job.Progress = 100
	completed := time.Now()
	job.CompletedAt = &completed

	if err := s.repo.UpdateJob(ctx, job); err != nil {
		return fmt.Errorf("failed to update job: %w", err)
	}

	// Update video status if all jobs are completed
	if err := s.updateVideoStatus(ctx, video.ID); err != nil {
		return fmt.Errorf("failed to update video status: %w", err)
	}

	return nil
}

// failJob marks a job as failed and updates the database
func (s *Service) failJob(ctx context.Context, job *models.Job, err error) error {
	job.Status = models.JobStatusFailed
	job.ErrorMsg = err.Error()
	completed := time.Now()
	job.CompletedAt = &completed

	if updateErr := s.repo.UpdateJob(ctx, job); updateErr != nil {
		return fmt.Errorf("failed to update job: %w (original error: %v)", updateErr, err)
	}

	return err
}

// updateVideoStatus updates video status based on job statuses
func (s *Service) updateVideoStatus(ctx context.Context, videoID string) error {
	jobs, err := s.repo.GetJobsByVideoID(ctx, videoID)
	if err != nil {
		return err
	}

	if len(jobs) == 0 {
		return nil
	}

	allCompleted := true
	anyFailed := false

	for _, job := range jobs {
		if job.Status == models.JobStatusPending || job.Status == models.JobStatusProcessing {
			allCompleted = false
		}
		if job.Status == models.JobStatusFailed {
			anyFailed = true
		}
	}

	video, err := s.repo.GetVideo(ctx, videoID)
	if err != nil {
		return err
	}

	if allCompleted {
		if anyFailed {
			video.Status = models.VideoStatusFailed
		} else {
			video.Status = models.VideoStatusCompleted
		}
	} else {
		video.Status = models.VideoStatusProcessing
	}

	return s.repo.UpdateVideo(ctx, video)
}

// parseResolution parses resolution strings like "1080p", "720p", etc.
func parseResolution(resolution string) (width, height int) {
	resolutions := map[string][2]int{
		"144p":  {256, 144},
		"240p":  {426, 240},
		"360p":  {640, 360},
		"480p":  {854, 480},
		"720p":  {1280, 720},
		"1080p": {1920, 1080},
		"1440p": {2560, 1440},
		"4k":    {3840, 2160},
		"2160p": {3840, 2160},
	}

	if res, ok := resolutions[resolution]; ok {
		return res[0], res[1]
	}

	return 0, 0
}
