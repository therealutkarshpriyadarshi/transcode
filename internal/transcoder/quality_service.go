package transcoder

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/therealutkarshpriyadarshi/transcode/internal/config"
	"github.com/therealutkarshpriyadarshi/transcode/internal/database"
	"github.com/therealutkarshpriyadarshi/transcode/internal/storage"
	"github.com/therealutkarshpriyadarshi/transcode/pkg/models"
)

// QualityService handles quality analysis and per-title encoding
type QualityService struct {
	ffmpeg     *FFmpeg
	storage    *storage.Storage
	repo       *database.Repository
	optimizer  *EncodingOptimizer
	vmaf       *VMAFAnalyzer
	complexity *ComplexityAnalyzer
	cfg        config.TranscoderConfig
}

// NewQualityService creates a new quality service
func NewQualityService(
	cfg config.TranscoderConfig,
	storage *storage.Storage,
	repo *database.Repository,
) *QualityService {
	ffmpeg := NewFFmpeg(cfg.FFmpegPath, cfg.FFprobePath)

	return &QualityService{
		ffmpeg:     ffmpeg,
		storage:    storage,
		repo:       repo,
		optimizer:  NewEncodingOptimizer(ffmpeg),
		vmaf:       NewVMAFAnalyzer(ffmpeg),
		complexity: NewComplexityAnalyzer(ffmpeg),
		cfg:        cfg,
	}
}

// AnalyzeVideoQuality performs comprehensive quality analysis on a video
func (s *QualityService) AnalyzeVideoQuality(ctx context.Context, videoID string) (*models.ContentComplexity, error) {
	// Get video
	video, err := s.repo.GetVideo(ctx, videoID)
	if err != nil {
		return nil, fmt.Errorf("failed to get video: %w", err)
	}

	// Download video temporarily
	tempPath := fmt.Sprintf("%s/%s_analysis", s.cfg.TempDir, videoID)
	if err := s.storage.DownloadFile(ctx, video.OriginalURL, tempPath); err != nil {
		return nil, fmt.Errorf("failed to download video: %w", err)
	}
	defer removeFile(tempPath)

	// Analyze complexity
	complexity, err := s.complexity.AnalyzeComplexity(ctx, tempPath)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze complexity: %w", err)
	}

	// Set video ID and timestamp
	complexity.ID = uuid.New().String()
	complexity.VideoID = videoID
	complexity.AnalyzedAt = time.Now()

	// Store in database
	if err := s.repo.CreateContentComplexity(ctx, complexity); err != nil {
		return nil, fmt.Errorf("failed to store complexity: %w", err)
	}

	return complexity, nil
}

// GenerateEncodingProfile generates an optimized encoding profile for a video
func (s *QualityService) GenerateEncodingProfile(
	ctx context.Context,
	videoID string,
	presetName string,
) (*models.EncodingProfile, error) {
	// Get video
	video, err := s.repo.GetVideo(ctx, videoID)
	if err != nil {
		return nil, fmt.Errorf("failed to get video: %w", err)
	}

	// Get or create complexity analysis
	complexity, err := s.repo.GetContentComplexity(ctx, videoID)
	if err != nil || complexity == nil {
		// Analyze if not exists
		complexity, err = s.AnalyzeVideoQuality(ctx, videoID)
		if err != nil {
			return nil, fmt.Errorf("failed to analyze video: %w", err)
		}
	}

	// Get quality preset
	preset, err := s.repo.GetQualityPresetByName(ctx, presetName)
	if err != nil {
		return nil, fmt.Errorf("failed to get quality preset: %w", err)
	}

	// Download video temporarily
	tempPath := fmt.Sprintf("%s/%s_profile", s.cfg.TempDir, videoID)
	if err := s.storage.DownloadFile(ctx, video.OriginalURL, tempPath); err != nil {
		return nil, fmt.Errorf("failed to download video: %w", err)
	}
	defer removeFile(tempPath)

	// Generate optimized ladder
	opts := OptimizationOptions{
		VideoPath:     tempPath,
		TargetVMAF:    preset.TargetVMAF,
		MinVMAF:       preset.MinVMAF,
		PreferQuality: preset.PreferQuality,
		MaxResolution: "1080p", // Can be configured
	}

	profile, err := s.optimizer.GenerateOptimizedLadder(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to generate optimized ladder: %w", err)
	}

	// Set profile details
	profile.ID = uuid.New().String()
	profile.VideoID = videoID
	profile.ProfileName = presetName
	profile.CreatedAt = time.Now()
	profile.UpdatedAt = time.Now()

	// Store in database
	if err := s.repo.CreateEncodingProfile(ctx, profile); err != nil {
		return nil, fmt.Errorf("failed to store encoding profile: %w", err)
	}

	return profile, nil
}

