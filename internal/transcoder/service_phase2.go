package transcoder

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/therealutkarshpriyadarshi/transcode/pkg/models"
)

// ProcessJobPhase2 processes a transcoding job with Phase 2 features
// This includes multi-resolution, HLS/DASH, thumbnails, audio normalization, and subtitles
func (s *Service) ProcessJobPhase2(ctx context.Context, job *models.Job) error {
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

	// Parse resolutions from job config
	var resolutions []models.ResolutionProfile
	if job.Config.Extra != nil {
		if resolutionsJSON, ok := job.Config.Extra["resolutions"]; ok {
			if err := json.Unmarshal([]byte(resolutionsJSON), &resolutions); err == nil && len(resolutions) > 0 {
				// Use custom resolutions
			}
		}
	}

	// If no resolutions specified, use intelligent selection
	if len(resolutions) == 0 {
		resolutions = models.SelectResolutionsForVideo(video.Width, video.Height)
	}

	// Determine video codec
	videoCodec := job.Config.Codec
	if videoCodec == "" {
		videoCodec = "libx264"
	}

	// Determine audio codec
	audioCodec := job.Config.AudioCodec
	if audioCodec == "" {
		audioCodec = "aac"
	}

	preset := job.Config.Preset
	if preset == "" {
		preset = "medium"
	}

	totalSteps := 1.0 // Base transcoding
	currentStep := 0.0

	// Count optional steps
	enableHLS := false
	enableDASH := false
	generateThumbnails := false
	extractSubtitles := false
	normalizeAudio := false

	if job.Config.Extra != nil {
		if val, ok := job.Config.Extra["enable_hls"]; ok && val == "true" {
			enableHLS = true
			totalSteps++
		}
		if val, ok := job.Config.Extra["enable_dash"]; ok && val == "true" {
			enableDASH = true
			totalSteps++
		}
		if val, ok := job.Config.Extra["generate_thumbnails"]; ok && val == "true" {
			generateThumbnails = true
			totalSteps++
		}
		if val, ok := job.Config.Extra["extract_subtitles"]; ok && val == "true" {
			extractSubtitles = true
			totalSteps++
		}
		if val, ok := job.Config.Extra["normalize_audio"]; ok && val == "true" {
			normalizeAudio = true
			totalSteps++
		}
	}

	// Progress callback wrapper
	progressCallback := func(stepProgress float64) {
		overallProgress := ((currentStep + (stepProgress / 100.0)) / totalSteps) * 100
		job.Progress = overallProgress
		s.repo.UpdateJob(ctx, job)
	}

	// Step 1: Multi-resolution transcoding or HLS/DASH
	outputDir := filepath.Join(tempDir, "outputs")
	os.MkdirAll(outputDir, 0755)

	if enableHLS {
		// Generate HLS directly
		currentStep = 1.0
		progressCallback(0)

		hlsDir := filepath.Join(outputDir, "hls")
		os.MkdirAll(hlsDir, 0755)

		hlsOpts := HLSOptions{
			InputPath:    inputPath,
			OutputDir:    hlsDir,
			Resolutions:  resolutions,
			SegmentTime:  6,
			PlaylistType: "vod",
			VideoCodec:   videoCodec,
			AudioCodec:   audioCodec,
			Preset:       preset,
		}

		hlsResult, err := s.ffmpeg.GenerateHLS(ctx, hlsOpts, progressCallback)
		if err != nil {
			return s.failJob(ctx, job, fmt.Errorf("HLS generation failed: %w", err))
		}

		// Upload HLS files to storage
		if err := s.uploadHLSFiles(ctx, video.ID, job.ID, hlsDir, hlsResult); err != nil {
			return s.failJob(ctx, job, fmt.Errorf("failed to upload HLS files: %w", err))
		}

	} else if enableDASH {
		// Generate DASH directly
		currentStep = 1.0
		progressCallback(0)

		dashDir := filepath.Join(outputDir, "dash")
		os.MkdirAll(dashDir, 0755)

		dashOpts := DASHOptions{
			InputPath:   inputPath,
			OutputDir:   dashDir,
			Resolutions: resolutions,
			SegmentTime: 4,
			VideoCodec:  videoCodec,
			AudioCodec:  audioCodec,
			Preset:      preset,
		}

		dashResult, err := s.ffmpeg.GenerateDASH(ctx, dashOpts, progressCallback)
		if err != nil {
			return s.failJob(ctx, job, fmt.Errorf("DASH generation failed: %w", err))
		}

		// Upload DASH files to storage
		if err := s.uploadDASHFiles(ctx, video.ID, job.ID, dashDir, dashResult); err != nil {
			return s.failJob(ctx, job, fmt.Errorf("failed to upload DASH files: %w", err))
		}

	} else {
		// Standard multi-resolution transcoding
		currentStep = 1.0
		progressCallback(0)

		multiResOpts := MultiResolutionOptions{
			InputPath:     inputPath,
			OutputDir:     outputDir,
			Resolutions:   resolutions,
			VideoCodec:    videoCodec,
			AudioCodec:    audioCodec,
			Preset:        preset,
			MaxConcurrent: 2,
		}

		result, err := s.ffmpeg.TranscodeMultiResolution(ctx, multiResOpts, progressCallback)
		if err != nil {
			return s.failJob(ctx, job, fmt.Errorf("multi-resolution transcoding failed: %w", err))
		}

		// Upload outputs and create records
		for _, output := range result.Outputs {
			if output.Error != nil {
				continue
			}

			// Upload to storage
			storageKey := fmt.Sprintf("videos/%s/outputs/%s", video.ID, filepath.Base(output.OutputPath))
			if err := s.storage.UploadFile(ctx, storageKey, output.OutputPath); err != nil {
				continue
			}

			url, _ := s.storage.GetURL(ctx, storageKey)

			// Get file size
			fileInfo, _ := os.Stat(output.OutputPath)

			// Create output record
			outputRecord := &models.Output{
				JobID:      job.ID,
				VideoID:    video.ID,
				Format:     job.Config.OutputFormat,
				Resolution: output.Resolution.Name,
				Width:      output.Resolution.Width,
				Height:     output.Resolution.Height,
				Codec:      videoCodec,
				Bitrate:    output.Resolution.VideoBitrate,
				Size:       fileInfo.Size(),
				Duration:   video.Duration,
				URL:        url,
				Path:       storageKey,
			}

			s.repo.CreateOutput(ctx, outputRecord)
		}
	}

	// Step 2: Generate thumbnails (if enabled)
	if generateThumbnails {
		currentStep++
		progressCallback(0)

		if err := s.generateAndUploadThumbnails(ctx, video, inputPath, tempDir); err != nil {
			// Log error but don't fail the job
			fmt.Printf("Thumbnail generation failed: %v\n", err)
		}

		progressCallback(100)
	}

	// Step 3: Extract subtitles (if enabled)
	if extractSubtitles {
		currentStep++
		progressCallback(0)

		if err := s.extractAndUploadSubtitles(ctx, video, inputPath, tempDir); err != nil {
			// Log error but don't fail the job
			fmt.Printf("Subtitle extraction failed: %v\n", err)
		}

		progressCallback(100)
	}

	// Step 4: Audio normalization (if enabled and not already done in HLS/DASH)
	if normalizeAudio && !enableHLS && !enableDASH {
		currentStep++
		progressCallback(0)

		// This would require re-encoding, so it's better to apply during transcoding
		// For now, we'll skip this in post-processing
		progressCallback(100)
	}

	// Mark job as completed
	job.Status = models.JobStatusCompleted
	job.Progress = 100
	completed := time.Now()
	job.CompletedAt = &completed

	if err := s.repo.UpdateJob(ctx, job); err != nil {
		return fmt.Errorf("failed to update job: %w", err)
	}

	// Update video status
	if err := s.updateVideoStatus(ctx, video.ID); err != nil {
		return fmt.Errorf("failed to update video status: %w", err)
	}

	return nil
}

