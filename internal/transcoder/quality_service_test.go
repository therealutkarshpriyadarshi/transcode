package transcoder

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/therealutkarshpriyadarshi/transcode/internal/config"
	"github.com/therealutkarshpriyadarshi/transcode/pkg/models"
)

// MockStorage is a mock implementation of storage.Storage
type MockStorage struct {
	mock.Mock
}

func (m *MockStorage) UploadFile(ctx context.Context, sourcePath, destPath string) error {
	args := m.Called(ctx, sourcePath, destPath)
	return args.Error(0)
}

func (m *MockStorage) DownloadFile(ctx context.Context, sourcePath, destPath string) error {
	args := m.Called(ctx, sourcePath, destPath)
	return args.Error(0)
}

func (m *MockStorage) DeleteFile(ctx context.Context, path string) error {
	args := m.Called(ctx, path)
	return args.Error(0)
}

func (m *MockStorage) GetFileURL(ctx context.Context, path string) (string, error) {
	args := m.Called(ctx, path)
	return args.String(0), args.Error(1)
}

func (m *MockStorage) FileExists(ctx context.Context, path string) (bool, error) {
	args := m.Called(ctx, path)
	return args.Bool(0), args.Error(1)
}

// MockRepository is a mock implementation of database.Repository
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) GetVideo(ctx context.Context, id string) (*models.Video, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Video), args.Error(1)
}

func (m *MockRepository) CreateContentComplexity(ctx context.Context, complexity *models.ContentComplexity) error {
	args := m.Called(ctx, complexity)
	return args.Error(0)
}

func (m *MockRepository) GetContentComplexity(ctx context.Context, videoID string) (*models.ContentComplexity, error) {
	args := m.Called(ctx, videoID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ContentComplexity), args.Error(1)
}

func (m *MockRepository) GetQualityPresetByName(ctx context.Context, name string) (*models.QualityPreset, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.QualityPreset), args.Error(1)
}

func (m *MockRepository) CreateEncodingProfile(ctx context.Context, profile *models.EncodingProfile) error {
	args := m.Called(ctx, profile)
	return args.Error(0)
}

func (m *MockRepository) GetEncodingProfile(ctx context.Context, id string) (*models.EncodingProfile, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.EncodingProfile), args.Error(1)
}

func (m *MockRepository) GetEncodingProfilesByVideoID(ctx context.Context, videoID string) ([]models.EncodingProfile, error) {
	args := m.Called(ctx, videoID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.EncodingProfile), args.Error(1)
}

func (m *MockRepository) GetOutput(ctx context.Context, id string) (*models.Output, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Output), args.Error(1)
}

func (m *MockRepository) CreateBitrateExperiment(ctx context.Context, experiment *models.BitrateExperiment) error {
	args := m.Called(ctx, experiment)
	return args.Error(0)
}

func (m *MockRepository) UpdateBitrateExperiment(ctx context.Context, experiment *models.BitrateExperiment) error {
	args := m.Called(ctx, experiment)
	return args.Error(0)
}

func (m *MockRepository) GetBitrateExperiment(ctx context.Context, id string) (*models.BitrateExperiment, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.BitrateExperiment), args.Error(1)
}

func TestNewQualityService(t *testing.T) {
	cfg := config.TranscoderConfig{
		FFmpegPath:  "ffmpeg",
		FFprobePath: "ffprobe",
		TempDir:     "/tmp",
	}

	mockStorage := new(MockStorage)
	mockRepo := new(MockRepository)

	service := NewQualityService(cfg, mockStorage, mockRepo)

	assert.NotNil(t, service)
	assert.NotNil(t, service.ffmpeg)
	assert.NotNil(t, service.storage)
	assert.NotNil(t, service.repo)
	assert.NotNil(t, service.optimizer)
	assert.NotNil(t, service.vmaf)
	assert.NotNil(t, service.complexity)
}

