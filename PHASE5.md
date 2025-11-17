## Phase 5: AI-Powered Per-Title Encoding ✅

**Status:** Complete
**Version:** 5.0.0
**Completion Date:** 2025-01-17

---

## Overview

Phase 5 implements AI-powered per-title encoding optimization using VMAF quality analysis and content complexity detection. This phase enables intelligent, content-aware bitrate ladder generation that can reduce file sizes by 10-30% while maintaining or improving perceived video quality.

### Key Features

- ✅ **VMAF Quality Analysis**: Industry-standard video quality assessment
- ✅ **Content Complexity Analysis**: SI/TI metrics, motion detection, scene analysis
- ✅ **Per-Title Bitrate Ladders**: Dynamic optimization based on content characteristics
- ✅ **Quality Presets**: Reusable encoding profiles (high quality, standard, bandwidth-optimized)
- ✅ **A/B Testing Framework**: Experiment with different bitrate configurations
- ✅ **Encoding Comparison**: Compare multiple encodings with VMAF scores
- ✅ **Rule-Based Optimizer**: Intelligent codec and preset recommendations

---

## Architecture

### Components

```
┌─────────────────────────────────────────────────────────────┐
│                      Phase 5 Components                      │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌──────────────────┐     ┌──────────────────┐            │
│  │  VMAF Analyzer   │     │   Complexity     │            │
│  │                  │     │    Analyzer      │            │
│  │ - VMAF scoring   │     │                  │            │
│  │ - SSIM/PSNR      │     │ - SI/TI metrics  │            │
│  │ - Segment        │     │ - Motion         │            │
│  │   analysis       │     │   detection      │            │
│  └────────┬─────────┘     │ - Scene changes  │            │
│           │               │ - Color analysis │            │
│           │               └────────┬─────────┘            │
│           │                        │                       │
│           └────────┬───────────────┘                       │
│                    │                                       │
│          ┌─────────▼──────────┐                           │
│          │  Encoding Optimizer │                           │
│          │                     │                           │
│          │ - Bitrate ladder    │                           │
│          │   generation        │                           │
│          │ - Codec selection   │                           │
│          │ - Preset tuning     │                           │
│          └─────────┬───────────┘                           │
│                    │                                       │
│          ┌─────────▼──────────┐                           │
│          │  Quality Service    │                           │
│          │                     │                           │
│          │ - Profile mgmt      │                           │
│          │ - A/B testing       │                           │
│          │ - Comparisons       │                           │
│          └─────────────────────┘                           │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### Database Schema

Phase 5 adds the following tables:

1. **quality_analysis**: Stores VMAF, SSIM, PSNR scores and complexity metrics
2. **encoding_profiles**: Per-video optimized encoding configurations
3. **content_complexity**: Cached complexity analysis results
4. **bitrate_experiments**: A/B testing results for bitrate ladders
5. **quality_presets**: Reusable quality configuration templates

---

## Quality Analysis

### VMAF (Video Multimethod Assessment Fusion)

VMAF is Netflix's perceptual video quality assessment algorithm. It scores videos from 0-100, where:
- **95-100**: Excellent quality (virtually indistinguishable from source)
- **85-95**: High quality (imperceptible to most viewers)
- **75-85**: Good quality (minor artifacts under scrutiny)
- **<75**: Noticeable quality degradation

#### How VMAF Analysis Works

1. **Encode Test Video**: Transcode at target bitrate/resolution
2. **Compare with Reference**: Run VMAF filter comparing original vs encoded
3. **Generate Metrics**: Calculate mean, harmonic mean, min, max VMAF scores
4. **Store Results**: Save metrics to database for future optimization

### Content Complexity Metrics

#### Spatial Information (SI)
Measures spatial detail/complexity in each frame using Sobel edge detection. High SI indicates:
- Detailed textures
- Complex scenes
- High-frequency content

#### Temporal Information (TI)
Measures frame-to-frame changes. High TI indicates:
- Fast motion
- Scene changes
- Dynamic content

#### Motion Analysis
- **Motion Intensity**: Average motion vector magnitude
- **Motion Variance**: Consistency of motion
- **Scene Changes**: Number of detected scene transitions

#### Content Categorization
Automatically classifies content as:
- **Sports/Action**: High motion, frequent scene changes
- **Animation**: Low SI, consistent TI, flat colors
- **Presentation**: Low SI/TI, static content
- **Gaming**: High variance in motion
- **Movie**: Balanced metrics

---

## Per-Title Encoding

### Concept

Traditional transcoding uses fixed bitrate ladders for all content. Per-title encoding optimizes the ladder for each video based on its complexity:

- **Simple content** (presentations, animations): Use lower bitrates while maintaining quality
- **Complex content** (sports, action): Use higher bitrates to avoid artifacts

### Bitrate Ladder Optimization

#### Standard Ladder (Fixed)
```json
{
  "2160p": 25000000,  // 25 Mbps
  "1080p": 8000000,   // 8 Mbps
  "720p": 4000000,    // 4 Mbps
  "480p": 2000000,    // 2 Mbps
  "360p": 1000000     // 1 Mbps
}
```

#### Optimized Ladder (Animation Example)
```json
{
  "1080p": 5000000,   // 5 Mbps (-37.5%)
  "720p": 2500000,    // 2.5 Mbps (-37.5%)
  "480p": 1200000,    // 1.2 Mbps (-40%)
  "360p": 700000      // 700 Kbps (-30%)
}
```

#### Optimized Ladder (Sports Example)
```json
{
  "1080p": 10000000,  // 10 Mbps (+25%)
  "720p": 5000000,    // 5 Mbps (+25%)
  "480p": 2500000,    // 2.5 Mbps (+25%)
  "360p": 1200000     // 1.2 Mbps (+20%)
}
```

### Optimization Algorithm

```
1. Analyze content complexity (SI/TI, motion, scene changes)
2. Calculate base multiplier based on complexity level:
   - Very High: 1.4x
   - High: 1.2x
   - Medium: 1.0x
   - Low: 0.7x