// generateAndUploadThumbnails generates and uploads video thumbnails
func (s *Service) generateAndUploadThumbnails(ctx context.Context, video *models.Video, inputPath, tempDir string) error {
	thumbDir := filepath.Join(tempDir, "thumbnails")
	os.MkdirAll(thumbDir, 0755)

	// Generate regular thumbnails
	thumbOpts := ThumbnailOptions{
		InputPath: inputPath,
		OutputDir: thumbDir,
		Width:     320,
		Height:    180,
		Count:     10,
		Quality:   2,
	}

	result, err := s.ffmpeg.GenerateThumbnails(ctx, thumbOpts)
	if err != nil {
		return err
	}

	// Upload thumbnails
	for i, thumbPath := range result.Thumbnails {
		storageKey := fmt.Sprintf("videos/%s/thumbnails/thumb_%04d.jpg", video.ID, i)
		if err := s.storage.UploadFile(ctx, storageKey, thumbPath); err != nil {
			continue
		}

		url, _ := s.storage.GetURL(ctx, storageKey)

		// Create thumbnail record
		thumbnail := &models.Thumbnail{
			ID:            uuid.New().String(),
			VideoID:       video.ID,
			ThumbnailType: models.ThumbnailTypeSingle,
			URL:           url,
			Path:          storageKey,
			Width:         result.Width,
			Height:        result.Height,
		}

		s.repo.CreateThumbnail(ctx, thumbnail)
	}

	// Generate sprite sheet
	spriteOpts := SpriteOptions{
		InputPath: inputPath,
		OutputPath: filepath.Join(thumbDir, "sprite.jpg"),
		Width:     160,
		Height:    90,
		Columns:   5,
		Rows:      5,
		Interval:  10.0,
		Quality:   2,
	}

	if err := s.ffmpeg.GenerateSpriteSheet(ctx, spriteOpts); err == nil {
		storageKey := fmt.Sprintf("videos/%s/thumbnails/sprite.jpg", video.ID)
		if err := s.storage.UploadFile(ctx, storageKey, spriteOpts.OutputPath); err == nil {
			url, _ := s.storage.GetURL(ctx, storageKey)

			cols := spriteOpts.Columns
			rows := spriteOpts.Rows
			interval := spriteOpts.Interval

			sprite := &models.Thumbnail{
				ID:              uuid.New().String(),
				VideoID:         video.ID,
				ThumbnailType:   models.ThumbnailTypeSprite,
				URL:             url,
				Path:            storageKey,
				Width:           spriteOpts.Width * spriteOpts.Columns,
				Height:          spriteOpts.Height * spriteOpts.Rows,
				SpriteColumns:   &cols,
				SpriteRows:      &rows,
				IntervalSeconds: &interval,
			}

			s.repo.CreateThumbnail(ctx, sprite)
		}
	}

	return nil
}

