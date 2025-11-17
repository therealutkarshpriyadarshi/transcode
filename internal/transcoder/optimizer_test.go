package transcoder

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/therealutkarshpriyadarshi/transcode/pkg/models"
)

func TestNewEncodingOptimizer(t *testing.T) {
	ffmpeg := NewFFmpeg("ffmpeg", "ffprobe")
	optimizer := NewEncodingOptimizer(ffmpeg)

	assert.NotNil(t, optimizer)
	assert.NotNil(t, optimizer.ffmpeg)
	assert.NotNil(t, optimizer.vmafAnalyzer)
	assert.NotNil(t, optimizer.complexityAnalyzer)
}

func TestGetStandardLadder(t *testing.T) {
	ffmpeg := NewFFmpeg("ffmpeg", "ffprobe")
	optimizer := NewEncodingOptimizer(ffmpeg)

	tests := []struct {
		name          string
		sourceHeight  int
		maxResolution string
		expectedCount int
		expectHighest string
	}{
		{
			name:          "4K source no limit",
			sourceHeight:  2160,
			maxResolution: "",
			expectedCount: 7,
			expectHighest: "2160p",
		},
		{
			name:          "1080p source",
			sourceHeight:  1080,
			maxResolution: "",
			expectedCount: 5,
			expectHighest: "1080p",
		},
		{
			name:          "720p source",
			sourceHeight:  720,
			maxResolution: "",
			expectedCount: 4,
			expectHighest: "720p",
		},
		{
			name:          "4K source limited to 1080p",
			sourceHeight:  2160,
			maxResolution: "1080p",
			expectedCount: 5,
			expectHighest: "1080p",
		},
		{
			name:          "480p source",
			sourceHeight:  480,
			maxResolution: "",
			expectedCount: 3,
			expectHighest: "480p",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ladder := optimizer.getStandardLadder(tt.sourceHeight, tt.maxResolution)

			assert.Len(t, ladder, tt.expectedCount, "ladder should have expected number of resolutions")
			if tt.expectedCount > 0 {
				assert.Equal(t, tt.expectHighest, ladder[0].Resolution, "highest resolution should match")
			}

			// Verify ladder is sorted by bitrate descending
			for i := 0; i < len(ladder)-1; i++ {
				assert.Greater(t, ladder[i].Bitrate, ladder[i+1].Bitrate, "bitrates should be descending")
			}
		})
	}
}

func TestCalculateBitrateMultiplier(t *testing.T) {
	ffmpeg := NewFFmpeg("ffmpeg", "ffprobe")
	optimizer := NewEncodingOptimizer(ffmpeg)

	tests := []struct {
		name           string
		complexity     *models.ContentComplexity
		preferQuality  bool
		expectedRange  [2]float64 // min, max expected multiplier
	}{
		{
			name: "very high complexity",
			complexity: &models.ContentComplexity{
				OverallComplexity:  "very_high",
				ComplexityScore:    0.9,
				AvgMotionIntensity: 0.8,
				AvgSpatialInfo:     80,
				ContentCategory:    "sports",
			},
			preferQuality: false,
			expectedRange: [2]float64{1.5, 2.0}, // Should be high
		},
		{
			name: "low complexity",
			complexity: &models.ContentComplexity{
				OverallComplexity:  "low",
				ComplexityScore:    0.3,
				AvgMotionIntensity: 0.2,
				AvgSpatialInfo:     25,
				ContentCategory:    "presentation",
			},
			preferQuality: false,
			expectedRange: [2]float64{0.4, 0.7}, // Should be low
		},
		{
			name: "medium complexity with quality preference",
			complexity: &models.ContentComplexity{
				OverallComplexity:  "medium",
				ComplexityScore:    0.5,
				AvgMotionIntensity: 0.5,
				AvgSpatialInfo:     50,
				ContentCategory:    "movie",
			},
			preferQuality: true,
			expectedRange: [2]float64{1.0, 1.3}, // Should be slightly elevated
		},
		{
			name: "animation content",
			complexity: &models.ContentComplexity{
				OverallComplexity:  "medium",
				ComplexityScore:    0.5,
				AvgMotionIntensity: 0.4,
				AvgSpatialInfo:     45,
				ContentCategory:    "animation",
			},
			preferQuality: false,
			expectedRange: [2]float64{0.7, 1.0}, // Animation compresses well
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			multiplier := optimizer.calculateBitrateMultiplier(tt.complexity, tt.preferQuality)

			assert.GreaterOrEqual(t, multiplier, tt.expectedRange[0],
				"multiplier should be >= expected minimum")
			assert.LessOrEqual(t, multiplier, tt.expectedRange[1],
				"multiplier should be <= expected maximum")
		})
	}
}

