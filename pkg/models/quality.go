package models

import "time"

// QualityAnalysis represents a quality analysis result
type QualityAnalysis struct {
	ID                 string    `json:"id" db:"id"`
	VideoID            string    `json:"video_id" db:"video_id"`
	AnalysisType       string    `json:"analysis_type" db:"analysis_type"` // 'vmaf', 'ssim', 'psnr', 'complexity'
	SegmentIndex       *int      `json:"segment_index,omitempty" db:"segment_index"`
	SegmentStartTime   *float64  `json:"segment_start_time,omitempty" db:"segment_start_time"`
	SegmentDuration    *float64  `json:"segment_duration,omitempty" db:"segment_duration"`

	// VMAF scores
	VMAFScore          *float64  `json:"vmaf_score,omitempty" db:"vmaf_score"`
	VMAFMin            *float64  `json:"vmaf_min,omitempty" db:"vmaf_min"`
	VMAFMax            *float64  `json:"vmaf_max,omitempty" db:"vmaf_max"`
	VMAFMean           *float64  `json:"vmaf_mean,omitempty" db:"vmaf_mean"`
	VMAFHarmonicMean   *float64  `json:"vmaf_harmonic_mean,omitempty" db:"vmaf_harmonic_mean"`

	// Other quality metrics
	SSIMScore          *float64  `json:"ssim_score,omitempty" db:"ssim_score"`
	PSNRScore          *float64  `json:"psnr_score,omitempty" db:"psnr_score"`

	// Complexity metrics
	SpatialComplexity  *float64  `json:"spatial_complexity,omitempty" db:"spatial_complexity"`
	TemporalComplexity *float64  `json:"temporal_complexity,omitempty" db:"temporal_complexity"`
	SceneComplexity    string    `json:"scene_complexity,omitempty" db:"scene_complexity"` // 'low', 'medium', 'high'
	MotionScore        *float64  `json:"motion_score,omitempty" db:"motion_score"`

	// Encoding settings used
	TestBitrate        *int64    `json:"test_bitrate,omitempty" db:"test_bitrate"`
	TestResolution     string    `json:"test_resolution,omitempty" db:"test_resolution"`
	TestCodec          string    `json:"test_codec,omitempty" db:"test_codec"`
	TestPreset         string    `json:"test_preset,omitempty" db:"test_preset"`

	// Analysis metadata
	AnalyzedAt         time.Time `json:"analyzed_at" db:"analyzed_at"`
	AnalysisDuration   *float64  `json:"analysis_duration,omitempty" db:"analysis_duration"`
}

// EncodingProfile represents a per-title optimized encoding profile
type EncodingProfile struct {
	ID                       string    `json:"id" db:"id"`
	VideoID                  string    `json:"video_id" db:"video_id"`
	ProfileName              string    `json:"profile_name" db:"profile_name"`
	IsActive                 bool      `json:"is_active" db:"is_active"`

	// Video characteristics
	ContentType              string    `json:"content_type,omitempty" db:"content_type"`
	ComplexityLevel          string    `json:"complexity_level" db:"complexity_level"`

	// Recommended settings (stored as JSONB)
	BitrateeLadder           []BitratePoint `json:"bitrate_ladder" db:"bitrate_ladder"`
	CodecRecommendation      string    `json:"codec_recommendation" db:"codec_recommendation"`
	PresetRecommendation     string    `json:"preset_recommendation" db:"preset_recommendation"`

	// Quality targets
	TargetVMAFScore          *float64  `json:"target_vmaf_score" db:"target_vmaf_score"`
	MinVMAFScore             *float64  `json:"min_vmaf_score" db:"min_vmaf_score"`

	// Optimization results
	EstimatedSizeReduction   *float64  `json:"estimated_size_reduction,omitempty" db:"estimated_size_reduction"`
	EstimatedQualityImprovement *float64 `json:"estimated_quality_improvement,omitempty" db:"estimated_quality_improvement"`
	ConfidenceScore          *float64  `json:"confidence_score" db:"confidence_score"`

	CreatedAt                time.Time `json:"created_at" db:"created_at"`
	UpdatedAt                time.Time `json:"updated_at" db:"updated_at"`
}

// BitratePoint represents a single point in the bitrate ladder
type BitratePoint struct {
	Resolution  string  `json:"resolution"`
	Bitrate     int64   `json:"bitrate"`
	TargetVMAF  float64 `json:"target_vmaf,omitempty"`
}