3. Adjust for motion intensity:
   - High motion (>0.7): +15%
   - Low motion (<0.3): -15%
4. Adjust for spatial complexity:
   - High SI (>70): +10%
   - Low SI (<30): -10%
5. Adjust for content type:
   - Sports/Gaming: +15%
   - Animation: -15%
   - Presentation: -25%
6. Apply quality preference:
   - Prefer quality: +10%
7. Clamp to bounds: 0.5x to 1.8x of standard bitrate
```

---

## API Documentation

### Quality Analysis Endpoints

#### Analyze Video Quality
```http
POST /api/v1/videos/:id/analyze
```

Performs comprehensive quality and complexity analysis on a video.

**Response:**
```json
{
  "video_id": "uuid",
  "complexity": {
    "overall_complexity": "high",
    "complexity_score": 0.75,
    "avg_spatial_info": 65.2,
    "avg_temporal_info": 28.4,
    "avg_motion_intensity": 0.68,
    "scene_changes": 45,
    "content_category": "sports",
    "has_fast_motion": true,
    "sample_points": 30
  },
  "message": "video quality analysis completed"
}
```

#### Get Complexity Analysis
```http
GET /api/v1/videos/:id/complexity
```

Retrieves cached complexity analysis for a video.

#### Get Quality Analyses
```http
GET /api/v1/videos/:id/quality-analysis
```

Retrieves all VMAF and quality analyses for a video.

**Response:**
```json
{
  "video_id": "uuid",
  "analyses": [
    {
      "id": "uuid",
      "analysis_type": "vmaf",
      "vmaf_score": 94.5,
      "vmaf_min": 89.2,
      "vmaf_max": 98.1,
      "test_bitrate": 5000000,
      "test_resolution": "1080p",
      "test_codec": "libx264",
      "analyzed_at": "2025-01-17T10:00:00Z"
    }
  ],
  "count": 1
}
```

### Encoding Profile Endpoints

#### Generate Encoding Profile
```http
POST /api/v1/videos/:id/encoding-profile
```

Generates an optimized encoding profile for a video.

**Request:**
```json
{
  "preset_name": "high_quality"
}
```

**Response:**
```json
{
  "video_id": "uuid",
  "profile": {
    "id": "profile-uuid",
    "profile_name": "high_quality",
    "complexity_level": "high",
    "content_type": "sports",
    "bitrate_ladder": [
      {
        "resolution": "1080p",
        "bitrate": 10000000,
        "target_vmaf": 95
      },
      {
        "resolution": "720p",
        "bitrate": 5000000,
        "target_vmaf": 95
      }
    ],
    "codec_recommendation": "libx265",
    "preset_recommendation": "medium",
    "target_vmaf_score": 95.0,
    "estimated_size_reduction": 0.0,
    "confidence_score": 0.85
  },
  "message": "encoding profile generated successfully"
}
```

#### Get Encoding Profiles
```http
GET /api/v1/videos/:id/encoding-profiles
```

Lists all encoding profiles for a video.

#### Get Recommended Profile
```http
GET /api/v1/videos/:id/recommended-profile
```

Retrieves the best encoding profile for a video based on confidence and efficiency.

#### Transcode with Profile
```http
POST /api/v1/videos/:id/transcode-with-profile
```

Creates transcoding jobs using an optimized encoding profile.

**Request:**
```json
{
  "profile_id": "profile-uuid",
  "priority": 5
}
```

**Response:**
```json
{
  "video_id": "uuid",
  "profile_id": "profile-uuid",
  "jobs": [
    {
      "id": "job1-uuid",
      "video_id": "uuid",
      "status": "pending",
      "config": {
        "resolution": "1080p",
        "codec": "libx265",
        "bitrate": 10000000,
        "preset": "medium"
      },
      "target_vmaf": 95.0
    }
  ],
  "jobs_count": 2,
  "message": "transcoding jobs created with optimized profile"
}
```

### Quality Presets

#### Get Quality Presets
```http
GET /api/v1/quality-presets
```

Lists available quality presets.

**Response:**
```json
{
  "presets": [
    {
      "id": "uuid",
      "name": "high_quality",
      "description": "High quality encoding - VMAF 95+",
      "target_vmaf": 95.0,
      "min_vmaf": 93.0,
      "prefer_quality": true,
      "standard_ladder": [...]
    },
    {
      "name": "standard_quality",
      "target_vmaf": 87.0,
      "min_vmaf": 82.0
    },
    {
      "name": "bandwidth_optimized",
      "target_vmaf": 78.0,
      "min_vmaf": 72.0
    }
  ],
  "count": 3
}
```

### Experiments

#### Create Bitrate Experiment
```http
POST /api/v1/videos/:id/experiments
```

Creates an A/B test for different bitrate configurations.

**Request:**
```json
{
  "name": "Low bitrate test",
  "ladder_config": [
    {
      "resolution": "1080p",
      "bitrate": 4000000
    },
    {
      "resolution": "720p",
      "bitrate": 2000000
    }
  ]
}
```

#### Get Experiment
```http
GET /api/v1/experiments/:id
```

Retrieves experiment results.

**Response:**
```json
{
  "id": "exp-uuid",
  "video_id": "uuid",
  "experiment_name": "Low bitrate test",
  "status": "completed",
  "avg_vmaf_score": 92.5,
  "min_vmaf_score": 88.3,
  "total_size": 450000000,
  "encoding_time": 125.5,
  "size_vs_baseline": -25.0,
  "completed_at": "2025-01-17T10:15:00Z"
}
```

### Comparisons

#### Compare Encodings
```http
POST /api/v1/videos/:id/compare
```

Compares two encodings using VMAF.

**Request:**
```json
{
  "output1_id": "output1-uuid",
  "output2_id": "output2-uuid"
}
```

**Response:**
```json
{
  "video_id": "uuid",
  "comparison": {
    "output1": {
      "output_id": "output1-uuid",
      "vmaf": 95.2,
      "size": 500000000,
      "bitrate": 8000000,
      "efficiency": 84033.6
    },
    "output2": {
      "output_id": "output2-uuid",
      "vmaf": 94.8,
      "size": 350000000,
      "bitrate": 5600000,
      "efficiency": 59071.7
    },
    "winner": "output2"
  }
}
```

---

## Usage Examples

### Example 1: Analyze and Optimize

```bash
# 1. Upload video
curl -X POST http://localhost:8080/api/v1/videos/upload \
  -F "video=@movie.mp4"
