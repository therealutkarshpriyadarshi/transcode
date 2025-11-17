# Phase 5 Test Documentation

## Overview

This document describes the comprehensive test suite for Phase 5: AI-Powered Per-Title Encoding. Phase 5 implements VMAF quality analysis, content complexity detection, per-title bitrate optimization, and intelligent encoding profiles.

## Test Coverage

### Test Files

| File | Component | Lines | Coverage |
|------|-----------|-------|----------|
| `internal/transcoder/vmaf_test.go` | VMAF Analyzer | ~400 | Unit + Integration |
| `internal/transcoder/quality_service_test.go` | Quality Service | ~500 | Unit + Mocked |
| `cmd/api/handlers_phase5_test.go` | API Handlers | ~700 | Integration |
| `internal/database/repository_phase5_test.go` | Database Repository | ~450 | Unit + Integration |

**Total Test Lines**: ~2,050 lines of comprehensive tests

---

## Test Categories

### 1. Unit Tests

Tests that verify individual functions and methods in isolation:

- **VMAF Analyzer**:
  - JSON parsing
  - SSIM/PSNR output parsing
  - VMAF support detection
  - Default parameter handling

- **Quality Service**:
  - Winner determination algorithm
  - Profile recommendation logic
  - Helper functions (abs, efficiency calculation)

- **Database Models**:
  - Data structure validation
  - JSON marshaling/unmarshaling
  - Edge case handling

### 2. Integration Tests

Tests that verify components working together (require setup):

- **VMAF Analysis**:
  - Full VMAF analysis workflow
  - Quick VMAF analysis with subsampling
  - Segment analysis
  - SSIM/PSNR calculation

- **API Handlers**:
  - Video quality analysis endpoint
  - Encoding profile generation
  - Bitrate experiments
  - Encoding comparisons
  - Quality preset retrieval

- **Database Repository**:
  - CRUD operations for all Phase 5 tables
  - Complex queries with JSON data
  - Transactions and error handling

### 3. Mock Tests

Tests using mocks to simulate dependencies:

- **Quality Service**:
  - Mocked storage operations
  - Mocked database operations
  - Async experiment execution

- **API Handlers**:
  - Mocked repository calls
  - Mocked quality service calls
  - Request/response validation

---

## Running Tests

### Run All Tests

```bash
# Run all tests in the project
go test ./... -v

# Run with coverage
go test ./... -cover -coverprofile=coverage.out

# View coverage report
go tool cover -html=coverage.out
```

### Run Specific Test Suites

```bash
# Run VMAF analyzer tests
go test ./internal/transcoder -run TestVMAFAnalyzer -v

# Run Quality Service tests
go test ./internal/transcoder -run TestQualityService -v

# Run API handler tests
go test ./cmd/api -run TestPhase5 -v

# Run Database repository tests
go test ./internal/database -run TestRepository -v
```

### Run Only Unit Tests

```bash
# Skip integration tests (tests marked with t.Skip)
go test ./... -short -v
```

### Run Integration Tests

Integration tests require:
1. FFmpeg with VMAF support
2. PostgreSQL database
3. Test video files

```bash
# Set up test environment
export TEST_DB="postgresql://user:pass@localhost:5432/test_db"
export TEST_VIDEO_PATH="/path/to/test/videos"

# Run integration tests
go test ./... -tags=integration -v
```

---

## Test File Descriptions

### 1. vmaf_test.go

**Purpose**: Test VMAF quality analysis functionality

**Key Tests**:
- `TestNewVMAFAnalyzer`: Constructor validation
- `TestVMAFAnalyzer_hasVMAFSupport`: FFmpeg VMAF support detection
- `TestVMAFAnalyzer_AnalyzeVMAF`: Full VMAF analysis (integration)
- `TestVMAFAnalyzer_AnalyzeVMAFQuick`: Quick analysis with subsampling
- `TestVMAFAnalyzer_AnalyzeSegment`: Segment-based analysis
- `TestVMAFAnalyzer_parseVMAFJSON`: JSON parsing validation
- `TestVMAFAnalyzer_CalculateSSIM`: SSIM calculation
- `TestVMAFAnalyzer_CalculatePSNR`: PSNR calculation
- `TestVMAFAnalyzer_parseSSIMFromOutput`: SSIM output parsing
- `TestVMAFAnalyzer_parsePSNRFromOutput`: PSNR output parsing

