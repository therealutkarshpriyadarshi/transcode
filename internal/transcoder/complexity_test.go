package transcoder

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewComplexityAnalyzer(t *testing.T) {
	ffmpeg := NewFFmpeg("ffmpeg", "ffprobe")
	analyzer := NewComplexityAnalyzer(ffmpeg)

	assert.NotNil(t, analyzer)
	assert.NotNil(t, analyzer.ffmpeg)
}

func TestCalculateSamplePoints(t *testing.T) {
	ffmpeg := NewFFmpeg("ffmpeg", "ffprobe")
	analyzer := NewComplexityAnalyzer(ffmpeg)

	tests := []struct {
		name     string
		duration float64
		expected int
	}{
		{"short video", 30.0, 5},      // Minimum 5 samples
		{"1 minute", 60.0, 6},         // 1 sample per 10 seconds
		{"5 minutes", 300.0, 30},
		{"10 minutes", 600.0, 50},     // Maximum 50 samples
		{"very long", 2000.0, 50},     // Capped at maximum
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			points := analyzer.calculateSamplePoints(tt.duration)
			assert.Equal(t, tt.expected, points)
		})
	}
}

func TestCalculateComplexityScore(t *testing.T) {
	ffmpeg := NewFFmpeg("ffmpeg", "ffprobe")
	analyzer := NewComplexityAnalyzer(ffmpeg)

	tests := []struct {
		name          string
		si            *SITIMetrics
		motion        *MotionMetrics
		color         *ColorMetrics
		expectedRange [2]float64 // min, max
	}{
		{
			name: "high complexity",
			si: &SITIMetrics{
				AvgSI: 90.0,
				AvgTI: 40.0,
			},
			motion: &MotionMetrics{
				AvgIntensity: 0.8,
			},
			color: &ColorMetrics{
				ColorVariance: 0.8,
				EdgeDensity:   0.9,
			},
			expectedRange: [2]float64{0.7, 1.0},
		},
		{
			name: "low complexity",
			si: &SITIMetrics{
				AvgSI: 20.0,
				AvgTI: 10.0,
			},
			motion: &MotionMetrics{
				AvgIntensity: 0.2,
			},
			color: &ColorMetrics{
				ColorVariance: 0.3,
				EdgeDensity:   0.2,
			},
			expectedRange: [2]float64{0.0, 0.4},
		},
		{
			name: "medium complexity",
			si: &SITIMetrics{
				AvgSI: 50.0,
				AvgTI: 25.0,
			},
			motion: &MotionMetrics{
				AvgIntensity: 0.5,
			},
			color: &ColorMetrics{
				ColorVariance: 0.5,
				EdgeDensity:   0.5,
			},
			expectedRange: [2]float64{0.4, 0.7},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := analyzer.calculateComplexityScore(tt.si, tt.motion, tt.color)

			assert.GreaterOrEqual(t, score, 0.0, "score should be >= 0")
			assert.LessOrEqual(t, score, 1.0, "score should be <= 1")
			assert.GreaterOrEqual(t, score, tt.expectedRange[0],
				"score should be >= expected minimum")
			assert.LessOrEqual(t, score, tt.expectedRange[1],
				"score should be <= expected maximum")
		})
	}
}

