package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/therealutkarshpriyadarshi/transcode/pkg/models"
)

// Phase 5: Quality Analysis and Per-Title Encoding API Handlers

// analyzeVideoQualityHandler analyzes video quality and complexity
// POST /api/v1/videos/:id/analyze
func (api *API) analyzeVideoQualityHandler(c *gin.Context) {
	videoID := c.Param("id")

	// Check if video exists
	video, err := api.repo.GetVideo(c.Request.Context(), videoID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "video not found"})
		return
	}

	// Check if video is ready
	if video.Status != models.VideoStatusCompleted && video.Status != models.VideoStatusPending {
		c.JSON(http.StatusBadRequest, gin.H{"error": "video must be uploaded before analysis"})
		return
	}

	// Analyze quality
	complexity, err := api.qualityService.AnalyzeVideoQuality(c.Request.Context(), videoID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to analyze video quality", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"video_id":   videoID,
		"complexity": complexity,
		"message":    "video quality analysis completed",
	})
}

// getComplexityAnalysisHandler retrieves complexity analysis for a video
// GET /api/v1/videos/:id/complexity
func (api *API) getComplexityAnalysisHandler(c *gin.Context) {
	videoID := c.Param("id")

	complexity, err := api.repo.GetContentComplexity(c.Request.Context(), videoID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "complexity analysis not found"})
		return
	}

	c.JSON(http.StatusOK, complexity)
}

// generateEncodingProfileHandler generates an optimized encoding profile
// POST /api/v1/videos/:id/encoding-profile
func (api *API) generateEncodingProfileHandler(c *gin.Context) {
	videoID := c.Param("id")

	var req struct {
		PresetName string `json:"preset_name" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request", "details": err.Error()})
		return
	}

	// Check if video exists
	if _, err := api.repo.GetVideo(c.Request.Context(), videoID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "video not found"})
		return
	}

	// Generate encoding profile
	profile, err := api.qualityService.GenerateEncodingProfile(c.Request.Context(), videoID, req.PresetName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate encoding profile", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"video_id": videoID,
		"profile":  profile,
		"message":  "encoding profile generated successfully",
	})
}

// getEncodingProfilesHandler retrieves encoding profiles for a video
// GET /api/v1/videos/:id/encoding-profiles
func (api *API) getEncodingProfilesHandler(c *gin.Context) {
	videoID := c.Param("id")

	profiles, err := api.repo.GetEncodingProfilesByVideoID(c.Request.Context(), videoID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve profiles"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"video_id": videoID,
		"profiles": profiles,
		"count":    len(profiles),
	})
}

// getRecommendedProfileHandler retrieves the recommended encoding profile
// GET /api/v1/videos/:id/recommended-profile
func (api *API) getRecommendedProfileHandler(c *gin.Context) {
	videoID := c.Param("id")

	profile, err := api.qualityService.GetRecommendedProfile(c.Request.Context(), videoID)
	if err != nil || profile == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no recommended profile found"})
		return
	}

	c.JSON(http.StatusOK, profile)
}

// transcodeWithProfileHandler creates transcoding jobs using an encoding profile
// POST /api/v1/videos/:id/transcode-with-profile
func (api *API) transcodeWithProfileHandler(c *gin.Context) {
	videoID := c.Param("id")

	var req struct {
		ProfileID string `json:"profile_id"`
		Priority  int    `json:"priority"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	// Get the profile
	var profile *models.EncodingProfile
	var err error

	if req.ProfileID != "" {
		profile, err = api.repo.GetEncodingProfile(c.Request.Context(), req.ProfileID)
	} else {
		// Use recommended profile
		profile, err = api.qualityService.GetRecommendedProfile(c.Request.Context(), videoID)
	}

	if err != nil || profile == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "encoding profile not found"})
		return
	}

	// Create jobs for each bitrate point in the ladder
	jobs := make([]*models.Job, 0)

	for _, point := range profile.BitrateeLadder {
		job := &models.Job{
			VideoID:  videoID,
			Status:   models.JobStatusPending,
			Priority: req.Priority,
			Config: models.JobConfig{
				Resolution:   point.Resolution,
				Codec:        profile.CodecRecommendation,
				Bitrate:      point.Bitrate,
				Preset:       profile.PresetRecommendation,
				OutputFormat: "mp4",
			},
		}

		if profile.TargetVMAFScore != nil {
			job.TargetVMAF = *profile.TargetVMAFScore
		}

		if err := api.repo.CreateJob(c.Request.Context(), job); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create job"})
			return
		}

		// Publish to queue
		if err := api.queue.PublishJob(c.Request.Context(), job); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to queue job"})
			return
		}

		jobs = append(jobs, job)
	}

	c.JSON(http.StatusCreated, gin.H{
		"video_id":   videoID,
		"profile_id": profile.ID,
		"jobs":       jobs,
		"jobs_count": len(jobs),
		"message":    "transcoding jobs created with optimized profile",
	})
}