# Response: {"id": "video-123", ...}

# 2. Analyze complexity
curl -X POST http://localhost:8080/api/v1/videos/video-123/analyze
# Response: complexity analysis with SI/TI metrics

# 3. Generate optimized profile
curl -X POST http://localhost:8080/api/v1/videos/video-123/encoding-profile \
  -H "Content-Type: application/json" \
  -d '{"preset_name": "high_quality"}'
# Response: optimized bitrate ladder

# 4. Transcode with optimized profile
curl -X POST http://localhost:8080/api/v1/videos/video-123/transcode-with-profile \
  -H "Content-Type: application/json" \
  -d '{"priority": 5}'
# Response: jobs created for each resolution in ladder
```

### Example 2: A/B Test Different Bitrates

```bash
# Create experiment with custom ladder
curl -X POST http://localhost:8080/api/v1/videos/video-123/experiments \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Aggressive compression test",
    "ladder_config": [
      {"resolution": "1080p", "bitrate": 4000000},
      {"resolution": "720p", "bitrate": 2000000}
    ]
  }'
# Response: {"experiment": {"id": "exp-456", "status": "running"}, ...}

# Check experiment status
curl http://localhost:8080/api/v1/experiments/exp-456
# Response: VMAF scores, file sizes, comparison to baseline
```

### Example 3: Compare Two Encodings

```bash
# Compare two different outputs
curl -X POST http://localhost:8080/api/v1/videos/video-123/compare \
  -H "Content-Type: application/json" \
  -d '{
    "output1_id": "out-789",
    "output2_id": "out-790"
  }'