// ContentComplexity represents content complexity analysis
type ContentComplexity struct {
	ID                   string    `json:"id" db:"id"`
	VideoID              string    `json:"video_id" db:"video_id"`

	// Overall complexity
	OverallComplexity    string    `json:"overall_complexity" db:"overall_complexity"` // 'low', 'medium', 'high', 'very_high'
	ComplexityScore      float64   `json:"complexity_score" db:"complexity_score"`

	// Spatial metrics
	AvgSpatialInfo       float64   `json:"avg_spatial_info" db:"avg_spatial_info"`
	MaxSpatialInfo       float64   `json:"max_spatial_info" db:"max_spatial_info"`
	MinSpatialInfo       float64   `json:"min_spatial_info" db:"min_spatial_info"`

	// Temporal metrics
	AvgTemporalInfo      float64   `json:"avg_temporal_info" db:"avg_temporal_info"`
	MaxTemporalInfo      float64   `json:"max_temporal_info" db:"max_temporal_info"`
	MinTemporalInfo      float64   `json:"min_temporal_info" db:"min_temporal_info"`

	// Motion analysis
	AvgMotionIntensity   float64   `json:"avg_motion_intensity" db:"avg_motion_intensity"`
	MotionVariance       float64   `json:"motion_variance" db:"motion_variance"`
	SceneChanges         int       `json:"scene_changes" db:"scene_changes"`

	// Color and detail
	ColorVariance        float64   `json:"color_variance" db:"color_variance"`
	EdgeDensity          float64   `json:"edge_density" db:"edge_density"`
	ContrastRatio        float64   `json:"contrast_ratio" db:"contrast_ratio"`

	// Content categorization
	ContentCategory      string    `json:"content_category,omitempty" db:"content_category"`
	HasTextOverlay       bool      `json:"has_text_overlay" db:"has_text_overlay"`
	HasFastMotion        bool      `json:"has_fast_motion" db:"has_fast_motion"`

	// Analysis metadata
	SamplePoints         int       `json:"sample_points" db:"sample_points"`
	AnalyzedAt           time.Time `json:"analyzed_at" db:"analyzed_at"`
}

// BitrateExperiment represents an A/B test for bitrate ladders
type BitrateExperiment struct {
	ID                  string                 `json:"id" db:"id"`
	VideoID             string                 `json:"video_id" db:"video_id"`
	ExperimentName      string                 `json:"experiment_name" db:"experiment_name"`

	// Experiment configuration
	LadderConfig        []BitratePoint         `json:"ladder_config" db:"ladder_config"`
	EncodingParams      map[string]interface{} `json:"encoding_params,omitempty" db:"encoding_params"`

	// Results
	TotalSize           *int64                 `json:"total_size,omitempty" db:"total_size"`
	AvgVMAFScore        *float64               `json:"avg_vmaf_score,omitempty" db:"avg_vmaf_score"`
	MinVMAFScore        *float64               `json:"min_vmaf_score,omitempty" db:"min_vmaf_score"`
	EncodingTime        *float64               `json:"encoding_time,omitempty" db:"encoding_time"`

	// Comparison to baseline
	SizeVsBaseline      *float64               `json:"size_vs_baseline,omitempty" db:"size_vs_baseline"`
	QualityVsBaseline   *float64               `json:"quality_vs_baseline,omitempty" db:"quality_vs_baseline"`

	// Status
	Status              string                 `json:"status" db:"status"` // 'pending', 'running', 'completed', 'failed'
	StartedAt           *time.Time             `json:"started_at,omitempty" db:"started_at"`
	CompletedAt         *time.Time             `json:"completed_at,omitempty" db:"completed_at"`
	CreatedAt           time.Time              `json:"created_at" db:"created_at"`
}

// QualityPreset represents a reusable quality preset
type QualityPreset struct {
	ID                     string         `json:"id" db:"id"`
	Name                   string         `json:"name" db:"name"`
	Description            string         `json:"description,omitempty" db:"description"`

	// Quality targets
	TargetVMAF             float64        `json:"target_vmaf" db:"target_vmaf"`
	MinVMAF                float64        `json:"min_vmaf" db:"min_vmaf"`

	// Encoding preferences
	PreferQuality          bool           `json:"prefer_quality" db:"prefer_quality"`
	MaxBitrateMultiplier   float64        `json:"max_bitrate_multiplier" db:"max_bitrate_multiplier"`
	MinBitrateMultiplier   float64        `json:"min_bitrate_multiplier" db:"min_bitrate_multiplier"`

	// Standard bitrate ladder
	StandardLadder         []BitratePoint `json:"standard_ladder" db:"standard_ladder"`

	IsActive               bool           `json:"is_active" db:"is_active"`
	CreatedAt              time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt              time.Time      `json:"updated_at" db:"updated_at"`
}

// VMAFResult represents VMAF analysis output
type VMAFResult struct {
	Score          float64 `json:"score"`
	Min            float64 `json:"min"`
	Max            float64 `json:"max"`
	Mean           float64 `json:"mean"`
	HarmonicMean   float64 `json:"harmonic_mean"`
}

// ComplexityMetrics represents video complexity metrics
type ComplexityMetrics struct {
	SpatialInfo   float64 `json:"si"`  // Spatial Information
	TemporalInfo  float64 `json:"ti"`  // Temporal Information
	Motion        float64 `json:"motion"`
	SceneChanges  int     `json:"scene_changes"`
}