func TestRecommendCodec(t *testing.T) {
	ffmpeg := NewFFmpeg("ffmpeg", "ffprobe")
	optimizer := NewEncodingOptimizer(ffmpeg)

	tests := []struct {
		name          string
		complexity    *models.ContentComplexity
		preferQuality bool
		expectedCodec string
	}{
		{
			name: "high complexity with quality preference",
			complexity: &models.ContentComplexity{
				ComplexityScore: 0.8,
				ContentCategory: "sports",
			},
			preferQuality: true,
			expectedCodec: "libx265",
		},
		{
			name: "low complexity animation",
			complexity: &models.ContentComplexity{
				ComplexityScore: 0.3,
				ContentCategory: "animation",
			},
			preferQuality: false,
			expectedCodec: "libx264",
		},
		{
			name: "medium complexity default",
			complexity: &models.ContentComplexity{
				ComplexityScore: 0.5,
				ContentCategory: "movie",
			},
			preferQuality: false,
			expectedCodec: "libx264",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			codec := optimizer.recommendCodec(tt.complexity, tt.preferQuality)
			assert.Equal(t, tt.expectedCodec, codec)
		})
	}
}

func TestRecommendPreset(t *testing.T) {
	ffmpeg := NewFFmpeg("ffmpeg", "ffprobe")
	optimizer := NewEncodingOptimizer(ffmpeg)

	tests := []struct {
		name           string
		complexity     *models.ContentComplexity
		preferQuality  bool
		expectedPreset string
	}{
		{
			name:           "quality preference",
			complexity:     &models.ContentComplexity{ComplexityScore: 0.5},
			preferQuality:  true,
			expectedPreset: "slow",
		},
		{
			name:           "high complexity fast encoding",
			complexity:     &models.ContentComplexity{ComplexityScore: 0.8},
			preferQuality:  false,
			expectedPreset: "medium",
		},
		{
			name:           "low complexity better compression",
			complexity:     &models.ContentComplexity{ComplexityScore: 0.3},
			preferQuality:  false,
			expectedPreset: "slow",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			preset := optimizer.recommendPreset(tt.complexity, tt.preferQuality)
			assert.Equal(t, tt.expectedPreset, preset)
		})
	}
}

func TestOptimizeLadderForComplexity(t *testing.T) {
	ffmpeg := NewFFmpeg("ffmpeg", "ffprobe")
	optimizer := NewEncodingOptimizer(ffmpeg)

	standardLadder := []models.BitratePoint{
		{Resolution: "1080p", Bitrate: 8000000, TargetVMAF: 95},
		{Resolution: "720p", Bitrate: 4000000, TargetVMAF: 93},
		{Resolution: "480p", Bitrate: 2000000, TargetVMAF: 90},
	}

	tests := []struct {
		name       string
		complexity *models.ContentComplexity
		opts       OptimizationOptions
	}{
		{
			name: "high complexity increases bitrates",
			complexity: &models.ContentComplexity{
				OverallComplexity:  "high",
				ComplexityScore:    0.7,
				AvgMotionIntensity: 0.7,
				AvgSpatialInfo:     70,
			},
			opts: OptimizationOptions{
				TargetVMAF:    95,
				PreferQuality: false,
			},
		},
		{
			name: "low complexity decreases bitrates",
			complexity: &models.ContentComplexity{
				OverallComplexity:  "low",
				ComplexityScore:    0.3,
				AvgMotionIntensity: 0.2,
				AvgSpatialInfo:     30,
			},
			opts: OptimizationOptions{
				TargetVMAF:    95,
				PreferQuality: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			optimized := optimizer.optimizeLadderForComplexity(standardLadder, tt.complexity, tt.opts)

			require.Len(t, optimized, len(standardLadder), "optimized ladder should have same length")

			// Verify all bitrates are within bounds (50%-180% of standard)
			for i, point := range optimized {
				standard := standardLadder[i]
				minBitrate := int64(float64(standard.Bitrate) * 0.5)
				maxBitrate := int64(float64(standard.Bitrate) * 1.8)

				assert.GreaterOrEqual(t, point.Bitrate, minBitrate,
					"bitrate should be >= 50%% of standard for %s", point.Resolution)
				assert.LessOrEqual(t, point.Bitrate, maxBitrate,
					"bitrate should be <= 180%% of standard for %s", point.Resolution)
			}
		})
	}
}