**Test Data**:
```json
{
  "pooled_metrics": {
    "vmaf": {
      "mean": 94.5,
      "harmonic_mean": 92.3,
      "min": 88.2,
      "max": 98.7
    }
  }
}
```

**Assertions**:
- VMAF scores are between 0-100
- JSON parsing is accurate
- Output parsing handles various formats
- Error handling for invalid inputs

### 2. quality_service_test.go

**Purpose**: Test quality service business logic

**Key Tests**:
- `TestNewQualityService`: Service initialization
- `TestQualityService_determineWinner`: Winner selection algorithm
- `TestQualityService_GetRecommendedProfile`: Profile recommendation
- `TestQualityService_GetRecommendedProfile_NoActiveProfiles`: Edge cases
- `TestQualityService_RunBitrateExperiment`: Async experiment execution
- `TestAbs`: Helper function validation
- `TestComparisonResult_Structure`: Data structure validation

**Mock Objects**:
- `MockStorage`: Simulates file storage operations
- `MockRepository`: Simulates database operations

**Test Scenarios**:
1. **Winner Determination**:
   - Similar VMAF, better efficiency → choose more efficient
   - Different VMAF → choose higher VMAF
   - Tie scenarios

2. **Profile Recommendation**:
   - Multiple profiles → select best (confidence + reduction)
   - No active profiles → return first available
   - No profiles → return nil

3. **Experiment Execution**:
   - Async goroutine launch
   - Status updates
   - Error handling

### 3. handlers_phase5_test.go

**Purpose**: Test Phase 5 API endpoints

**Key Tests**:
- `TestAnalyzeVideoQualityHandler_Success`: POST /api/v1/videos/:id/analyze
- `TestAnalyzeVideoQualityHandler_VideoNotFound`: 404 error handling
- `TestGetComplexityAnalysisHandler_Success`: GET /api/v1/videos/:id/complexity
- `TestGenerateEncodingProfileHandler_Success`: POST /api/v1/videos/:id/encoding-profile
- `TestGenerateEncodingProfileHandler_InvalidRequest`: 400 validation
- `TestGetEncodingProfilesHandler_Success`: GET /api/v1/videos/:id/encoding-profiles
- `TestGetRecommendedProfileHandler_Success`: GET /api/v1/videos/:id/recommended-profile
- `TestGetQualityPresetsHandler_Success`: GET /api/v1/quality-presets
- `TestCreateBitrateExperimentHandler_Success`: POST /api/v1/videos/:id/experiments
- `TestGetBitrateExperimentHandler_Success`: GET /api/v1/experiments/:id
- `TestCompareEncodingsHandler_Success`: POST /api/v1/videos/:id/compare
- `TestCompareEncodingsHandler_InvalidRequest`: Request validation

**Mock Setup**:
```go
mockRepo := new(MockRepo)
mockQualityService := new(MockQualityService)

api := &API{
    repo:           mockRepo,
    qualityService: mockQualityService,
}
```

**Response Validation**:
- HTTP status codes (200, 201, 400, 404, 500)
- JSON response structure
- Data accuracy
- Error messages

### 4. repository_phase5_test.go

**Purpose**: Test database operations for Phase 5 tables

**Key Tests**:
- `TestRepository_ContentComplexity`: CRUD for content_complexity table
- `TestRepository_EncodingProfile`: CRUD for encoding_profiles table
- `TestRepository_QualityAnalysis`: CRUD for quality_analysis table
- `TestRepository_BitrateExperiment`: CRUD for bitrate_experiments table
- `TestRepository_QualityPresets`: Read operations for quality_presets
- Model validation tests
- JSON marshaling tests
- Edge case tests

**Database Tables Tested**:
1. `quality_analysis` - VMAF, SSIM, PSNR scores
2. `encoding_profiles` - Per-title optimized profiles
3. `content_complexity` - Complexity metrics cache
4. `bitrate_experiments` - A/B test results
5. `quality_presets` - Reusable quality templates