# Response: VMAF scores, sizes, efficiency metrics, winner
```

---

## Performance Optimizations

### 1. Quick VMAF Analysis

For faster analysis, use subsampling (analyze every Nth frame):

```go
// Analyze every 4th frame (75% faster)
vmafResult, err := vmafAnalyzer.AnalyzeVMAFQuick(ctx, reference, distorted)
```

### 2. Segment Analysis

Analyze representative segments instead of full video:

```go
// Analyze 10-second segment at 30s mark
vmafResult, err := vmafAnalyzer.AnalyzeSegment(ctx, reference, distorted, 30.0, 10.0)
```

### 3. Caching

Complexity analysis results are cached in the database to avoid re-analysis.

---

## Configuration

### FFmpeg with VMAF Support

Ensure FFmpeg is compiled with libvmaf:

```bash
ffmpeg -filters | grep vmaf
# Should show: libvmaf filter
```

If not available, FFmpeg must be recompiled with `--enable-libvmaf`.

### Quality Presets

Three built-in presets are available:

| Preset                | Target VMAF | Use Case                           |
|-----------------------|-------------|------------------------------------|
| high_quality          | 95+         | Premium content, archival          |
| standard_quality      | 85+         | General streaming, balanced        |
| bandwidth_optimized   | 75+         | Mobile, low bandwidth              |

---

## Best Practices

### 1. Analyze Before Optimizing

Always run complexity analysis before generating encoding profiles:

```bash
POST /api/v1/videos/:id/analyze  # First
POST /api/v1/videos/:id/encoding-profile  # Then
```

### 2. Use Appropriate Presets

- **high_quality**: Important content, higher bitrates acceptable
- **standard_quality**: Most content, balanced approach
- **bandwidth_optimized**: Mobile delivery, minimize file size

### 3. A/B Test Your Ladders

Run experiments to validate optimizations:

1. Create experiment with optimized ladder
2. Compare VMAF scores to standard ladder
3. Verify file size reduction meets targets
4. Check minimum VMAF is above threshold

### 4. Monitor VMAF Scores

- Target VMAF ≥ 90 for high quality
- Never allow VMAF < 75 (noticeable degradation)
- Use harmonic mean for worst-case quality assessment

### 5. Content-Specific Tuning

- **Animation/Graphics**: Aggressive compression (0.7-0.85x bitrate)
- **Sports/Action**: Conservative compression (1.1-1.4x bitrate)
- **Mixed Content**: Use recommended profile (automatic adjustment)

---

## Troubleshooting

### FFmpeg VMAF Not Available

**Error**: `FFmpeg does not have VMAF support`

**Solution**: Compile FFmpeg with libvmaf:
```bash
# Install libvmaf
apt-get install libvmaf-dev

