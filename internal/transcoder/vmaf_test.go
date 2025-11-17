package transcoder

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewVMAFAnalyzer(t *testing.T) {
	ffmpeg := NewFFmpeg("ffmpeg", "ffprobe")
	analyzer := NewVMAFAnalyzer(ffmpeg)

	assert.NotNil(t, analyzer)
	assert.NotNil(t, analyzer.ffmpeg)
}

func TestVMAFAnalyzer_hasVMAFSupport(t *testing.T) {
	ffmpeg := NewFFmpeg("ffmpeg", "ffprobe")
	analyzer := NewVMAFAnalyzer(ffmpeg)

	ctx := context.Background()
	hasSupport := analyzer.hasVMAFSupport(ctx)

	// This test will pass even if VMAF is not available
	// It just checks that the method doesn't panic
	t.Logf("VMAF support available: %v", hasSupport)
}

func TestVMAFAnalyzer_AnalyzeVMAF(t *testing.T) {
	t.Skip("Skipping integration test - requires FFmpeg with VMAF support and test videos")

	// This is an integration test that would require:
	// 1. FFmpeg compiled with libvmaf
	// 2. Test video files
	// 3. Proper environment setup

	ffmpeg := NewFFmpeg("ffmpeg", "ffprobe")
	analyzer := NewVMAFAnalyzer(ffmpeg)

	ctx := context.Background()
	opts := VMAFOptions{
		ReferenceVideo: "/tmp/reference.mp4",
		DistortedVideo: "/tmp/distorted.mp4",
		OutputJSON:     "/tmp/vmaf_output.json",
		Model:          "version=vmaf_v0.6.1",
		Subsample:      1,
	}

	result, err := analyzer.AnalyzeVMAF(ctx, opts)

	if err != nil {
		t.Logf("VMAF analysis failed (expected in test environment): %v", err)
		return
	}

	assert.NotNil(t, result)
	assert.GreaterOrEqual(t, result.Score, 0.0)
	assert.LessOrEqual(t, result.Score, 100.0)
}

func TestVMAFAnalyzer_AnalyzeVMAFQuick(t *testing.T) {
	t.Skip("Skipping integration test - requires test videos")

	ffmpeg := NewFFmpeg("ffmpeg", "ffprobe")
	analyzer := NewVMAFAnalyzer(ffmpeg)

	ctx := context.Background()
	result, err := analyzer.AnalyzeVMAFQuick(ctx, "/tmp/reference.mp4", "/tmp/distorted.mp4")

	if err != nil {
		t.Logf("Quick VMAF analysis failed (expected in test environment): %v", err)
		return
	}

	assert.NotNil(t, result)
}

func TestVMAFAnalyzer_AnalyzeSegment(t *testing.T) {
	t.Skip("Skipping integration test - requires test videos")

	ffmpeg := NewFFmpeg("ffmpeg", "ffprobe")
	analyzer := NewVMAFAnalyzer(ffmpeg)

	ctx := context.Background()
	result, err := analyzer.AnalyzeSegment(
		ctx,
		"/tmp/reference.mp4",
		"/tmp/distorted.mp4",
		10.0,  // start at 10 seconds
		5.0,   // analyze 5 seconds
	)

	if err != nil {
		t.Logf("Segment VMAF analysis failed (expected in test environment): %v", err)
		return
	}

	assert.NotNil(t, result)
}

func TestVMAFAnalyzer_parseVMAFJSON(t *testing.T) {
	ffmpeg := NewFFmpeg("ffmpeg", "ffprobe")
	analyzer := NewVMAFAnalyzer(ffmpeg)

	// Create a test VMAF JSON file
	vmafJSON := `{
		"pooled_metrics": {
			"vmaf": {
				"mean": 94.5,
				"harmonic_mean": 92.3,
				"min": 88.2,
				"max": 98.7
			}
		}
	}`

	tmpFile, err := os.CreateTemp("", "vmaf_test_*.json")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(vmafJSON)
	require.NoError(t, err)
	tmpFile.Close()

	result, err := analyzer.parseVMAFJSON(tmpFile.Name())
	require.NoError(t, err)
	assert.NotNil(t, result)

	assert.Equal(t, 94.5, result.Score)
	assert.Equal(t, 94.5, result.Mean)
	assert.Equal(t, 92.3, result.HarmonicMean)
	assert.Equal(t, 88.2, result.Min)
	assert.Equal(t, 98.7, result.Max)
}

