package transcoder

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/therealutkarshpriyadarshi/transcode/pkg/models"
)

// VMAFAnalyzer handles VMAF quality analysis
type VMAFAnalyzer struct {
	ffmpeg *FFmpeg
}

// NewVMAFAnalyzer creates a new VMAF analyzer
func NewVMAFAnalyzer(ffmpeg *FFmpeg) *VMAFAnalyzer {
	return &VMAFAnalyzer{
		ffmpeg: ffmpeg,
	}
}

// VMAFOptions holds options for VMAF analysis
type VMAFOptions struct {
	ReferenceVideo string  // Original video
	DistortedVideo string  // Encoded video to compare
	OutputJSON     string  // Output JSON file path
	Model          string  // VMAF model (default: vmaf_v0.6.1)
	Subsample      int     // Subsample factor (1 = every frame, 2 = every other frame, etc.)
}

// AnalyzeVMAF performs VMAF quality analysis comparing reference and distorted videos
func (v *VMAFAnalyzer) AnalyzeVMAF(ctx context.Context, opts VMAFOptions) (*models.VMAFResult, error) {
	// Check if FFmpeg has VMAF support
	if !v.hasVMAFSupport(ctx) {
		return nil, fmt.Errorf("FFmpeg does not have VMAF support compiled in")
	}

	// Set default model if not specified
	model := opts.Model
	if model == "" {
		model = "version=vmaf_v0.6.1"
	}

	// Set default subsample
	subsample := opts.Subsample
	if subsample == 0 {
		subsample = 1
	}

	// Build VMAF filter string
	vmafFilter := fmt.Sprintf(
		"[0:v]setpts=PTS-STARTPTS[reference];[1:v]setpts=PTS-STARTPTS,scale=w=iw:h=ih[distorted];[distorted][reference]libvmaf=log_fmt=json:log_path=%s:model=%s:n_subsample=%d",
		opts.OutputJSON,
		model,
		subsample,
	)

	// Build FFmpeg command
	args := []string{
		"-i", opts.ReferenceVideo,
		"-i", opts.DistortedVideo,
		"-filter_complex", vmafFilter,
		"-f", "null",
		"-",
	}

	cmd := exec.CommandContext(ctx, v.ffmpeg.ffmpegPath, args...)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("VMAF analysis failed: %w, stderr: %s", err, stderr.String())
	}

	// Parse VMAF JSON output
	result, err := v.parseVMAFJSON(opts.OutputJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to parse VMAF results: %w", err)
	}

	return result, nil
}

// AnalyzeVMAFQuick performs a quick VMAF analysis using subsampling
func (v *VMAFAnalyzer) AnalyzeVMAFQuick(ctx context.Context, reference, distorted string) (*models.VMAFResult, error) {
	tempJSON := "/tmp/vmaf_quick_" + generateRandomString(8) + ".json"
	defer removeFile(tempJSON)

	opts := VMAFOptions{
		ReferenceVideo: reference,
		DistortedVideo: distorted,
		OutputJSON:     tempJSON,
		Subsample:      4, // Analyze every 4th frame for speed
	}

	return v.AnalyzeVMAF(ctx, opts)
}

// AnalyzeSegment analyzes VMAF for a specific time segment
func (v *VMAFAnalyzer) AnalyzeSegment(ctx context.Context, reference, distorted string, startTime, duration float64) (*models.VMAFResult, error) {
	// Extract segments first
	refSegment := fmt.Sprintf("/tmp/ref_segment_%s.mp4", generateRandomString(8))
	distSegment := fmt.Sprintf("/tmp/dist_segment_%s.mp4", generateRandomString(8))
	defer removeFile(refSegment)
	defer removeFile(distSegment)

	// Extract reference segment
	if err := v.extractSegment(ctx, reference, refSegment, startTime, duration); err != nil {
		return nil, fmt.Errorf("failed to extract reference segment: %w", err)
	}

	// Extract distorted segment
	if err := v.extractSegment(ctx, distorted, distSegment, startTime, duration); err != nil {
		return nil, fmt.Errorf("failed to extract distorted segment: %w", err)
	}

	// Analyze segment
	return v.AnalyzeVMAFQuick(ctx, refSegment, distSegment)
}

