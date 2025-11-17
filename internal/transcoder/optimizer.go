package transcoder

import (
	"context"
	"fmt"
	"math"

	"github.com/therealutkarshpriyadarshi/transcode/pkg/models"
)

// EncodingOptimizer optimizes encoding settings based on content analysis
type EncodingOptimizer struct {
	ffmpeg              *FFmpeg
	vmafAnalyzer        *VMAFAnalyzer
	complexityAnalyzer  *ComplexityAnalyzer
}

// NewEncodingOptimizer creates a new encoding optimizer
func NewEncodingOptimizer(ffmpeg *FFmpeg) *EncodingOptimizer {
	return &EncodingOptimizer{
		ffmpeg:             ffmpeg,
		vmafAnalyzer:       NewVMAFAnalyzer(ffmpeg),
		complexityAnalyzer: NewComplexityAnalyzer(ffmpeg),
	}
}

// OptimizationOptions holds options for encoding optimization
type OptimizationOptions struct {
	VideoPath       string
	TargetVMAF      float64  // Target VMAF score (e.g., 95)
	MinVMAF         float64  // Minimum acceptable VMAF (e.g., 90)
	PreferQuality   bool     // Prefer quality over file size
	MaxResolution   string   // Maximum resolution to encode (e.g., "1080p")
}

// GenerateOptimizedLadder generates an optimized bitrate ladder for a video
func (o *EncodingOptimizer) GenerateOptimizedLadder(ctx context.Context, opts OptimizationOptions) (*models.EncodingProfile, error) {
	// Analyze content complexity
	complexity, err := o.complexityAnalyzer.AnalyzeComplexity(ctx, opts.VideoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze complexity: %w", err)
	}

	// Get video metadata
	metadata, err := o.ffmpeg.ExtractVideoInfo(ctx, opts.VideoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to extract video info: %w", err)
	}

	// Determine source resolution
	sourceHeight := metadata.Height

	// Get standard ladder as baseline
	standardLadder := o.getStandardLadder(sourceHeight, opts.MaxResolution)

	// Optimize ladder based on complexity
	optimizedLadder := o.optimizeLadderForComplexity(standardLadder, complexity, opts)

	// Determine codec recommendation
	codecRec := o.recommendCodec(complexity, opts.PreferQuality)

	// Determine preset recommendation
	presetRec := o.recommendPreset(complexity, opts.PreferQuality)

	// Estimate size reduction
	sizeReduction := o.estimateSizeReduction(standardLadder, optimizedLadder)

	// Create encoding profile
	profile := &models.EncodingProfile{
		VideoID:                     "", // Will be set by caller
		ProfileName:                 "optimized",
		IsActive:                    true,
		ContentType:                 complexity.ContentCategory,
		ComplexityLevel:             complexity.OverallComplexity,
		BitrateeLadder:              optimizedLadder,
		CodecRecommendation:         codecRec,
		PresetRecommendation:        presetRec,
		TargetVMAFScore:             &opts.TargetVMAF,
		MinVMAFScore:                &opts.MinVMAF,
		EstimatedSizeReduction:      &sizeReduction,
		ConfidenceScore:             o.calculateConfidence(complexity),
	}

	return profile, nil
}

// getStandardLadder returns the standard bitrate ladder based on source resolution
func (o *EncodingOptimizer) getStandardLadder(sourceHeight int, maxResolution string) []models.BitratePoint {
	// Define standard ladder
	allResolutions := []models.BitratePoint{
		{Resolution: "2160p", Bitrate: 25000000, TargetVMAF: 95},
		{Resolution: "1440p", Bitrate: 16000000, TargetVMAF: 95},
		{Resolution: "1080p", Bitrate: 8000000, TargetVMAF: 95},
		{Resolution: "720p", Bitrate: 4000000, TargetVMAF: 93},
		{Resolution: "480p", Bitrate: 2000000, TargetVMAF: 90},
		{Resolution: "360p", Bitrate: 1000000, TargetVMAF: 85},
		{Resolution: "240p", Bitrate: 400000, TargetVMAF: 80},
	}

	// Filter by source resolution (don't upscale)
	ladder := make([]models.BitratePoint, 0)
	maxHeight := o.resolutionToHeight(maxResolution)
	if maxHeight == 0 {
		maxHeight = 99999
	}

	for _, point := range allResolutions {
		height := o.resolutionToHeight(point.Resolution)
		if height <= sourceHeight && height <= maxHeight {
			ladder = append(ladder, point)
		}
	}

	// Ensure at least one resolution
	if len(ladder) == 0 && len(allResolutions) > 0 {
		ladder = append(ladder, allResolutions[len(allResolutions)-1])
	}

	return ladder
}