func TestQualityService_determineWinner(t *testing.T) {
	cfg := config.TranscoderConfig{
		FFmpegPath:  "ffmpeg",
		FFprobePath: "ffprobe",
		TempDir:     "/tmp",
	}

	mockStorage := new(MockStorage)
	mockRepo := new(MockRepository)
	service := NewQualityService(cfg, mockStorage, mockRepo)

	tests := []struct {
		name     string
		eff1     float64
		eff2     float64
		vmaf1    float64
		vmaf2    float64
		expected string
	}{
		{
			name:     "Similar VMAF, better efficiency in output1",
			eff1:     50000.0,
			eff2:     60000.0,
			vmaf1:    95.0,
			vmaf2:    95.5,
			expected: "output1",
		},
		{
			name:     "Similar VMAF, better efficiency in output2",
			eff1:     60000.0,
			eff2:     50000.0,
			vmaf1:    95.0,
			vmaf2:    95.5,
			expected: "output2",
		},
		{
			name:     "Different VMAF, output1 better",
			eff1:     60000.0,
			eff2:     50000.0,
			vmaf1:    96.0,
			vmaf2:    92.0,
			expected: "output1",
		},
		{
			name:     "Different VMAF, output2 better",
			eff1:     50000.0,
			eff2:     60000.0,
			vmaf1:    92.0,
			vmaf2:    96.0,
			expected: "output2",
		},
		{
			name:     "Tie scenario",
			eff1:     50000.0,
			eff2:     50000.0,
			vmaf1:    95.0,
			vmaf2:    95.5,
			expected: "tie",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.determineWinner(tt.eff1, tt.eff2, tt.vmaf1, tt.vmaf2)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestQualityService_GetRecommendedProfile(t *testing.T) {
	cfg := config.TranscoderConfig{
		FFmpegPath:  "ffmpeg",
		FFprobePath: "ffprobe",
		TempDir:     "/tmp",
	}

	mockStorage := new(MockStorage)
	mockRepo := new(MockRepository)
	service := NewQualityService(cfg, mockStorage, mockRepo)

	ctx := context.Background()
	videoID := "test-video-id"

	// Test case 1: Multiple profiles, select best one
	confidence1 := 0.8
	reduction1 := 15.0
	confidence2 := 0.9
	reduction2 := 20.0

	profiles := []models.EncodingProfile{
		{
			ID:                     "profile1",
			VideoID:                videoID,
			ProfileName:            "standard",
			IsActive:               true,
			ConfidenceScore:        &confidence1,
			EstimatedSizeReduction: &reduction1,
		},
		{
			ID:                     "profile2",
			VideoID:                videoID,
			ProfileName:            "optimized",
			IsActive:               true,
			ConfidenceScore:        &confidence2,
			EstimatedSizeReduction: &reduction2,
		},
	}

	mockRepo.On("GetEncodingProfilesByVideoID", ctx, videoID).Return(profiles, nil)

	result, err := service.GetRecommendedProfile(ctx, videoID)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "profile2", result.ID) // Should select profile2 with higher confidence and reduction

	mockRepo.AssertExpectations(t)
}

func TestQualityService_GetRecommendedProfile_NoActiveProfiles(t *testing.T) {
	cfg := config.TranscoderConfig{
		FFmpegPath:  "ffmpeg",
		FFprobePath: "ffprobe",
		TempDir:     "/tmp",
	}

	mockStorage := new(MockStorage)
	mockRepo := new(MockRepository)
	service := NewQualityService(cfg, mockStorage, mockRepo)

	ctx := context.Background()
	videoID := "test-video-id"

	// All profiles are inactive
	confidence := 0.8
	profiles := []models.EncodingProfile{
		{
			ID:              "profile1",
			VideoID:         videoID,
			ProfileName:     "inactive",
			IsActive:        false,
			ConfidenceScore: &confidence,
		},
	}

	mockRepo.On("GetEncodingProfilesByVideoID", ctx, videoID).Return(profiles, nil)

	result, err := service.GetRecommendedProfile(ctx, videoID)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "profile1", result.ID) // Should return first profile even if inactive

	mockRepo.AssertExpectations(t)
}

func TestQualityService_GetRecommendedProfile_NoProfiles(t *testing.T) {
	cfg := config.TranscoderConfig{
		FFmpegPath:  "ffmpeg",
		FFprobePath: "ffprobe",
		TempDir:     "/tmp",
	}

	mockStorage := new(MockStorage)
	mockRepo := new(MockRepository)
	service := NewQualityService(cfg, mockStorage, mockRepo)

	ctx := context.Background()
	videoID := "test-video-id"

	mockRepo.On("GetEncodingProfilesByVideoID", ctx, videoID).Return([]models.EncodingProfile{}, nil)

	result, err := service.GetRecommendedProfile(ctx, videoID)
	assert.NoError(t, err)
	assert.Nil(t, result)

	mockRepo.AssertExpectations(t)
}

func TestQualityService_RunBitrateExperiment(t *testing.T) {
	cfg := config.TranscoderConfig{
		FFmpegPath:  "ffmpeg",
		FFprobePath: "ffprobe",
		TempDir:     "/tmp",
	}

	mockStorage := new(MockStorage)
	mockRepo := new(MockRepository)
	service := NewQualityService(cfg, mockStorage, mockRepo)

	ctx := context.Background()
	videoID := "test-video-id"
	experimentName := "Test Experiment"
	ladderConfig := []models.BitratePoint{
		{Resolution: "1080p", Bitrate: 5000000},
		{Resolution: "720p", Bitrate: 2500000},
	}

	mockRepo.On("CreateBitrateExperiment", ctx, mock.AnythingOfType("*models.BitrateExperiment")).Return(nil)

	result, err := service.RunBitrateExperiment(ctx, videoID, experimentName, ladderConfig)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, videoID, result.VideoID)
	assert.Equal(t, experimentName, result.ExperimentName)
	assert.Equal(t, "pending", result.Status)
	assert.Len(t, result.LadderConfig, 2)

	mockRepo.AssertExpectations(t)

	// Give some time for async goroutine to start
	time.Sleep(100 * time.Millisecond)
}

