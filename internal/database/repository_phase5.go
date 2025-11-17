package database

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/therealutkarshpriyadarshi/transcode/pkg/models"
)

// Phase 5: Quality Analysis and Per-Title Encoding Repository Methods

// CreateQualityAnalysis creates a new quality analysis record
func (r *Repository) CreateQualityAnalysis(ctx context.Context, analysis *models.QualityAnalysis) error {
	query := `
		INSERT INTO quality_analysis (
			id, video_id, analysis_type, segment_index, segment_start_time, segment_duration,
			vmaf_score, vmaf_min, vmaf_max, vmaf_mean, vmaf_harmonic_mean,
			ssim_score, psnr_score,
			spatial_complexity, temporal_complexity, scene_complexity, motion_score,
			test_bitrate, test_resolution, test_codec, test_preset,
			analyzed_at, analysis_duration
		) VALUES (
			gen_random_uuid(), $1, $2, $3, $4, $5,
			$6, $7, $8, $9, $10,
			$11, $12,
			$13, $14, $15, $16,
			$17, $18, $19, $20,
			$21, $22
		) RETURNING id
	`

	return r.db.QueryRow(ctx, query,
		analysis.VideoID, analysis.AnalysisType, analysis.SegmentIndex, analysis.SegmentStartTime, analysis.SegmentDuration,
		analysis.VMAFScore, analysis.VMAFMin, analysis.VMAFMax, analysis.VMAFMean, analysis.VMAFHarmonicMean,
		analysis.SSIMScore, analysis.PSNRScore,
		analysis.SpatialComplexity, analysis.TemporalComplexity, analysis.SceneComplexity, analysis.MotionScore,
		analysis.TestBitrate, analysis.TestResolution, analysis.TestCodec, analysis.TestPreset,
		analysis.AnalyzedAt, analysis.AnalysisDuration,
	).Scan(&analysis.ID)
}