func TestVMAFAnalyzer_parseVMAFJSON_InvalidJSON(t *testing.T) {
	ffmpeg := NewFFmpeg("ffmpeg", "ffprobe")
	analyzer := NewVMAFAnalyzer(ffmpeg)

	tmpFile, err := os.CreateTemp("", "vmaf_invalid_*.json")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString("invalid json")
	require.NoError(t, err)
	tmpFile.Close()

	result, err := analyzer.parseVMAFJSON(tmpFile.Name())
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestVMAFAnalyzer_parseVMAFJSON_FileNotFound(t *testing.T) {
	ffmpeg := NewFFmpeg("ffmpeg", "ffprobe")
	analyzer := NewVMAFAnalyzer(ffmpeg)

	result, err := analyzer.parseVMAFJSON("/nonexistent/file.json")
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestVMAFAnalyzer_CalculateSSIM(t *testing.T) {
	t.Skip("Skipping integration test - requires test videos")

	ffmpeg := NewFFmpeg("ffmpeg", "ffprobe")
	analyzer := NewVMAFAnalyzer(ffmpeg)

	ctx := context.Background()
	ssim, err := analyzer.CalculateSSIM(ctx, "/tmp/reference.mp4", "/tmp/distorted.mp4")

	if err != nil {
		t.Logf("SSIM calculation failed (expected in test environment): %v", err)
		return
	}

	assert.GreaterOrEqual(t, ssim, 0.0)
	assert.LessOrEqual(t, ssim, 1.0)
}

func TestVMAFAnalyzer_CalculatePSNR(t *testing.T) {
	t.Skip("Skipping integration test - requires test videos")

	ffmpeg := NewFFmpeg("ffmpeg", "ffprobe")
	analyzer := NewVMAFAnalyzer(ffmpeg)

	ctx := context.Background()
	psnr, err := analyzer.CalculatePSNR(ctx, "/tmp/reference.mp4", "/tmp/distorted.mp4")

	if err != nil {
		t.Logf("PSNR calculation failed (expected in test environment): %v", err)
		return
	}

	assert.Greater(t, psnr, 0.0)
}

func TestVMAFAnalyzer_parseSSIMFromOutput(t *testing.T) {
	ffmpeg := NewFFmpeg("ffmpeg", "ffprobe")
	analyzer := NewVMAFAnalyzer(ffmpeg)

	tests := []struct {
		name     string
		output   string
		expected float64
	}{
		{
			name:     "Valid SSIM output",
			output:   "n:100 Y:0.95 U:0.96 V:0.97 All:0.95 (15.23)",
			expected: 0.95,
		},
		{
			name:     "No SSIM in output",
			output:   "Some random output",
			expected: 0.0,
		},
		{
			name:     "Empty output",
			output:   "",
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzer.parseSSIMFromOutput(tt.output)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestVMAFAnalyzer_parsePSNRFromOutput(t *testing.T) {
	ffmpeg := NewFFmpeg("ffmpeg", "ffprobe")
	analyzer := NewVMAFAnalyzer(ffmpeg)

	tests := []struct {
		name     string
		output   string
		expected float64
	}{
		{
			name:     "Valid PSNR output",
			output:   "PSNR y:45.23 u:46.12 v:47.01 average:45.79 min:42.10 max:48.50",
			expected: 45.79,
		},
		{
			name:     "No PSNR in output",
			output:   "Some random output",
			expected: 0.0,
		},
		{
			name:     "Empty output",
			output:   "",
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzer.parsePSNRFromOutput(tt.output)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestVMAFOptions_DefaultValues(t *testing.T) {
	ffmpeg := NewFFmpeg("ffmpeg", "ffprobe")
	analyzer := NewVMAFAnalyzer(ffmpeg)

	ctx := context.Background()
	opts := VMAFOptions{
		ReferenceVideo: "/tmp/ref.mp4",
		DistortedVideo: "/tmp/dist.mp4",
		OutputJSON:     "/tmp/out.json",
		// Model and Subsample not set - should use defaults
	}

	// This test just verifies the structure
	// The actual analysis would fail without real files
	_, err := analyzer.AnalyzeVMAF(ctx, opts)
	assert.Error(t, err) // Expected to fail without real files
}