func TestAbs(t *testing.T) {
	tests := []struct {
		name     string
		input    float64
		expected float64
	}{
		{"Positive number", 5.5, 5.5},
		{"Negative number", -5.5, 5.5},
		{"Zero", 0.0, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := abs(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestComparisonResult_Structure(t *testing.T) {
	result := &ComparisonResult{
		Output1: EncodingMetrics{
			OutputID:   "output1",
			VMAF:       95.0,
			Size:       500000000,
			Bitrate:    8000000,
			Efficiency: 84210.5,
		},
		Output2: EncodingMetrics{
			OutputID:   "output2",
			VMAF:       94.0,
			Size:       400000000,
			Bitrate:    6400000,
			Efficiency: 68085.1,
		},
		Winner: "output2",
	}

	assert.Equal(t, "output1", result.Output1.OutputID)
	assert.Equal(t, 95.0, result.Output1.VMAF)
	assert.Equal(t, "output2", result.Output2.OutputID)
	assert.Equal(t, "output2", result.Winner)
}

func TestEncodingMetrics_Structure(t *testing.T) {
	metrics := EncodingMetrics{
		OutputID:   "test-output",
		VMAF:       93.5,
		Size:       450000000,
		Bitrate:    7200000,
		Efficiency: 77005.3,
	}

	assert.Equal(t, "test-output", metrics.OutputID)
	assert.Equal(t, 93.5, metrics.VMAF)
	assert.Equal(t, int64(450000000), metrics.Size)
	assert.Equal(t, int64(7200000), metrics.Bitrate)
	assert.InDelta(t, 77005.3, metrics.Efficiency, 0.1)
}
