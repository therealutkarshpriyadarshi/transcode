package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/therealutkarshpriyadarshi/transcode/pkg/models"
)

// MockQualityService is a mock implementation of QualityService
type MockQualityService struct {
	mock.Mock
}

func (m *MockQualityService) AnalyzeVideoQuality(ctx interface{}, videoID string) (*models.ContentComplexity, error) {
	args := m.Called(ctx, videoID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ContentComplexity), args.Error(1)
}

func (m *MockQualityService) GenerateEncodingProfile(ctx interface{}, videoID, presetName string) (*models.EncodingProfile, error) {
	args := m.Called(ctx, videoID, presetName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.EncodingProfile), args.Error(1)
}

func (m *MockQualityService) GetRecommendedProfile(ctx interface{}, videoID string) (*models.EncodingProfile, error) {
	args := m.Called(ctx, videoID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.EncodingProfile), args.Error(1)
}

func (m *MockQualityService) CompareEncodings(ctx interface{}, videoID, output1ID, output2ID string) (interface{}, error) {
	args := m.Called(ctx, videoID, output1ID, output2ID)
	return args.Get(0), args.Error(1)
}

func (m *MockQualityService) RunBitrateExperiment(ctx interface{}, videoID, name string, ladderConfig []models.BitratePoint) (*models.BitrateExperiment, error) {
	args := m.Called(ctx, videoID, name, ladderConfig)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.BitrateExperiment), args.Error(1)
}

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.Default()
}