func TestClassifyComplexity(t *testing.T) {
	ffmpeg := NewFFmpeg("ffmpeg", "ffprobe")
	analyzer := NewComplexityAnalyzer(ffmpeg)

	tests := []struct {
		score    float64
		expected string
	}{
		{0.9, "very_high"},
		{0.75, "very_high"},
		{0.7, "high"},
		{0.6, "high"},
		{0.5, "medium"},
		{0.4, "medium"},
		{0.3, "low"},
		{0.1, "low"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := analyzer.classifyComplexity(tt.score)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCategorizeContent(t *testing.T) {
	ffmpeg := NewFFmpeg("ffmpeg", "ffprobe")
	analyzer := NewComplexityAnalyzer(ffmpeg)

	tests := []struct {
		name         string
		si           *SITIMetrics
		motion       *MotionMetrics
		sceneChanges int
		duration     float64
		expected     string
	}{
		{
			name: "sports content",
			si: &SITIMetrics{
				AvgSI: 60.0,
				AvgTI: 30.0,
			},
			motion: &MotionMetrics{
				AvgIntensity: 0.8,
				Variance:     0.2,
			},
			sceneChanges: 100,
			duration:     300.0, // Scene rate: 0.33 > 0.2
			expected:     "sports",
		},
		{
			name: "presentation content",
			si: &SITIMetrics{
				AvgSI: 25.0,
				AvgTI: 10.0,
			},
			motion: &MotionMetrics{
				AvgIntensity: 0.3,
				Variance:     0.1,
			},
			sceneChanges: 20,
			duration:     300.0,
			expected:     "presentation",
		},
		{
			name: "gaming content",
			si: &SITIMetrics{
				AvgSI: 50.0,
				AvgTI: 25.0,
			},
			motion: &MotionMetrics{
				AvgIntensity: 0.6,
				Variance:     0.4, // High variance
			},
			sceneChanges: 50,
			duration:     300.0,
			expected:     "gaming",
		},
		{
			name: "movie content",
			si: &SITIMetrics{
				AvgSI: 50.0,
				AvgTI: 20.0,
			},
			motion: &MotionMetrics{
				AvgIntensity: 0.5,
				Variance:     0.2,
			},
			sceneChanges: 40,
			duration:     300.0,
			expected:     "movie",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzer.categorizeContent(tt.si, tt.motion, tt.sceneChanges, tt.duration)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDetectTextOverlay(t *testing.T) {
	ffmpeg := NewFFmpeg("ffmpeg", "ffprobe")
	analyzer := NewComplexityAnalyzer(ffmpeg)

	tests := []struct {
		name     string
		color    *ColorMetrics
		expected bool
	}{
		{
			name: "high edge density indicates text",
			color: &ColorMetrics{
				EdgeDensity: 0.8,
			},
			expected: true,
		},
		{
			name: "low edge density no text",
			color: &ColorMetrics{
				EdgeDensity: 0.5,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzer.detectTextOverlay(tt.color)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseSITIOutput(t *testing.T) {
	ffmpeg := NewFFmpeg("ffmpeg", "ffprobe")
	analyzer := NewComplexityAnalyzer(ffmpeg)

	tests := []struct {
		name   string
		output string
		check  func(*testing.T, *SITIMetrics)
	}{
		{
			name:   "valid output",
			output: "si_avg: 45.2\nti_avg: 12.3\nsi_max: 80.0\nti_max: 25.0\nsi_min: 20.0\nti_min: 5.0",
			check: func(t *testing.T, m *SITIMetrics) {
				assert.InDelta(t, 45.2, m.AvgSI, 0.1)
				assert.InDelta(t, 12.3, m.AvgTI, 0.1)
				assert.InDelta(t, 80.0, m.MaxSI, 0.1)
				assert.InDelta(t, 25.0, m.MaxTI, 0.1)
			},
		},
		{
			name:   "empty output uses defaults",
			output: "",
			check: func(t *testing.T, m *SITIMetrics) {
				assert.Greater(t, m.AvgSI, 0.0)
				assert.Greater(t, m.AvgTI, 0.0)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metrics := analyzer.parseSITIOutput(tt.output)
			require.NotNil(t, metrics)
			tt.check(t, metrics)
		})
	}
}

func TestAnalyzeComplexity_Integration(t *testing.T) {
	// This test requires a real video file
	t.Skip("Integration test - requires real video file and FFmpeg")

	ffmpeg := NewFFmpeg("ffmpeg", "ffprobe")
	analyzer := NewComplexityAnalyzer(ffmpeg)

	complexity, err := analyzer.AnalyzeComplexity(context.Background(), "/path/to/test/video.mp4")

	require.NoError(t, err)
	require.NotNil(t, complexity)

	// Check that basic fields are populated
	assert.NotEmpty(t, complexity.OverallComplexity)
	assert.GreaterOrEqual(t, complexity.ComplexityScore, 0.0)
	assert.LessOrEqual(t, complexity.ComplexityScore, 1.0)
	assert.Greater(t, complexity.SamplePoints, 0)
}

// Helper function tests
func TestAverage(t *testing.T) {
	tests := []struct {
		name     string
		values   []float64
		expected float64
	}{
		{"normal values", []float64{1.0, 2.0, 3.0, 4.0, 5.0}, 3.0},
		{"single value", []float64{5.0}, 5.0},
		{"empty slice", []float64{}, 0.0},
		{"negative values", []float64{-1.0, -2.0, -3.0}, -2.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := average(tt.values)
			assert.InDelta(t, tt.expected, result, 0.001)
		})
	}
}

func TestMaximum(t *testing.T) {
	tests := []struct {
		name     string
		values   []float64
		expected float64
	}{
		{"normal values", []float64{1.0, 5.0, 3.0, 2.0}, 5.0},
		{"single value", []float64{5.0}, 5.0},
		{"empty slice", []float64{}, 0.0},
		{"negative values", []float64{-5.0, -2.0, -10.0}, -2.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := maximum(tt.values)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMinimum(t *testing.T) {
	tests := []struct {
		name     string
		values   []float64
		expected float64
	}{
		{"normal values", []float64{5.0, 1.0, 3.0, 2.0}, 1.0},
		{"single value", []float64{5.0}, 5.0},
		{"empty slice", []float64{}, 0.0},
		{"negative values", []float64{-5.0, -2.0, -10.0}, -10.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := minimum(tt.values)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStandardDeviation(t *testing.T) {
	tests := []struct {
		name     string
		values   []float64
		expected float64
	}{
		{"normal values", []float64{2.0, 4.0, 4.0, 4.0, 5.0, 5.0, 7.0, 9.0}, 2.0},
		{"single value", []float64{5.0}, 0.0},
		{"empty slice", []float64{}, 0.0},
		{"all same", []float64{3.0, 3.0, 3.0, 3.0}, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := standardDeviation(tt.values)
			assert.InDelta(t, tt.expected, result, 0.1)
		})
	}
}