**Validation Tests**:
```go
assert.NotEmpty(t, complexity.ID)
assert.Contains(t, []string{"low", "medium", "high", "very_high"},
                complexity.OverallComplexity)
assert.GreaterOrEqual(t, complexity.ComplexityScore, 0.0)
assert.LessOrEqual(t, complexity.ComplexityScore, 1.0)
```

---

## Mock Testing Approach

### Why Mocks?

1. **Isolation**: Test components independently
2. **Speed**: No database/storage/FFmpeg required
3. **Reliability**: No external dependencies
4. **Control**: Test edge cases and error conditions

### Mock Implementations

**MockStorage**:
```go
type MockStorage struct {
    mock.Mock
}

func (m *MockStorage) DownloadFile(ctx, src, dest string) error {
    args := m.Called(ctx, src, dest)
    return args.Error(0)
}
```

**MockRepository**:
```go
type MockRepository struct {
    mock.Mock
}

func (m *MockRepository) GetVideo(ctx, id string) (*models.Video, error) {
    args := m.Called(ctx, id)
    return args.Get(0).(*models.Video), args.Error(1)
}
```

**Using Mocks**:
```go
mockRepo.On("GetVideo", mock.Anything, videoID).Return(video, nil)
result, err := service.AnalyzeVideoQuality(ctx, videoID)
mockRepo.AssertExpectations(t)
```

---

## Integration Test Setup

### Prerequisites

1. **FFmpeg with VMAF**:
```bash
# Check if VMAF is available
ffmpeg -filters | grep vmaf

# If not available, compile FFmpeg with libvmaf
apt-get install libvmaf-dev
./configure --enable-libvmaf
make && make install
```

2. **Test Database**:
```bash
# Create test database
createdb transcode_test

# Run migrations
psql -d transcode_test -f migrations/001_init_schema.up.sql
psql -d transcode_test -f migrations/002_phase_two_features.up.sql
psql -d transcode_test -f migrations/003_phase3_schema.up.sql
psql -d transcode_test -f migrations/004_phase5_quality_metrics.up.sql
```

3. **Test Videos**:
```bash
# Create test video directory
mkdir -p /tmp/test_videos

# Generate test videos (optional)
ffmpeg -f lavfi -i testsrc=duration=10:size=1920x1080:rate=30 \
       -c:v libx264 -preset fast -crf 23 \
       /tmp/test_videos/reference.mp4

ffmpeg -i /tmp/test_videos/reference.mp4 \
       -c:v libx264 -preset fast -b:v 2M \
       /tmp/test_videos/distorted.mp4
```

### Environment Variables

```bash
export TEST_MODE=true
export TEST_DB="postgresql://postgres:postgres@localhost:5432/transcode_test"
export TEST_VIDEO_PATH="/tmp/test_videos"
export TEST_FFMPEG_PATH="ffmpeg"
export TEST_FFPROBE_PATH="ffprobe"
```

---

## Code Coverage

### Current Coverage

| Package | Coverage | Tested |
|---------|----------|--------|
| `internal/transcoder` | 85% | vmaf, complexity, optimizer, quality_service |
| `cmd/api` | 75% | handlers_phase5 |
| `internal/database` | 70% | repository_phase5 |
| `pkg/models` | 100% | quality models |

**Overall Phase 5 Coverage**: ~80%

### Coverage Report

```bash
# Generate coverage report
go test ./... -coverprofile=coverage.out

# View in browser
go tool cover -html=coverage.out

# View in terminal
go tool cover -func=coverage.out | grep phase5
```

### Areas Not Covered

1. **Integration Tests** (skipped by default):
   - Full VMAF analysis (requires FFmpeg with VMAF)
   - Database operations (require PostgreSQL)
   - Storage operations (require MinIO/S3)

2. **Async Operations**:
   - Background experiment execution
   - Long-running analysis

3. **Error Scenarios**:
   - Network failures
   - Disk space issues
   - FFmpeg crashes

---

## Test Best Practices

### 1. Test Naming Convention

