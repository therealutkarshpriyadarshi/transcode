package transcoder

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/therealutkarshpriyadarshi/transcode/pkg/models"
)

// Phase4Service extends Service with GPU acceleration and optimization features
type Phase4Service struct {
	*Service
	gpuManager *GPUManager
	useGPU     bool
	twoPass    bool
}

// NewPhase4Service creates a new Phase 4 service with GPU support
func NewPhase4Service(service *Service, enableGPU bool, enableTwoPass bool) *Phase4Service {
	gpuManager := NewGPUManager(service.cfg.FFmpegPath)

	// Check if GPU is actually available
	useGPU := enableGPU && gpuManager.IsAvailable()

	return &Phase4Service{
		Service:    service,
		gpuManager: gpuManager,
		useGPU:     useGPU,
		twoPass:    enableTwoPass,
	}
}

// ProcessJobWithGPU processes a transcoding job with GPU acceleration and optimizations
func (s *Phase4Service) ProcessJobWithGPU(ctx context.Context, job *models.Job) error {
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

	// Check if we should use GPU for this job
	useGPU := s.shouldUseGPU(ctx, job)

	// Select GPU if available
	var gpuIndex int = -1
	if useGPU {
		gpuIndex, err = s.gpuManager.SelectBestGPU(ctx)
		if err != nil {
			// Fallback to CPU
			useGPU = false
		}
	}

	// Get optimal codec
	codec := s.gpuManager.GetOptimalCodec(job.Config.Codec, useGPU)

	// Build transcode options
	opts := TranscodeOptions{
		InputPath:    inputPath,
		OutputPath:   outputPath,
		VideoCodec:   codec,
		AudioCodec:   job.Config.AudioCodec,
		Preset:       job.Config.Preset,
		Format:       format,
		ExtraArgs:    []string{},
	}

	// Add GPU arguments if using GPU
	if useGPU && gpuIndex >= 0 {
		gpuArgs := s.gpuManager.BuildGPUArgs(gpuIndex, codec, job.Config.Preset)
		opts.ExtraArgs = append(opts.ExtraArgs, gpuArgs...)

		// Update job metadata to indicate GPU usage
		if job.Metadata == nil {
			job.Metadata = make(models.Metadata)
		}
		job.Metadata["gpu_enabled"] = true
		job.Metadata["gpu_device"] = gpuIndex
		job.Metadata["gpu_codec"] = codec
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

	// Progress callback
	progressCallback := func(progress float64) {
		job.Progress = progress
		s.repo.UpdateJob(ctx, job)
	}

	// Perform transcoding (with two-pass if enabled)
	var transcodeErr error
	if s.twoPass && !useGPU && (codec == "libx264" || codec == "libx265") {
		// Two-pass encoding for better quality (CPU only)
		transcodeErr = s.TranscodeTwoPass(ctx, opts, progressCallback)
	} else {
		// Single-pass encoding
		transcodeErr = s.ffmpeg.Transcode(ctx, opts, progressCallback)
	}

	if transcodeErr != nil {
		// If GPU encoding failed, try CPU fallback
		if useGPU {
			return s.retryWithCPUFallback(ctx, job, video, tempDir, inputPath, outputPath, format, progressCallback)
		}
		return s.failJob(ctx, job, fmt.Errorf("transcoding failed: %w", transcodeErr))
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

	// Upload output to storage with optimized upload
	storageKey := fmt.Sprintf("videos/%s/outputs/%s", video.ID, outputFilename)
	if err := s.uploadWithOptimization(ctx, storageKey, outputPath); err != nil {
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

	// Calculate processing time
	if job.StartedAt != nil {
		processingTime := completed.Sub(*job.StartedAt).Seconds()
		if job.Metadata == nil {
			job.Metadata = make(models.Metadata)
		}
		job.Metadata["processing_time_seconds"] = processingTime
		if video.Duration > 0 {
			job.Metadata["processing_speed"] = video.Duration / processingTime
		}
	}

	if err := s.repo.UpdateJob(ctx, job); err != nil {
		return fmt.Errorf("failed to update job: %w", err)
	}

	// Update video status if all jobs are completed
	if err := s.updateVideoStatus(ctx, video.ID); err != nil {
		return fmt.Errorf("failed to update video status: %w", err)
	}

	return nil
}

// shouldUseGPU determines if GPU should be used for a job
func (s *Phase4Service) shouldUseGPU(ctx context.Context, job *models.Job) bool {
	if !s.useGPU {
		return false
	}

	// Check if job explicitly requests CPU
	if job.Config.Preset == "cpu" {
		return false
	}

	// Check minimum memory requirement (estimate 500MB per encode)
	if !s.gpuManager.CheckMinMemory(500) {
		return false
	}

	// GPU is beneficial for most codecs except VP9 and AV1
	codec := job.Config.Codec
	if codec == "libvpx-vp9" || codec == "libaom-av1" {
		return false
	}

	return true
}

// retryWithCPUFallback retries transcoding with CPU after GPU failure
func (s *Phase4Service) retryWithCPUFallback(
	ctx context.Context,
	job *models.Job,
	video *models.Video,
	tempDir string,
	inputPath string,
	outputPath string,
	format string,
	progressCallback ProgressCallback,
) error {
	// Get CPU codec
	cpuCodec := s.gpuManager.GetOptimalCodec(job.Config.Codec, false)

	// Build CPU transcode options
	opts := TranscodeOptions{
		InputPath:    inputPath,
		OutputPath:   outputPath,
		VideoCodec:   cpuCodec,
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

	// Update job metadata
	if job.Metadata == nil {
		job.Metadata = make(models.Metadata)
	}
	job.Metadata["gpu_fallback"] = true
	job.Metadata["cpu_codec"] = cpuCodec

	// Retry with CPU
	if err := s.ffmpeg.Transcode(ctx, opts, progressCallback); err != nil {
		return s.failJob(ctx, job, fmt.Errorf("CPU fallback transcoding failed: %w", err))
	}

	return nil
}

// TranscodeTwoPass performs two-pass encoding for better quality
func (s *Phase4Service) TranscodeTwoPass(ctx context.Context, opts TranscodeOptions, progressCB ProgressCallback) error {
	// Get total duration for progress calculation
	metadata, err := s.ffmpeg.ProbeVideo(ctx, opts.InputPath)
	if err != nil {
		return fmt.Errorf("failed to probe video: %w", err)
	}

	// First pass - analysis
	pass1Opts := opts
	pass1Opts.ExtraArgs = append(pass1Opts.ExtraArgs,
		"-pass", "1",
		"-passlogfile", filepath.Join(filepath.Dir(opts.OutputPath), "ffmpeg2pass"),
		"-f", "null",
	)
	pass1Opts.OutputPath = "/dev/null"
	if _, err := os.Stat("/dev/null"); os.IsNotExist(err) {
		// Windows
		pass1Opts.OutputPath = "NUL"
	}

	// Progress for first pass (0-50%)
	pass1Progress := func(progress float64) {
		if progressCB != nil {
			progressCB(progress * 0.5)
		}
	}

	if err := s.ffmpeg.Transcode(ctx, pass1Opts, pass1Progress); err != nil {
		return fmt.Errorf("first pass failed: %w", err)
	}

	// Second pass - actual encoding
	pass2Opts := opts
	pass2Opts.ExtraArgs = append(pass2Opts.ExtraArgs,
		"-pass", "2",
		"-passlogfile", filepath.Join(filepath.Dir(opts.OutputPath), "ffmpeg2pass"),
	)

	// Progress for second pass (50-100%)
	pass2Progress := func(progress float64) {
		if progressCB != nil {
			progressCB(50 + (progress * 0.5))
		}
	}

	if err := s.ffmpeg.Transcode(ctx, pass2Opts, pass2Progress); err != nil {
		return fmt.Errorf("second pass failed: %w", err)
	}

	// Cleanup pass log files
	passLogBase := filepath.Join(filepath.Dir(opts.OutputPath), "ffmpeg2pass")
	os.Remove(passLogBase + "-0.log")
	os.Remove(passLogBase + "-0.log.mbtree")

	if progressCB != nil {
		progressCB(100)
	}

	return nil
}

// uploadWithOptimization uploads file with optimized settings
func (s *Phase4Service) uploadWithOptimization(ctx context.Context, key, filePath string) error {
	// For now, use standard upload
	// In production, this could use parallel multipart upload for large files
	return s.storage.UploadFile(ctx, key, filePath)
}

// GetGPUStatus returns current GPU status
func (s *Phase4Service) GetGPUStatus(ctx context.Context) (*GPUStatus, error) {
	capability := s.gpuManager.GetCapability()

	status := &GPUStatus{
		Available:      capability.Available,
		NVENCSupported: capability.NVENCSupported,
		DeviceCount:    capability.DeviceCount,
		Devices:        make([]GPUDeviceStatus, 0),
	}

	if !capability.Available {
		return status, nil
	}

	// Get memory usage
	memInfo, err := s.gpuManager.GetMemoryUsage(ctx)
	if err != nil {
		return status, fmt.Errorf("failed to get memory usage: %w", err)
	}

	for i, info := range memInfo {
		deviceName := ""
		if i < len(capability.DeviceNames) {
			deviceName = capability.DeviceNames[i]
		}

		status.Devices = append(status.Devices, GPUDeviceStatus{
			Index:          info.DeviceIndex,
			Name:           deviceName,
			MemoryTotal:    info.MemoryTotal,
			MemoryUsed:     info.MemoryUsed,
			MemoryFree:     info.MemoryFree,
			Utilization:    info.GPUUtilization,
		})
	}

	return status, nil
}

// GPUStatus represents overall GPU status
type GPUStatus struct {
	Available      bool              `json:"available"`
	NVENCSupported bool              `json:"nvenc_supported"`
	DeviceCount    int               `json:"device_count"`
	Devices        []GPUDeviceStatus `json:"devices"`
}

// GPUDeviceStatus represents individual GPU device status
type GPUDeviceStatus struct {
	Index       int     `json:"index"`
	Name        string  `json:"name"`
	MemoryTotal int64   `json:"memory_total_mb"`
	MemoryUsed  int64   `json:"memory_used_mb"`
	MemoryFree  int64   `json:"memory_free_mb"`
	Utilization float64 `json:"utilization_percent"`
}