# Recompile FFmpeg
./configure --enable-libvmaf
make && make install
```

### Analysis Takes Too Long

**Issue**: Full VMAF analysis is slow

**Solutions**:
1. Use quick analysis (subsampling)
2. Analyze representative segments
3. Enable GPU acceleration (if available)

### Inaccurate Complexity Detection

**Issue**: Content misclassified

**Solutions**:
1. Increase sample points (longer videos need more samples)
2. Verify video quality (corrupted videos produce bad metrics)
3. Check for interlaced content (deinterlace first)

---

## Performance Benchmarks

### Analysis Performance

| Operation                | 1080p 60s Video | Notes                    |
|--------------------------|-----------------|--------------------------|
| Complexity Analysis      | 15-30s          | SI/TI, motion, scenes    |
| VMAF Full Analysis       | 60-120s         | Compare 2 videos         |
| VMAF Quick (4x subsample)| 15-30s          | 75% faster              |
| Segment Analysis (10s)   | 5-10s           | Representative sampling  |

### Storage Savings

| Content Type   | Typical Savings | Quality Impact |
|----------------|-----------------|----------------|
| Animation      | 20-35%          | Imperceptible  |
| Presentation   | 25-40%          | None           |
| Sports         | 0-10%           | Maintained     |
| Movies         | 10-20%          | Minimal        |
| Mixed          | 15-25%          | Low            |

---

## Database Schema Details

### quality_analysis
Stores VMAF and quality metrics for video analysis.

**Key Fields:**
- `vmaf_score`: Overall VMAF score (0-100)
- `vmaf_harmonic_mean`: Worst-case quality indicator
- `spatial_complexity`: SI metric
- `temporal_complexity`: TI metric
- `test_bitrate`: Bitrate used for test encoding

### encoding_profiles
Optimized encoding configurations per video.

**Key Fields:**
- `bitrate_ladder`: JSON array of resolution/bitrate pairs
- `codec_recommendation`: Suggested codec (libx264, libx265)
- `complexity_level`: low/medium/high/very_high
- `estimated_size_reduction`: % savings vs standard ladder
- `confidence_score`: 0-1 confidence in optimization

### content_complexity
Cached complexity analysis results.

**Key Fields:**
- `overall_complexity`: Classified complexity level
- `complexity_score`: Normalized 0-1 score
- `avg_spatial_info`: Mean SI across video
- `avg_temporal_info`: Mean TI across video
- `content_category`: Detected content type

---

## Future Enhancements

### Planned for Phase 5.1

- [ ] **ML-Based Optimization**: Train models on encoding history
- [ ] **Scene-Based Encoding**: Variable bitrate per scene
- [ ] **Perceptual Optimization**: Allocate bits based on visual importance
- [ ] **Real-Time VMAF**: Live quality monitoring during encoding
- [ ] **Multi-Codec Comparison**: Automatic codec selection (H.264 vs H.265 vs AV1)
- [ ] **Cost Optimization**: Balance quality vs encoding cost
- [ ] **CDN Integration**: Optimize for delivery network characteristics

---

## References

### VMAF
- [Netflix VMAF GitHub](https://github.com/Netflix/vmaf)
- [Netflix Tech Blog: Per-Title Encoding](https://netflixtechblog.com/per-title-encode-optimization-7e99442b62a2)
- [VMAF Documentation](https://github.com/Netflix/vmaf/blob/master/resource/doc/VMAF_DOCS.md)

### Video Quality Metrics
- [ITU-T BT.500: Subjective Video Quality Assessment](https://www.itu.int/rec/R-REC-BT.500)
- [Spatial Information & Temporal Information](https://ieeexplore.ieee.org/document/5535017)

### Encoding Optimization
- [YouTube Engineering: Dynamic Optimizer](https://youtube-eng.googleblog.com/)
- [Per-Title Encoding: The Next Evolution](https://medium.com/netflix-techblog)

---

**Documentation Version:** 1.0
**Last Updated:** 2025-01-17
**Maintained By:** Video Transcoding Pipeline Team
