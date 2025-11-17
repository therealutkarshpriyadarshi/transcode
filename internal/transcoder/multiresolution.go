package transcoder

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"

	"github.com/therealutkarshpriyadarshi/transcode/pkg/models"
)

// MultiResolutionOptions holds options for multi-resolution transcoding
type MultiResolutionOptions struct {
	InputPath       string
	OutputDir       string
	Resolutions     []models.ResolutionProfile
	VideoCodec      string
	AudioCodec      string
	Preset          string
	EnableHLS       bool
	EnableDASH      bool
	HLSSegmentTime  int // Segment duration in seconds
	DASHSegmentTime int
	MaxConcurrent   int // Maximum concurrent transcoding jobs
}

// MultiResolutionResult holds the results of multi-resolution transcoding
type MultiResolutionResult struct {
	Outputs     []*ResolutionOutput
	HLSManifest *ManifestInfo
	DASHManifest *ManifestInfo
	Errors      []error
}

// ResolutionOutput represents a single resolution output
type ResolutionOutput struct {
	Resolution models.ResolutionProfile
	OutputPath string
	Size       int64
	Duration   float64
	Error      error
}

// ManifestInfo holds manifest file information
type ManifestInfo struct {
	ManifestPath string
	VariantPaths []string
	SegmentDir   string
}

// TranscodeMultiResolution transcodes a video to multiple resolutions in parallel
func (f *FFmpeg) TranscodeMultiResolution(ctx context.Context, opts MultiResolutionOptions, progressCB ProgressCallback) (*MultiResolutionResult, error) {
	result := &MultiResolutionResult{
		Outputs: make([]*ResolutionOutput, 0),
		Errors:  make([]error, 0),
	}

	if len(opts.Resolutions) == 0 {
		return nil, fmt.Errorf("no resolutions specified")
	}

	// Determine max concurrent jobs
	maxConcurrent := opts.MaxConcurrent
	if maxConcurrent <= 0 {
		maxConcurrent = 2 // Default to 2 concurrent jobs
	}

	// Create semaphore for concurrency control
	sem := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup
	var mu sync.Mutex // Protect shared result

	totalJobs := len(opts.Resolutions)
	completedJobs := 0
	jobProgress := make(map[int]float64)

	// Transcode each resolution
	for i, resolution := range opts.Resolutions {
		wg.Add(1)
		go func(idx int, res models.ResolutionProfile) {
			defer wg.Done()

			// Acquire semaphore
			sem <- struct{}{}
			defer func() { <-sem }()

			output := &ResolutionOutput{
				Resolution: res,
			}

			// Generate output filename
			outputFilename := fmt.Sprintf("%s_%s_%s.mp4",
				filepath.Base(opts.InputPath[:len(opts.InputPath)-len(filepath.Ext(opts.InputPath))]),
				res.Name,
				opts.VideoCodec,
			)
			output.OutputPath = filepath.Join(opts.OutputDir, outputFilename)

			// Prepare transcode options
			transcodeOpts := TranscodeOptions{
				InputPath:    opts.InputPath,
				OutputPath:   output.OutputPath,
				Width:        res.Width,
				Height:       res.Height,
				VideoBitrate: fmt.Sprintf("%d", res.VideoBitrate),
				AudioBitrate: fmt.Sprintf("%d", res.AudioBitrate),
				VideoCodec:   opts.VideoCodec,
				AudioCodec:   opts.AudioCodec,
				Preset:       opts.Preset,
			}

			// Progress callback for this resolution
			jobProgressCB := func(progress float64) {
				mu.Lock()
				defer mu.Unlock()

				jobProgress[idx] = progress

				// Calculate overall progress
				totalProgress := 0.0
				for _, p := range jobProgress {
					totalProgress += p
				}
				avgProgress := totalProgress / float64(totalJobs)

				if progressCB != nil {
					progressCB(avgProgress)
				}
			}

			// Transcode
			err := f.Transcode(ctx, transcodeOpts, jobProgressCB)
			if err != nil {
				output.Error = err
				mu.Lock()
				result.Errors = append(result.Errors, fmt.Errorf("failed to transcode %s: %w", res.Name, err))
				mu.Unlock()
			} else {
				// Get output file size and duration
				metadata, _ := f.ProbeVideo(ctx, output.OutputPath)
				if metadata != nil {
					// Parse size and duration
					// (Size and duration extraction would be similar to ExtractVideoInfo)
				}

				mu.Lock()
				completedJobs++
				mu.Unlock()
			}

			mu.Lock()
			result.Outputs = append(result.Outputs, output)
			mu.Unlock()
		}(i, resolution)
	}

	// Wait for all jobs to complete
	wg.Wait()

	return result, nil
}