// CompareEncodings compares two encodings using VMAF
func (s *QualityService) CompareEncodings(
	ctx context.Context,
	videoID string,
	outputID1, outputID2 string,
) (*ComparisonResult, error) {
	// Get original video
	video, err := s.repo.GetVideo(ctx, videoID)
	if err != nil {
		return nil, fmt.Errorf("failed to get video: %w", err)
	}

	// Get outputs
	output1, err := s.repo.GetOutput(ctx, outputID1)
	if err != nil {
		return nil, fmt.Errorf("failed to get output1: %w", err)
	}

	output2, err := s.repo.GetOutput(ctx, outputID2)
	if err != nil {
		return nil, fmt.Errorf("failed to get output2: %w", err)
	}

	// Download files
	refPath := fmt.Sprintf("%s/ref_%s", s.cfg.TempDir, videoID)
	out1Path := fmt.Sprintf("%s/out1_%s", s.cfg.TempDir, outputID1)
	out2Path := fmt.Sprintf("%s/out2_%s", s.cfg.TempDir, outputID2)

	defer removeFile(refPath)
	defer removeFile(out1Path)
	defer removeFile(out2Path)

	if err := s.storage.DownloadFile(ctx, video.OriginalURL, refPath); err != nil {
		return nil, fmt.Errorf("failed to download reference: %w", err)
	}
	if err := s.storage.DownloadFile(ctx, output1.URL, out1Path); err != nil {
		return nil, fmt.Errorf("failed to download output1: %w", err)
	}
	if err := s.storage.DownloadFile(ctx, output2.URL, out2Path); err != nil {
		return nil, fmt.Errorf("failed to download output2: %w", err)
	}

	// Analyze both
	vmaf1, err := s.vmaf.AnalyzeVMAFQuick(ctx, refPath, out1Path)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze output1: %w", err)
	}

	vmaf2, err := s.vmaf.AnalyzeVMAFQuick(ctx, refPath, out2Path)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze output2: %w", err)
	}

	// Calculate efficiency (bits per VMAF point)
	efficiency1 := float64(output1.Bitrate) / vmaf1.Score
	efficiency2 := float64(output2.Bitrate) / vmaf2.Score

	result := &ComparisonResult{
		Output1: EncodingMetrics{
			OutputID:   outputID1,
			VMAF:       vmaf1.Score,
			Size:       output1.Size,
			Bitrate:    output1.Bitrate,
			Efficiency: efficiency1,
		},
		Output2: EncodingMetrics{
			OutputID:   outputID2,
			VMAF:       vmaf2.Score,
			Size:       output2.Size,
			Bitrate:    output2.Bitrate,
			Efficiency: efficiency2,
		},
		Winner: s.determineWinner(efficiency1, efficiency2, vmaf1.Score, vmaf2.Score),
	}

	return result, nil
}

// ComparisonResult holds comparison results
type ComparisonResult struct {
	Output1 EncodingMetrics `json:"output1"`
	Output2 EncodingMetrics `json:"output2"`
	Winner  string          `json:"winner"` // "output1", "output2", or "tie"
}

// EncodingMetrics holds encoding performance metrics
type EncodingMetrics struct {
	OutputID   string  `json:"output_id"`
	VMAF       float64 `json:"vmaf"`
	Size       int64   `json:"size"`
	Bitrate    int64   `json:"bitrate"`
	Efficiency float64 `json:"efficiency"` // bits per VMAF point (lower is better)
}

// determineWinner determines which encoding is better
func (s *QualityService) determineWinner(eff1, eff2, vmaf1, vmaf2 float64) string {
	// If VMAF scores are similar (within 2 points), choose better efficiency
	if abs(vmaf1-vmaf2) < 2.0 {
		if eff1 < eff2 {
			return "output1"
		} else if eff2 < eff1 {
			return "output2"
		}
		return "tie"
	}

	// Otherwise, choose better VMAF
	if vmaf1 > vmaf2 {
		return "output1"
	} else if vmaf2 > vmaf1 {
		return "output2"
	}
	return "tie"
}