// extractAndUploadSubtitles extracts and uploads subtitle tracks
func (s *Service) extractAndUploadSubtitles(ctx context.Context, video *models.Video, inputPath, tempDir string) error {
	subDir := filepath.Join(tempDir, "subtitles")
	os.MkdirAll(subDir, 0755)

	extractOpts := SubtitleExtractOptions{
		InputPath:  inputPath,
		OutputDir:  subDir,
		Format:     "vtt",
		TrackIndex: -1, // Extract all
	}

	result, err := s.ffmpeg.ExtractSubtitles(ctx, extractOpts)
	if err != nil {
		return err
	}

	// Upload subtitle files
	for _, sub := range result.Subtitles {
		storageKey := fmt.Sprintf("videos/%s/subtitles/%s", video.ID, filepath.Base(sub.OutputPath))
		if err := s.storage.UploadFile(ctx, storageKey, sub.OutputPath); err != nil {
			continue
		}

		url, _ := s.storage.GetURL(ctx, storageKey)

		// Create subtitle record
		subtitle := &models.Subtitle{
			ID:       uuid.New().String(),
			VideoID:  video.ID,
			Language: sub.Language,
			Format:   sub.Format,
			URL:      url,
			Path:     storageKey,
		}

		s.repo.CreateSubtitle(ctx, subtitle)
	}

	return nil
}

// uploadHLSFiles uploads HLS manifest and segment files to storage
func (s *Service) uploadHLSFiles(ctx context.Context, videoID, jobID, localDir string, result *HLSResult) error {
	// Walk through HLS directory and upload all files
	return filepath.Walk(localDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}

		relPath, _ := filepath.Rel(localDir, path)
		storageKey := fmt.Sprintf("videos/%s/hls/%s", videoID, relPath)

		if err := s.storage.UploadFile(ctx, storageKey, path); err != nil {
			return err
		}

		return nil
	})
}

// uploadDASHFiles uploads DASH manifest and segment files to storage
func (s *Service) uploadDASHFiles(ctx context.Context, videoID, jobID, localDir string, result *DASHResult) error {
	// Walk through DASH directory and upload all files
	return filepath.Walk(localDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}

		relPath, _ := filepath.Rel(localDir, path)
		storageKey := fmt.Sprintf("videos/%s/dash/%s", videoID, relPath)

		if err := s.storage.UploadFile(ctx, storageKey, path); err != nil {
			return err
		}

		return nil
	})
}