// optimizeLadderForComplexity optimizes the bitrate ladder based on content complexity
func (o *EncodingOptimizer) optimizeLadderForComplexity(
	standardLadder []models.BitratePoint,
	complexity *models.ContentComplexity,
	opts OptimizationOptions,
) []models.BitratePoint {
	optimized := make([]models.BitratePoint, len(standardLadder))

	// Calculate bitrate multiplier based on complexity
	multiplier := o.calculateBitrateMultiplier(complexity, opts.PreferQuality)

	for i, point := range standardLadder {
		newBitrate := int64(float64(point.Bitrate) * multiplier)

		// Apply bounds
		minBitrate := int64(float64(point.Bitrate) * 0.5) // At least 50% of standard
		maxBitrate := int64(float64(point.Bitrate) * 1.8) // At most 180% of standard

		if newBitrate < minBitrate {
			newBitrate = minBitrate
		}
		if newBitrate > maxBitrate {
			newBitrate = maxBitrate
		}

		optimized[i] = models.BitratePoint{
			Resolution: point.Resolution,
			Bitrate:    newBitrate,
			TargetVMAF: opts.TargetVMAF,
		}
	}

	return optimized
}

// calculateBitrateMultiplier calculates the bitrate adjustment multiplier
func (o *EncodingOptimizer) calculateBitrateMultiplier(complexity *models.ContentComplexity, preferQuality bool) float64 {
	baseMultiplier := 1.0

	// Adjust based on complexity level
	switch complexity.OverallComplexity {
	case "very_high":
		baseMultiplier = 1.4
	case "high":
		baseMultiplier = 1.2
	case "medium":
		baseMultiplier = 1.0
	case "low":
		baseMultiplier = 0.7
	}

	// Adjust based on motion
	if complexity.AvgMotionIntensity > 0.7 {
		baseMultiplier *= 1.15
	} else if complexity.AvgMotionIntensity < 0.3 {
		baseMultiplier *= 0.85
	}

	// Adjust based on spatial complexity
	if complexity.AvgSpatialInfo > 70 {
		baseMultiplier *= 1.1
	} else if complexity.AvgSpatialInfo < 30 {
		baseMultiplier *= 0.9
	}

	// Adjust for quality preference
	if preferQuality {
		baseMultiplier *= 1.1
	}

	// Adjust for content type
	switch complexity.ContentCategory {
	case "sports", "gaming":
		baseMultiplier *= 1.15 // Higher bitrate for fast motion
	case "presentation":
		baseMultiplier *= 0.75 // Lower bitrate for static content
	case "animation":
		baseMultiplier *= 0.85 // Animations compress well
	}

	return baseMultiplier
}

// recommendCodec recommends the best codec for the content
func (o *EncodingOptimizer) recommendCodec(complexity *models.ContentComplexity, preferQuality bool) string {
	// For high complexity or quality preference, use H.265
	if complexity.ComplexityScore > 0.7 || preferQuality {
		return "libx265"
	}

	// For animation or low complexity, H.264 is sufficient and faster
	if complexity.ContentCategory == "animation" || complexity.ComplexityScore < 0.4 {
		return "libx264"
	}

	// Default to H.264 for compatibility
	return "libx264"
}

// recommendPreset recommends the encoding preset
func (o *EncodingOptimizer) recommendPreset(complexity *models.ContentComplexity, preferQuality bool) string {
	if preferQuality {
		return "slow" // Better compression, longer encoding time
	}

	// For high complexity, use faster preset to save encoding time
	if complexity.ComplexityScore > 0.7 {
		return "medium"
	}

	// For low complexity, can use slower preset for better compression
	if complexity.ComplexityScore < 0.4 {
		return "slow"
	}

	return "medium"
}

// estimateSizeReduction estimates the percentage size reduction
func (o *EncodingOptimizer) estimateSizeReduction(standard, optimized []models.BitratePoint) float64 {
	if len(standard) == 0 || len(optimized) == 0 {
		return 0
	}

	standardTotal := int64(0)
	optimizedTotal := int64(0)

	for _, p := range standard {
		standardTotal += p.Bitrate
	}
	for _, p := range optimized {
		optimizedTotal += p.Bitrate
	}

	if standardTotal == 0 {
		return 0
	}

	reduction := float64(standardTotal-optimizedTotal) / float64(standardTotal) * 100
	return reduction
}