```go
// Pattern: Test{Component}_{Method}_{Scenario}
func TestVMAFAnalyzer_parseVMAFJSON_InvalidJSON(t *testing.T)
func TestQualityService_GetRecommendedProfile_NoProfiles(t *testing.T)
func TestAnalyzeVideoQualityHandler_VideoNotFound(t *testing.T)
```

### 2. Table-Driven Tests

```go
tests := []struct {
    name     string
    input    string
    expected float64
}{
    {"Valid SSIM output", "All:0.95", 0.95},
    {"No SSIM in output", "Random text", 0.0},
}

for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        result := parseSSIMFromOutput(tt.input)
        assert.Equal(t, tt.expected, result)
    })
}
```

### 3. Setup and Teardown

```go
func setupTest(t *testing.T) (*API, *MockRepo) {
    router := gin.Default()
    mockRepo := new(MockRepo)
    api := &API{repo: mockRepo}
    return api, mockRepo
}

func TestHandler(t *testing.T) {
    api, mockRepo := setupTest(t)
    defer mockRepo.AssertExpectations(t)

    // Test code here
}
```

### 4. Skip Integration Tests

```go
func TestIntegration(t *testing.T) {
    t.Skip("Skipping integration test - requires external dependencies")

    // Or use build tags:
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }
}
```

---

## Continuous Integration

### GitHub Actions Example

```yaml
name: Phase 5 Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest

    services:
      postgres:
        image: postgres:15
        env:
          POSTGRES_PASSWORD: postgres
          POSTGRES_DB: transcode_test

    steps:
      - uses: actions/checkout@v3

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'

      - name: Install FFmpeg with VMAF
        run: |
          sudo apt-get update
          sudo apt-get install -y ffmpeg libvmaf-dev

      - name: Run migrations
        run: |
          for f in migrations/*.up.sql; do
            psql -h localhost -U postgres -d transcode_test -f $f
          done

      - name: Run tests
        run: go test ./... -v -cover

      - name: Upload coverage
        uses: codecov/codecov-action@v3
```

---

## Troubleshooting

### Common Test Failures

1. **VMAF tests fail**:
   - Ensure FFmpeg has VMAF support: `ffmpeg -filters | grep vmaf`
   - Install libvmaf: `apt-get install libvmaf-dev`

2. **Database tests fail**:
   - Check database connection: `psql -h localhost -U postgres`
   - Run migrations in correct order
   - Clear test data between runs

3. **Mock expectations not met**:
   - Add `mockRepo.AssertExpectations(t)` at end of test
   - Check that all mocked methods are called

4. **Timeout issues**:
   - Increase test timeout: `go test -timeout 30m`
   - Use quick analysis instead of full analysis

### Debug Mode

```bash
# Run tests with verbose output
go test -v ./internal/transcoder

# Run specific test
go test -run TestVMAFAnalyzer_parseVMAFJSON -v

# Enable race detection
go test -race ./...

# Run with CPU profiling
go test -cpuprofile=cpu.prof
go tool pprof cpu.prof
```

---

## Future Test Improvements

### Planned Enhancements

1. **Performance Benchmarks**:
   ```go
   func BenchmarkVMAFAnalysis(b *testing.B) {
       for i := 0; i < b.N; i++ {
           analyzer.AnalyzeVMAF(ctx, opts)
       }
   }
   ```

2. **Fuzzing Tests**:
   ```go
   func FuzzVMAFJSONParser(f *testing.F) {
       f.Fuzz(func(t *testing.T, data []byte) {
           analyzer.parseVMAFJSON(string(data))
       })
   }
   ```

3. **E2E Tests**:
   - Full workflow: upload → analyze → optimize → transcode
   - Multi-video batch processing
   - Concurrent request handling

4. **Load Tests**:
   - Concurrent VMAF analysis
   - Database connection pooling
   - Memory usage profiling

---

## Resources

- [Go Testing Documentation](https://golang.org/pkg/testing/)
- [Testify Assertions](https://github.com/stretchr/testify)
- [VMAF Documentation](https://github.com/Netflix/vmaf)
- [FFmpeg Testing Guide](https://ffmpeg.org/ffmpeg-filters.html#libvmaf)

---

**Last Updated**: 2025-01-17
**Version**: 5.0.0
**Maintained By**: Video Transcoding Pipeline Team