// GetQualityAnalysisByVideoID retrieves all quality analyses for a video
func (r *Repository) GetQualityAnalysisByVideoID(ctx context.Context, videoID string) ([]models.QualityAnalysis, error) {
	query := `
		SELECT id, video_id, analysis_type, segment_index, segment_start_time, segment_duration,
			vmaf_score, vmaf_min, vmaf_max, vmaf_mean, vmaf_harmonic_mean,
			ssim_score, psnr_score,
			spatial_complexity, temporal_complexity, scene_complexity, motion_score,
			test_bitrate, test_resolution, test_codec, test_preset,
			analyzed_at, analysis_duration
		FROM quality_analysis
		WHERE video_id = $1
		ORDER BY analyzed_at DESC
	`

	rows, err := r.db.Query(ctx, query, videoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	analyses := make([]models.QualityAnalysis, 0)
	for rows.Next() {
		var a models.QualityAnalysis
		err := rows.Scan(
			&a.ID, &a.VideoID, &a.AnalysisType, &a.SegmentIndex, &a.SegmentStartTime, &a.SegmentDuration,
			&a.VMAFScore, &a.VMAFMin, &a.VMAFMax, &a.VMAFMean, &a.VMAFHarmonicMean,
			&a.SSIMScore, &a.PSNRScore,
			&a.SpatialComplexity, &a.TemporalComplexity, &a.SceneComplexity, &a.MotionScore,
			&a.TestBitrate, &a.TestResolution, &a.TestCodec, &a.TestPreset,
			&a.AnalyzedAt, &a.AnalysisDuration,
		)
		if err != nil {
			return nil, err
		}
		analyses = append(analyses, a)
	}

	return analyses, rows.Err()
}

// CreateEncodingProfile creates a new encoding profile
func (r *Repository) CreateEncodingProfile(ctx context.Context, profile *models.EncodingProfile) error {
	ladderJSON, err := json.Marshal(profile.BitrateeLadder)
	if err != nil {
		return fmt.Errorf("failed to marshal bitrate ladder: %w", err)
	}

	query := `
		INSERT INTO encoding_profiles (
			id, video_id, profile_name, is_active,
			content_type, complexity_level,
			bitrate_ladder, codec_recommendation, preset_recommendation,
			target_vmaf_score, min_vmaf_score,
			estimated_size_reduction, estimated_quality_improvement, confidence_score,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4,
			$5, $6,
			$7, $8, $9,
			$10, $11,
			$12, $13, $14,
			$15, $16
		)
	`

	_, err = r.db.Exec(ctx, query,
		profile.ID, profile.VideoID, profile.ProfileName, profile.IsActive,
		profile.ContentType, profile.ComplexityLevel,
		ladderJSON, profile.CodecRecommendation, profile.PresetRecommendation,
		profile.TargetVMAFScore, profile.MinVMAFScore,
		profile.EstimatedSizeReduction, profile.EstimatedQualityImprovement, profile.ConfidenceScore,
		profile.CreatedAt, profile.UpdatedAt,
	)

	return err
}

// GetEncodingProfile retrieves an encoding profile by ID
func (r *Repository) GetEncodingProfile(ctx context.Context, profileID string) (*models.EncodingProfile, error) {
	query := `
		SELECT id, video_id, profile_name, is_active,
			content_type, complexity_level,
			bitrate_ladder, codec_recommendation, preset_recommendation,
			target_vmaf_score, min_vmaf_score,
			estimated_size_reduction, estimated_quality_improvement, confidence_score,
			created_at, updated_at
		FROM encoding_profiles
		WHERE id = $1
	`

	var profile models.EncodingProfile
	var ladderJSON []byte

	err := r.db.QueryRow(ctx, query, profileID).Scan(
		&profile.ID, &profile.VideoID, &profile.ProfileName, &profile.IsActive,
		&profile.ContentType, &profile.ComplexityLevel,
		&ladderJSON, &profile.CodecRecommendation, &profile.PresetRecommendation,
		&profile.TargetVMAFScore, &profile.MinVMAFScore,
		&profile.EstimatedSizeReduction, &profile.EstimatedQualityImprovement, &profile.ConfidenceScore,
		&profile.CreatedAt, &profile.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(ladderJSON, &profile.BitrateeLadder); err != nil {
		return nil, fmt.Errorf("failed to unmarshal bitrate ladder: %w", err)
	}

	return &profile, nil
}

// GetEncodingProfilesByVideoID retrieves all encoding profiles for a video
func (r *Repository) GetEncodingProfilesByVideoID(ctx context.Context, videoID string) ([]models.EncodingProfile, error) {
	query := `
		SELECT id, video_id, profile_name, is_active,
			content_type, complexity_level,
			bitrate_ladder, codec_recommendation, preset_recommendation,
			target_vmaf_score, min_vmaf_score,
			estimated_size_reduction, estimated_quality_improvement, confidence_score,
			created_at, updated_at
		FROM encoding_profiles
		WHERE video_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(ctx, query, videoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	profiles := make([]models.EncodingProfile, 0)
	for rows.Next() {
		var profile models.EncodingProfile
		var ladderJSON []byte

		err := rows.Scan(
			&profile.ID, &profile.VideoID, &profile.ProfileName, &profile.IsActive,
			&profile.ContentType, &profile.ComplexityLevel,
			&ladderJSON, &profile.CodecRecommendation, &profile.PresetRecommendation,
			&profile.TargetVMAFScore, &profile.MinVMAFScore,
			&profile.EstimatedSizeReduction, &profile.EstimatedQualityImprovement, &profile.ConfidenceScore,
			&profile.CreatedAt, &profile.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		if err := json.Unmarshal(ladderJSON, &profile.BitrateeLadder); err != nil {
			continue
		}

		profiles = append(profiles, profile)
	}

	return profiles, rows.Err()
}

// CreateContentComplexity creates a content complexity record
func (r *Repository) CreateContentComplexity(ctx context.Context, complexity *models.ContentComplexity) error {
	query := `
		INSERT INTO content_complexity (
			id, video_id,
			overall_complexity, complexity_score,
			avg_spatial_info, max_spatial_info, min_spatial_info,
			avg_temporal_info, max_temporal_info, min_temporal_info,
			avg_motion_intensity, motion_variance, scene_changes,
			color_variance, edge_density, contrast_ratio,
			content_category, has_text_overlay, has_fast_motion,
			sample_points, analyzed_at
		) VALUES (
			$1, $2,
			$3, $4,
			$5, $6, $7,
			$8, $9, $10,
			$11, $12, $13,
			$14, $15, $16,
			$17, $18, $19,
			$20, $21
		)
	`

	_, err := r.db.Exec(ctx, query,
		complexity.ID, complexity.VideoID,
		complexity.OverallComplexity, complexity.ComplexityScore,
		complexity.AvgSpatialInfo, complexity.MaxSpatialInfo, complexity.MinSpatialInfo,
		complexity.AvgTemporalInfo, complexity.MaxTemporalInfo, complexity.MinTemporalInfo,
		complexity.AvgMotionIntensity, complexity.MotionVariance, complexity.SceneChanges,
		complexity.ColorVariance, complexity.EdgeDensity, complexity.ContrastRatio,
		complexity.ContentCategory, complexity.HasTextOverlay, complexity.HasFastMotion,
		complexity.SamplePoints, complexity.AnalyzedAt,
	)

	return err
}

// GetContentComplexity retrieves content complexity for a video
func (r *Repository) GetContentComplexity(ctx context.Context, videoID string) (*models.ContentComplexity, error) {
	query := `
		SELECT id, video_id,
			overall_complexity, complexity_score,
			avg_spatial_info, max_spatial_info, min_spatial_info,
			avg_temporal_info, max_temporal_info, min_temporal_info,
			avg_motion_intensity, motion_variance, scene_changes,
			color_variance, edge_density, contrast_ratio,
			content_category, has_text_overlay, has_fast_motion,
			sample_points, analyzed_at
		FROM content_complexity
		WHERE video_id = $1
	`

	var complexity models.ContentComplexity
	err := r.db.QueryRow(ctx, query, videoID).Scan(
		&complexity.ID, &complexity.VideoID,
		&complexity.OverallComplexity, &complexity.ComplexityScore,
		&complexity.AvgSpatialInfo, &complexity.MaxSpatialInfo, &complexity.MinSpatialInfo,
		&complexity.AvgTemporalInfo, &complexity.MaxTemporalInfo, &complexity.MinTemporalInfo,
		&complexity.AvgMotionIntensity, &complexity.MotionVariance, &complexity.SceneChanges,
		&complexity.ColorVariance, &complexity.EdgeDensity, &complexity.ContrastRatio,
		&complexity.ContentCategory, &complexity.HasTextOverlay, &complexity.HasFastMotion,
		&complexity.SamplePoints, &complexity.AnalyzedAt,
	)

	if err != nil {
		return nil, err
	}

	return &complexity, nil
}

// CreateBitrateExperiment creates a bitrate experiment
func (r *Repository) CreateBitrateExperiment(ctx context.Context, experiment *models.BitrateExperiment) error {
	ladderJSON, err := json.Marshal(experiment.LadderConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal ladder config: %w", err)
	}

	paramsJSON := []byte("{}")
	if experiment.EncodingParams != nil {
		paramsJSON, err = json.Marshal(experiment.EncodingParams)
		if err != nil {
			return fmt.Errorf("failed to marshal encoding params: %w", err)
		}
	}

	query := `
		INSERT INTO bitrate_experiments (
			id, video_id, experiment_name,
			ladder_config, encoding_params,
			total_size, avg_vmaf_score, min_vmaf_score, encoding_time,
			size_vs_baseline, quality_vs_baseline,
			status, started_at, completed_at, created_at
		) VALUES (
			$1, $2, $3,
			$4, $5,
			$6, $7, $8, $9,
			$10, $11,
			$12, $13, $14, $15
		)
	`

	_, err = r.db.Exec(ctx, query,
		experiment.ID, experiment.VideoID, experiment.ExperimentName,
		ladderJSON, paramsJSON,
		experiment.TotalSize, experiment.AvgVMAFScore, experiment.MinVMAFScore, experiment.EncodingTime,
		experiment.SizeVsBaseline, experiment.QualityVsBaseline,
		experiment.Status, experiment.StartedAt, experiment.CompletedAt, experiment.CreatedAt,
	)

	return err
}

// UpdateBitrateExperiment updates a bitrate experiment
func (r *Repository) UpdateBitrateExperiment(ctx context.Context, experiment *models.BitrateExperiment) error {
	query := `
		UPDATE bitrate_experiments
		SET total_size = $1, avg_vmaf_score = $2, min_vmaf_score = $3, encoding_time = $4,
			size_vs_baseline = $5, quality_vs_baseline = $6,
			status = $7, started_at = $8, completed_at = $9
		WHERE id = $10
	`

	_, err := r.db.Exec(ctx, query,
		experiment.TotalSize, experiment.AvgVMAFScore, experiment.MinVMAFScore, experiment.EncodingTime,
		experiment.SizeVsBaseline, experiment.QualityVsBaseline,
		experiment.Status, experiment.StartedAt, experiment.CompletedAt,
		experiment.ID,
	)

	return err
}

// GetBitrateExperiment retrieves a bitrate experiment
func (r *Repository) GetBitrateExperiment(ctx context.Context, experimentID string) (*models.BitrateExperiment, error) {
	query := `
		SELECT id, video_id, experiment_name,
			ladder_config, encoding_params,
			total_size, avg_vmaf_score, min_vmaf_score, encoding_time,
			size_vs_baseline, quality_vs_baseline,
			status, started_at, completed_at, created_at
		FROM bitrate_experiments
		WHERE id = $1
	`

	var experiment models.BitrateExperiment
	var ladderJSON, paramsJSON []byte

	err := r.db.QueryRow(ctx, query, experimentID).Scan(
		&experiment.ID, &experiment.VideoID, &experiment.ExperimentName,
		&ladderJSON, &paramsJSON,
		&experiment.TotalSize, &experiment.AvgVMAFScore, &experiment.MinVMAFScore, &experiment.EncodingTime,
		&experiment.SizeVsBaseline, &experiment.QualityVsBaseline,
		&experiment.Status, &experiment.StartedAt, &experiment.CompletedAt, &experiment.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(ladderJSON, &experiment.LadderConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal ladder config: %w", err)
	}

	if len(paramsJSON) > 0 {
		if err := json.Unmarshal(paramsJSON, &experiment.EncodingParams); err != nil {
			return nil, fmt.Errorf("failed to unmarshal encoding params: %w", err)
		}
	}

	return &experiment, nil
}

// GetQualityPresets retrieves all active quality presets
func (r *Repository) GetQualityPresets(ctx context.Context) ([]models.QualityPreset, error) {
	query := `
		SELECT id, name, description,
			target_vmaf, min_vmaf,
			prefer_quality, max_bitrate_multiplier, min_bitrate_multiplier,
			standard_ladder,
			is_active, created_at, updated_at
		FROM quality_presets
		WHERE is_active = true
		ORDER BY name
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	presets := make([]models.QualityPreset, 0)
	for rows.Next() {
		var preset models.QualityPreset
		var ladderJSON []byte

		err := rows.Scan(
			&preset.ID, &preset.Name, &preset.Description,
			&preset.TargetVMAF, &preset.MinVMAF,
			&preset.PreferQuality, &preset.MaxBitrateMultiplier, &preset.MinBitrateMultiplier,
			&ladderJSON,
			&preset.IsActive, &preset.CreatedAt, &preset.UpdatedAt,
		)
		if err != nil {
			continue
		}

		if err := json.Unmarshal(ladderJSON, &preset.StandardLadder); err != nil {
			continue
		}

		presets = append(presets, preset)
	}

	return presets, rows.Err()
}

// GetQualityPresetByName retrieves a quality preset by name
func (r *Repository) GetQualityPresetByName(ctx context.Context, name string) (*models.QualityPreset, error) {
	query := `
		SELECT id, name, description,
			target_vmaf, min_vmaf,
			prefer_quality, max_bitrate_multiplier, min_bitrate_multiplier,
			standard_ladder,
			is_active, created_at, updated_at
		FROM quality_presets
		WHERE name = $1 AND is_active = true
	`

	var preset models.QualityPreset
	var ladderJSON []byte

	err := r.db.QueryRow(ctx, query, name).Scan(
		&preset.ID, &preset.Name, &preset.Description,
		&preset.TargetVMAF, &preset.MinVMAF,
		&preset.PreferQuality, &preset.MaxBitrateMultiplier, &preset.MinBitrateMultiplier,
		&ladderJSON,
		&preset.IsActive, &preset.CreatedAt, &preset.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(ladderJSON, &preset.StandardLadder); err != nil {
		return nil, fmt.Errorf("failed to unmarshal standard ladder: %w", err)
	}

	return &preset, nil
}

// GetOutput retrieves an output by ID
func (r *Repository) GetOutput(ctx context.Context, outputID string) (*models.Output, error) {
	query := `
		SELECT id, job_id, video_id, format, resolution, width, height,
			codec, bitrate, size, duration, url, path, created_at
		FROM outputs
		WHERE id = $1
	`

	var output models.Output
	err := r.db.QueryRow(ctx, query, outputID).Scan(
		&output.ID, &output.JobID, &output.VideoID, &output.Format, &output.Resolution,
		&output.Width, &output.Height, &output.Codec, &output.Bitrate, &output.Size,
		&output.Duration, &output.URL, &output.Path, &output.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &output, nil
}