// extractSegment extracts a video segment
func (v *VMAFAnalyzer) extractSegment(ctx context.Context, input, output string, startTime, duration float64) error {
	args := []string{
		"-ss", fmt.Sprintf("%.2f", startTime),
		"-t", fmt.Sprintf("%.2f", duration),
		"-i", input,
		"-c", "copy",
		"-y",
		output,
	}

	cmd := exec.CommandContext(ctx, v.ffmpeg.ffmpegPath, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("segment extraction failed: %w, stderr: %s", err, stderr.String())
	}

	return nil
}

// hasVMAFSupport checks if FFmpeg has VMAF support
func (v *VMAFAnalyzer) hasVMAFSupport(ctx context.Context) bool {
	cmd := exec.CommandContext(ctx, v.ffmpeg.ffmpegPath, "-filters")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}

	return strings.Contains(string(output), "libvmaf")
}

// parseVMAFJSON parses VMAF JSON output
func (v *VMAFAnalyzer) parseVMAFJSON(filepath string) (*models.VMAFResult, error) {
	// Read and parse JSON file
	data, err := readFile(filepath)
	if err != nil {
		return nil, err
	}

	var vmafOutput struct {
		PooledMetrics struct {
			VMAF struct {
				Mean         float64 `json:"mean"`
				HarmonicMean float64 `json:"harmonic_mean"`
				Min          float64 `json:"min"`
				Max          float64 `json:"max"`
			} `json:"vmaf"`
		} `json:"pooled_metrics"`
	}

	if err := json.Unmarshal(data, &vmafOutput); err != nil {
		return nil, fmt.Errorf("failed to parse VMAF JSON: %w", err)
	}

	result := &models.VMAFResult{
		Score:        vmafOutput.PooledMetrics.VMAF.Mean,
		Mean:         vmafOutput.PooledMetrics.VMAF.Mean,
		HarmonicMean: vmafOutput.PooledMetrics.VMAF.HarmonicMean,
		Min:          vmafOutput.PooledMetrics.VMAF.Min,
		Max:          vmafOutput.PooledMetrics.VMAF.Max,
	}

	return result, nil
}

// CalculateSSIM calculates SSIM (Structural Similarity Index)
func (v *VMAFAnalyzer) CalculateSSIM(ctx context.Context, reference, distorted string) (float64, error) {
	args := []string{
		"-i", reference,
		"-i", distorted,
		"-filter_complex", "[0:v][1:v]ssim=stats_file=-",
		"-f", "null",
		"-",
	}

	cmd := exec.CommandContext(ctx, v.ffmpeg.ffmpegPath, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return 0, fmt.Errorf("SSIM calculation failed: %w", err)
	}

	// Parse SSIM from stderr (FFmpeg outputs SSIM stats to stderr)
	output := stderr.String()
	ssim := v.parseSSIMFromOutput(output)

	return ssim, nil
}

// CalculatePSNR calculates PSNR (Peak Signal-to-Noise Ratio)
func (v *VMAFAnalyzer) CalculatePSNR(ctx context.Context, reference, distorted string) (float64, error) {
	args := []string{
		"-i", reference,
		"-i", distorted,
		"-filter_complex", "[0:v][1:v]psnr=stats_file=-",
		"-f", "null",
		"-",
	}

	cmd := exec.CommandContext(ctx, v.ffmpeg.ffmpegPath, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return 0, fmt.Errorf("PSNR calculation failed: %w", err)
	}

	// Parse PSNR from stderr
	output := stderr.String()
	psnr := v.parsePSNRFromOutput(output)

	return psnr, nil
}

// parseSSIMFromOutput extracts average SSIM from FFmpeg output
func (v *VMAFAnalyzer) parseSSIMFromOutput(output string) float64 {
	// Look for pattern like "All:0.95" or "average:0.95"
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "All:") {
			parts := strings.Split(line, "All:")
			if len(parts) > 1 {
				var ssim float64
				fmt.Sscanf(parts[1], "%f", &ssim)
				return ssim
			}
		}
	}
	return 0
}

// parsePSNRFromOutput extracts average PSNR from FFmpeg output
func (v *VMAFAnalyzer) parsePSNRFromOutput(output string) float64 {
	// Look for pattern like "average:45.23"
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "average:") {
			parts := strings.Split(line, "average:")
			if len(parts) > 1 {
				var psnr float64
				fmt.Sscanf(parts[1], "%f", &psnr)
				return psnr
			}
		}
	}
	return 0
}