// calculateConfidence calculates confidence score for the optimization
func (o *EncodingOptimizer) calculateConfidence(complexity *models.ContentComplexity) *float64 {
	confidence := 0.8 // Base confidence

	// Higher confidence for more sample points
	if complexity.SamplePoints >= 30 {
		confidence += 0.1
	}

	// Lower confidence for very high or very low complexity (edge cases)
	if complexity.ComplexityScore > 0.85 || complexity.ComplexityScore < 0.15 {
		confidence -= 0.1
	}

	// Ensure bounds
	confidence = math.Min(math.Max(confidence, 0.5), 1.0)

	return &confidence
}

// resolutionToHeight converts resolution string to height in pixels
func (o *EncodingOptimizer) resolutionToHeight(resolution string) int {
	heights := map[string]int{
		"2160p": 2160,
		"1440p": 1440,
		"1080p": 1080,
		"720p":  720,
		"480p":  480,
		"360p":  360,
		"240p":  240,
		"144p":  144,
		"4k":    2160,
		"2k":    1440,
		"qhd":   1440,
		"fhd":   1080,
		"hd":    720,
		"sd":    480,
	}

	if height, ok := heights[resolution]; ok {
		return height
	}
	return 0
}

// TestBitratePoint tests a specific bitrate for a resolution and returns VMAF score
func (o *EncodingOptimizer) TestBitratePoint(
	ctx context.Context,
	videoPath string,
	resolution string,
	bitrate int64,
	codec string,
) (*models.QualityAnalysis, error) {
	// Create temporary output path
	tempOutput := fmt.Sprintf("/tmp/test_%s_%d_%s.mp4", resolution, bitrate, generateRandomString(8))
	defer removeFile(tempOutput)

	// Get resolution dimensions
	width, height := parseResolution(resolution)

	// Transcode with test settings
	opts := TranscodeOptions{
		InputPath:    videoPath,
		OutputPath:   tempOutput,
		Width:        width,
		Height:       height,
		VideoBitrate: fmt.Sprintf("%dk", bitrate/1000),
		VideoCodec:   codec,
		Preset:       "medium",
		Format:       "mp4",
		ExtraArgs:    []string{},
	}

	if err := o.ffmpeg.Transcode(ctx, opts, nil); err != nil {
		return nil, fmt.Errorf("test transcode failed: %w", err)
	}

	// Analyze VMAF
	vmafResult, err := o.vmafAnalyzer.AnalyzeVMAFQuick(ctx, videoPath, tempOutput)
	if err != nil {
		return nil, fmt.Errorf("VMAF analysis failed: %w", err)
	}

	// Create quality analysis result
	analysis := &models.QualityAnalysis{
		AnalysisType:   "vmaf",
		TestBitrate:    &bitrate,
		TestResolution: resolution,
		TestCodec:      codec,
		VMAFScore:      &vmafResult.Score,
		VMAFMin:        &vmafResult.Min,
		VMAFMax:        &vmafResult.Max,
		VMAFMean:       &vmafResult.Mean,
		VMAFHarmonicMean: &vmafResult.HarmonicMean,
	}

	return analysis, nil
}

// FindOptimalBitrate finds the optimal bitrate for a target VMAF score using binary search
func (o *EncodingOptimizer) FindOptimalBitrate(
	ctx context.Context,
	videoPath string,
	resolution string,
	targetVMAF float64,
	minBitrate, maxBitrate int64,
	codec string,
) (int64, error) {
	const maxIterations = 5
	const tolerance = 2.0 // VMAF tolerance

	for i := 0; i < maxIterations; i++ {
		// Test middle bitrate
		testBitrate := (minBitrate + maxBitrate) / 2

		analysis, err := o.TestBitratePoint(ctx, videoPath, resolution, testBitrate, codec)
		if err != nil {
			return 0, err
		}

		vmafScore := *analysis.VMAFScore

		// Check if we're within tolerance
		if math.Abs(vmafScore-targetVMAF) <= tolerance {
			return testBitrate, nil
		}

		// Adjust search range
		if vmafScore < targetVMAF {
			// Need higher bitrate
			minBitrate = testBitrate
		} else {
			// Can use lower bitrate
			maxBitrate = testBitrate
		}

		// Prevent infinite loop
		if maxBitrate-minBitrate < 100000 { // 100 kbps
			break
		}
	}

	// Return the higher end to ensure quality
	return maxBitrate, nil
}