// RunBitrateExperiment runs an A/B test for different bitrate ladders
func (s *QualityService) RunBitrateExperiment(
	ctx context.Context,
	videoID string,
	experimentName string,
	ladderConfig []models.BitratePoint,
) (*models.BitrateExperiment, error) {
	experiment := &models.BitrateExperiment{
		ID:             uuid.New().String(),
		VideoID:        videoID,
		ExperimentName: experimentName,
		LadderConfig:   ladderConfig,
		Status:         "pending",
		CreatedAt:      time.Now(),
	}

	// Store experiment
	if err := s.repo.CreateBitrateExperiment(ctx, experiment); err != nil {
		return nil, fmt.Errorf("failed to create experiment: %w", err)
	}

	// Run experiment in background
	go s.runExperimentAsync(context.Background(), experiment)

	return experiment, nil
}

// runExperimentAsync runs the experiment asynchronously
func (s *QualityService) runExperimentAsync(ctx context.Context, experiment *models.BitrateExperiment) {
	// Update status to running
	experiment.Status = "running"
	now := time.Now()
	experiment.StartedAt = &now
	s.repo.UpdateBitrateExperiment(ctx, experiment)

	// Get video
	video, err := s.repo.GetVideo(ctx, experiment.VideoID)
	if err != nil {
		experiment.Status = "failed"
		s.repo.UpdateBitrateExperiment(ctx, experiment)
		return
	}

	// Download video
	tempPath := fmt.Sprintf("%s/exp_%s", s.cfg.TempDir, experiment.ID)
	if err := s.storage.DownloadFile(ctx, video.OriginalURL, tempPath); err != nil {
		experiment.Status = "failed"
		s.repo.UpdateBitrateExperiment(ctx, experiment)
		return
	}
	defer removeFile(tempPath)

	// Encode and analyze each bitrate point
	totalSize := int64(0)
	vmafScores := make([]float64, 0)
	startTime := time.Now()

	for _, point := range experiment.LadderConfig {
		analysis, err := s.optimizer.TestBitratePoint(
			ctx,
			tempPath,
			point.Resolution,
			point.Bitrate,
			"libx264",
		)

		if err != nil {
			continue
		}

		if analysis.VMAFScore != nil {
			vmafScores = append(vmafScores, *analysis.VMAFScore)
		}

		// Estimate size (bitrate * duration)
		estimatedSize := point.Bitrate * int64(video.Duration) / 8
		totalSize += estimatedSize
	}

	encodingTime := time.Since(startTime).Seconds()

	// Calculate averages
	avgVMAF := 0.0
	minVMAF := 100.0
	for _, score := range vmafScores {
		avgVMAF += score
		if score < minVMAF {
			minVMAF = score
		}
	}
	if len(vmafScores) > 0 {
		avgVMAF /= float64(len(vmafScores))
	}

	// Update experiment results
	experiment.Status = "completed"
	experiment.TotalSize = &totalSize
	experiment.AvgVMAFScore = &avgVMAF
	experiment.MinVMAFScore = &minVMAF
	experiment.EncodingTime = &encodingTime
	completed := time.Now()
	experiment.CompletedAt = &completed

	s.repo.UpdateBitrateExperiment(ctx, experiment)
}

// GetRecommendedProfile gets the recommended encoding profile for a video
func (s *QualityService) GetRecommendedProfile(ctx context.Context, videoID string) (*models.EncodingProfile, error) {
	// Get active profiles
	profiles, err := s.repo.GetEncodingProfilesByVideoID(ctx, videoID)
	if err != nil {
		return nil, err
	}

	// Find the best profile (highest confidence, best estimated reduction)
	var bestProfile *models.EncodingProfile
	bestScore := 0.0

	for i := range profiles {
		if !profiles[i].IsActive {
			continue
		}

		score := 0.0
		if profiles[i].ConfidenceScore != nil {
			score += *profiles[i].ConfidenceScore * 0.7
		}
		if profiles[i].EstimatedSizeReduction != nil && *profiles[i].EstimatedSizeReduction > 0 {
			score += (*profiles[i].EstimatedSizeReduction / 100.0) * 0.3
		}

		if score > bestScore {
			bestScore = score
			bestProfile = &profiles[i]
		}
	}

	if bestProfile == nil && len(profiles) > 0 {
		bestProfile = &profiles[0]
	}

	return bestProfile, nil
}

// Helper function
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