// getQualityPresetsHandler retrieves available quality presets
// GET /api/v1/quality-presets
func (api *API) getQualityPresetsHandler(c *gin.Context) {
	presets, err := api.repo.GetQualityPresets(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve presets"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"presets": presets,
		"count":   len(presets),
	})
}

// createBitrateExperimentHandler creates a bitrate experiment
// POST /api/v1/videos/:id/experiments
func (api *API) createBitrateExperimentHandler(c *gin.Context) {
	videoID := c.Param("id")

	var req struct {
		Name         string                 `json:"name" binding:"required"`
		LadderConfig []models.BitratePoint  `json:"ladder_config" binding:"required"`
		Params       map[string]interface{} `json:"params"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	// Check if video exists
	if _, err := api.repo.GetVideo(c.Request.Context(), videoID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "video not found"})
		return
	}

	// Create experiment
	experiment, err := api.qualityService.RunBitrateExperiment(
		c.Request.Context(),
		videoID,
		req.Name,
		req.LadderConfig,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create experiment"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"experiment": experiment,
		"message":    "experiment created and running in background",
	})
}

// getBitrateExperimentHandler retrieves a bitrate experiment
// GET /api/v1/experiments/:id
func (api *API) getBitrateExperimentHandler(c *gin.Context) {
	experimentID := c.Param("id")

	experiment, err := api.repo.GetBitrateExperiment(c.Request.Context(), experimentID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "experiment not found"})
		return
	}

	c.JSON(http.StatusOK, experiment)
}

// getQualityAnalysisHandler retrieves quality analyses for a video
// GET /api/v1/videos/:id/quality-analysis
func (api *API) getQualityAnalysisHandler(c *gin.Context) {
	videoID := c.Param("id")

	analyses, err := api.repo.GetQualityAnalysisByVideoID(c.Request.Context(), videoID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve analyses"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"video_id": videoID,
		"analyses": analyses,
		"count":    len(analyses),
	})
}

// compareEncodingsHandler compares two encodings using VMAF
// POST /api/v1/videos/:id/compare
func (api *API) compareEncodingsHandler(c *gin.Context) {
	videoID := c.Param("id")

	var req struct {
		Output1ID string `json:"output1_id" binding:"required"`
		Output2ID string `json:"output2_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	// Compare encodings
	result, err := api.qualityService.CompareEncodings(
		c.Request.Context(),
		videoID,
		req.Output1ID,
		req.Output2ID,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to compare encodings", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"video_id":   videoID,
		"comparison": result,
	})
}

// registerPhase5Routes registers Phase 5 API routes
func (api *API) registerPhase5Routes(router *gin.Engine) {
	v1 := router.Group("/api/v1")

	// Quality analysis routes
	v1.POST("/videos/:id/analyze", api.analyzeVideoQualityHandler)
	v1.GET("/videos/:id/complexity", api.getComplexityAnalysisHandler)
	v1.GET("/videos/:id/quality-analysis", api.getQualityAnalysisHandler)

	// Encoding profile routes
	v1.POST("/videos/:id/encoding-profile", api.generateEncodingProfileHandler)
	v1.GET("/videos/:id/encoding-profiles", api.getEncodingProfilesHandler)
	v1.GET("/videos/:id/recommended-profile", api.getRecommendedProfileHandler)
	v1.POST("/videos/:id/transcode-with-profile", api.transcodeWithProfileHandler)

	// Quality presets
	v1.GET("/quality-presets", api.getQualityPresetsHandler)

	// Experiments
	v1.POST("/videos/:id/experiments", api.createBitrateExperimentHandler)
	v1.GET("/experiments/:id", api.getBitrateExperimentHandler)

	// Comparisons
	v1.POST("/videos/:id/compare", api.compareEncodingsHandler)
}