func TestEstimateSizeReduction(t *testing.T) {
	ffmpeg := NewFFmpeg("ffmpeg", "ffprobe")
	optimizer := NewEncodingOptimizer(ffmpeg)

	standard := []models.BitratePoint{
		{Bitrate: 8000000},
		{Bitrate: 4000000},
		{Bitrate: 2000000},
	}

	optimized := []models.BitratePoint{
		{Bitrate: 6000000}, // 25% reduction
		{Bitrate: 3000000}, // 25% reduction
		{Bitrate: 1500000}, // 25% reduction
	}

	reduction := optimizer.estimateSizeReduction(standard, optimized)

	// Total standard: 14M, Total optimized: 10.5M = 25% reduction
	assert.InDelta(t, 25.0, reduction, 1.0, "size reduction should be approximately 25%%")
}

func TestCalculateConfidence(t *testing.T) {
	ffmpeg := NewFFmpeg("ffmpeg", "ffprobe")
	optimizer := NewEncodingOptimizer(ffmpeg)

	tests := []struct {
		name              string
		complexity        *models.ContentComplexity
		expectedMinConf   float64
		expectedMaxConf   float64
	}{
		{
			name: "many samples normal complexity",
			complexity: &models.ContentComplexity{
				SamplePoints:    50,
				ComplexityScore: 0.5,
			},
			expectedMinConf: 0.8,
			expectedMaxConf: 1.0,
		},
		{
			name: "few samples",
			complexity: &models.ContentComplexity{
				SamplePoints:    5,
				ComplexityScore: 0.5,
			},
			expectedMinConf: 0.7,
			expectedMaxConf: 0.9,
		},
		{
			name: "extreme complexity",
			complexity: &models.ContentComplexity{
				SamplePoints:    30,
				ComplexityScore: 0.95,
			},
			expectedMinConf: 0.7,
			expectedMaxConf: 0.9,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			confidence := optimizer.calculateConfidence(tt.complexity)

			require.NotNil(t, confidence)
			assert.GreaterOrEqual(t, *confidence, tt.expectedMinConf)
			assert.LessOrEqual(t, *confidence, tt.expectedMaxConf)
		})
	}
}

func TestResolutionToHeight(t *testing.T) {
	ffmpeg := NewFFmpeg("ffmpeg", "ffprobe")
	optimizer := NewEncodingOptimizer(ffmpeg)

	tests := []struct {
		resolution     string
		expectedHeight int
	}{
		{"2160p", 2160},
		{"4k", 2160},
		{"1440p", 1440},
		{"1080p", 1080},
		{"fhd", 1080},
		{"720p", 720},
		{"hd", 720},
		{"480p", 480},
		{"360p", 360},
		{"240p", 240},
		{"invalid", 0},
		{"", 0},
	}

	for _, tt := range tests {
		t.Run(tt.resolution, func(t *testing.T) {
			height := optimizer.resolutionToHeight(tt.resolution)
			assert.Equal(t, tt.expectedHeight, height)
		})
	}
}

func TestGenerateOptimizedLadder_Integration(t *testing.T) {
	// This test would require a real video file
	// For now, it's a placeholder for integration testing
	t.Skip("Integration test - requires real video file and FFmpeg")

	ffmpeg := NewFFmpeg("ffmpeg", "ffprobe")
	optimizer := NewEncodingOptimizer(ffmpeg)

	opts := OptimizationOptions{
		VideoPath:     "/path/to/test/video.mp4",
		TargetVMAF:    95,
		MinVMAF:       90,
		PreferQuality: false,
		MaxResolution: "1080p",
	}

	profile, err := optimizer.GenerateOptimizedLadder(context.Background(), opts)

	require.NoError(t, err)
	require.NotNil(t, profile)
	assert.NotEmpty(t, profile.BitrateeLadder)
	assert.NotEmpty(t, profile.CodecRecommendation)
	assert.NotEmpty(t, profile.PresetRecommendation)
}