func TestAnalyzeVideoQualityHandler_Success(t *testing.T) {
	router := setupTestRouter()
	mockRepo := new(MockRepo)
	mockQualityService := new(MockQualityService)

	api := &API{
		repo:           mockRepo,
		qualityService: mockQualityService,
	}

	videoID := "test-video-123"
	video := &models.Video{
		ID:        videoID,
		Filename:  "test.mp4",
		Status:    models.VideoStatusCompleted,
		CreatedAt: time.Now(),
	}

	complexity := &models.ContentComplexity{
		ID:                "complexity-123",
		VideoID:           videoID,
		OverallComplexity: "high",
		ComplexityScore:   0.75,
		AvgSpatialInfo:    65.2,
		AvgTemporalInfo:   28.4,
	}

	mockRepo.On("GetVideo", mock.Anything, videoID).Return(video, nil)
	mockQualityService.On("AnalyzeVideoQuality", mock.Anything, videoID).Return(complexity, nil)

	router.POST("/api/v1/videos/:id/analyze", api.analyzeVideoQualityHandler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/videos/"+videoID+"/analyze", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, videoID, response["video_id"])
	assert.NotNil(t, response["complexity"])

	mockRepo.AssertExpectations(t)
	mockQualityService.AssertExpectations(t)
}

func TestAnalyzeVideoQualityHandler_VideoNotFound(t *testing.T) {
	router := setupTestRouter()
	mockRepo := new(MockRepo)
	mockQualityService := new(MockQualityService)

	api := &API{
		repo:           mockRepo,
		qualityService: mockQualityService,
	}

	videoID := "nonexistent"
	mockRepo.On("GetVideo", mock.Anything, videoID).Return(nil, assert.AnError)

	router.POST("/api/v1/videos/:id/analyze", api.analyzeVideoQualityHandler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/videos/"+videoID+"/analyze", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestGetComplexityAnalysisHandler_Success(t *testing.T) {
	router := setupTestRouter()
	mockRepo := new(MockRepo)

	api := &API{
		repo: mockRepo,
	}

	videoID := "test-video-123"
	complexity := &models.ContentComplexity{
		ID:                "complexity-123",
		VideoID:           videoID,
		OverallComplexity: "medium",
		ComplexityScore:   0.55,
	}

	mockRepo.On("GetContentComplexity", mock.Anything, videoID).Return(complexity, nil)

	router.GET("/api/v1/videos/:id/complexity", api.getComplexityAnalysisHandler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/videos/"+videoID+"/complexity", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response models.ContentComplexity
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, videoID, response.VideoID)

	mockRepo.AssertExpectations(t)
}

func TestGenerateEncodingProfileHandler_Success(t *testing.T) {
	router := setupTestRouter()
	mockRepo := new(MockRepo)
	mockQualityService := new(MockQualityService)

	api := &API{
		repo:           mockRepo,
		qualityService: mockQualityService,
	}

	videoID := "test-video-123"
	presetName := "high_quality"
	video := &models.Video{
		ID:       videoID,
		Filename: "test.mp4",
		Status:   models.VideoStatusCompleted,
	}

	targetVMAF := 95.0
	confidence := 0.85
	profile := &models.EncodingProfile{
		ID:                  "profile-123",
		VideoID:             videoID,
		ProfileName:         presetName,
		ComplexityLevel:     "high",
		TargetVMAFScore:     &targetVMAF,
		ConfidenceScore:     &confidence,
		CodecRecommendation: "libx265",
	}

	mockRepo.On("GetVideo", mock.Anything, videoID).Return(video, nil)
	mockQualityService.On("GenerateEncodingProfile", mock.Anything, videoID, presetName).Return(profile, nil)

	router.POST("/api/v1/videos/:id/encoding-profile", api.generateEncodingProfileHandler)

	requestBody := map[string]string{
		"preset_name": presetName,
	}
	jsonBody, _ := json.Marshal(requestBody)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/videos/"+videoID+"/encoding-profile", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, videoID, response["video_id"])
	assert.NotNil(t, response["profile"])

	mockRepo.AssertExpectations(t)
	mockQualityService.AssertExpectations(t)
}

func TestGenerateEncodingProfileHandler_InvalidRequest(t *testing.T) {
	router := setupTestRouter()
	mockRepo := new(MockRepo)

	api := &API{
		repo: mockRepo,
	}

	videoID := "test-video-123"

	router.POST("/api/v1/videos/:id/encoding-profile", api.generateEncodingProfileHandler)

	// Missing required preset_name
	requestBody := map[string]string{}
	jsonBody, _ := json.Marshal(requestBody)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/videos/"+videoID+"/encoding-profile", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetEncodingProfilesHandler_Success(t *testing.T) {
	router := setupTestRouter()
	mockRepo := new(MockRepo)

	api := &API{
		repo: mockRepo,
	}

	videoID := "test-video-123"
	profiles := []models.EncodingProfile{
		{
			ID:              "profile-1",
			VideoID:         videoID,
			ProfileName:     "high_quality",
			ComplexityLevel: "high",
		},
		{
			ID:              "profile-2",
			VideoID:         videoID,
			ProfileName:     "standard_quality",
			ComplexityLevel: "medium",
		},
	}

	mockRepo.On("GetEncodingProfilesByVideoID", mock.Anything, videoID).Return(profiles, nil)

	router.GET("/api/v1/videos/:id/encoding-profiles", api.getEncodingProfilesHandler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/videos/"+videoID+"/encoding-profiles", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, videoID, response["video_id"])
	assert.Equal(t, float64(2), response["count"])

	mockRepo.AssertExpectations(t)
}

func TestGetRecommendedProfileHandler_Success(t *testing.T) {
	router := setupTestRouter()
	mockRepo := new(MockRepo)
	mockQualityService := new(MockQualityService)

	api := &API{
		repo:           mockRepo,
		qualityService: mockQualityService,
	}

	videoID := "test-video-123"
	confidence := 0.9
	profile := &models.EncodingProfile{
		ID:              "profile-1",
		VideoID:         videoID,
		ProfileName:     "optimized",
		ConfidenceScore: &confidence,
	}

	mockQualityService.On("GetRecommendedProfile", mock.Anything, videoID).Return(profile, nil)

	router.GET("/api/v1/videos/:id/recommended-profile", api.getRecommendedProfileHandler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/videos/"+videoID+"/recommended-profile", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response models.EncodingProfile
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, videoID, response.VideoID)

	mockQualityService.AssertExpectations(t)
}

func TestGetQualityPresetsHandler_Success(t *testing.T) {
	router := setupTestRouter()
	mockRepo := new(MockRepo)

	api := &API{
		repo: mockRepo,
	}

	presets := []models.QualityPreset{
		{
			ID:          "preset-1",
			Name:        "high_quality",
			Description: "High quality encoding",
			TargetVMAF:  95.0,
		},
		{
			ID:          "preset-2",
			Name:        "standard_quality",
			Description: "Standard quality encoding",
			TargetVMAF:  87.0,
		},
	}

	mockRepo.On("GetQualityPresets", mock.Anything).Return(presets, nil)

	router.GET("/api/v1/quality-presets", api.getQualityPresetsHandler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/quality-presets", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, float64(2), response["count"])

	mockRepo.AssertExpectations(t)
}

func TestCreateBitrateExperimentHandler_Success(t *testing.T) {
	router := setupTestRouter()
	mockRepo := new(MockRepo)
	mockQualityService := new(MockQualityService)

	api := &API{
		repo:           mockRepo,
		qualityService: mockQualityService,
	}

	videoID := "test-video-123"
	experimentName := "Test Experiment"
	video := &models.Video{
		ID:       videoID,
		Filename: "test.mp4",
		Status:   models.VideoStatusCompleted,
	}

	ladderConfig := []models.BitratePoint{
		{Resolution: "1080p", Bitrate: 5000000},
		{Resolution: "720p", Bitrate: 2500000},
	}

	experiment := &models.BitrateExperiment{
		ID:             "exp-123",
		VideoID:        videoID,
		ExperimentName: experimentName,
		LadderConfig:   ladderConfig,
		Status:         "pending",
	}

	mockRepo.On("GetVideo", mock.Anything, videoID).Return(video, nil)
	mockQualityService.On("RunBitrateExperiment", mock.Anything, videoID, experimentName, ladderConfig).Return(experiment, nil)

	router.POST("/api/v1/videos/:id/experiments", api.createBitrateExperimentHandler)

	requestBody := map[string]interface{}{
		"name":          experimentName,
		"ladder_config": ladderConfig,
	}
	jsonBody, _ := json.Marshal(requestBody)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/videos/"+videoID+"/experiments", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.NotNil(t, response["experiment"])

	mockRepo.AssertExpectations(t)
	mockQualityService.AssertExpectations(t)
}

func TestGetBitrateExperimentHandler_Success(t *testing.T) {
	router := setupTestRouter()
	mockRepo := new(MockRepo)

	api := &API{
		repo: mockRepo,
	}

	experimentID := "exp-123"
	avgVMAF := 92.5
	experiment := &models.BitrateExperiment{
		ID:             experimentID,
		VideoID:        "video-123",
		ExperimentName: "Test Experiment",
		Status:         "completed",
		AvgVMAFScore:   &avgVMAF,
	}

	mockRepo.On("GetBitrateExperiment", mock.Anything, experimentID).Return(experiment, nil)

	router.GET("/api/v1/experiments/:id", api.getBitrateExperimentHandler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/experiments/"+experimentID, nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response models.BitrateExperiment
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, experimentID, response.ID)

	mockRepo.AssertExpectations(t)
}

func TestCompareEncodingsHandler_Success(t *testing.T) {
	router := setupTestRouter()
	mockRepo := new(MockRepo)
	mockQualityService := new(MockQualityService)

	api := &API{
		repo:           mockRepo,
		qualityService: mockQualityService,
	}

	videoID := "test-video-123"
	output1ID := "output-1"
	output2ID := "output-2"

	comparisonResult := map[string]interface{}{
		"output1": map[string]interface{}{
			"output_id":  output1ID,
			"vmaf":       95.2,
			"size":       500000000,
			"bitrate":    8000000,
			"efficiency": 84033.6,
		},
		"output2": map[string]interface{}{
			"output_id":  output2ID,
			"vmaf":       94.8,
			"size":       350000000,
			"bitrate":    5600000,
			"efficiency": 59071.7,
		},
		"winner": "output2",
	}

	mockQualityService.On("CompareEncodings", mock.Anything, videoID, output1ID, output2ID).Return(comparisonResult, nil)

	router.POST("/api/v1/videos/:id/compare", api.compareEncodingsHandler)

	requestBody := map[string]string{
		"output1_id": output1ID,
		"output2_id": output2ID,
	}
	jsonBody, _ := json.Marshal(requestBody)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/videos/"+videoID+"/compare", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, videoID, response["video_id"])
	assert.NotNil(t, response["comparison"])

	mockQualityService.AssertExpectations(t)
}

func TestCompareEncodingsHandler_InvalidRequest(t *testing.T) {
	router := setupTestRouter()
	api := &API{}

	videoID := "test-video-123"

	router.POST("/api/v1/videos/:id/compare", api.compareEncodingsHandler)

	// Missing required output IDs
	requestBody := map[string]string{}
	jsonBody, _ := json.Marshal(requestBody)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/videos/"+videoID+"/compare", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// MockRepo is a mock implementation for testing
type MockRepo struct {
	mock.Mock
}

func (m *MockRepo) GetVideo(ctx interface{}, id string) (*models.Video, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Video), args.Error(1)
}

func (m *MockRepo) GetContentComplexity(ctx interface{}, videoID string) (*models.ContentComplexity, error) {
	args := m.Called(ctx, videoID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ContentComplexity), args.Error(1)
}

func (m *MockRepo) GetEncodingProfilesByVideoID(ctx interface{}, videoID string) ([]models.EncodingProfile, error) {
	args := m.Called(ctx, videoID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.EncodingProfile), args.Error(1)
}

func (m *MockRepo) GetEncodingProfile(ctx interface{}, id string) (*models.EncodingProfile, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.EncodingProfile), args.Error(1)
}

func (m *MockRepo) GetQualityPresets(ctx interface{}) ([]models.QualityPreset, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.QualityPreset), args.Error(1)
}

func (m *MockRepo) GetBitrateExperiment(ctx interface{}, id string) (*models.BitrateExperiment, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.BitrateExperiment), args.Error(1)
}

func (m *MockRepo) GetQualityAnalysisByVideoID(ctx interface{}, videoID string) ([]models.QualityAnalysis, error) {
	args := m.Called(ctx, videoID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.QualityAnalysis), args.Error(1)
}
