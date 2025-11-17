package main

import (
	"github.com/gin-gonic/gin"
	"github.com/therealutkarshpriyadarshi/transcode/internal/config"
	"github.com/therealutkarshpriyadarshi/transcode/internal/database"
	"github.com/therealutkarshpriyadarshi/transcode/internal/storage"
	"github.com/therealutkarshpriyadarshi/transcode/internal/transcoder"
)

// Phase 5 Integration Guide
//
// To integrate Phase 5 into the API server, follow these steps:
//
// 1. Add qualityService field to the API struct:
//    type API struct {
//        repo           *database.Repository
//        storage        *storage.Storage
//        queue          *queue.Queue
//        ffmpeg         *transcoder.FFmpeg
//        qualityService *transcoder.QualityService  // Add this field
//        ... (other fields)
//    }
//
// 2. In the main() or mainPhase3() function, initialize the quality service:
//    qualityService := transcoder.NewQualityService(cfg.Transcoder, stor, repo)
//
// 3. Add qualityService to the API struct initialization:
//    api := &API{
//        repo:           repo,
//        storage:        stor,
//        queue:          q,
//        ffmpeg:         ffmpeg,
//        qualityService: qualityService,  // Add this field
//        ... (other fields)
//    }
//
// 4. In setupRouter or similar function, register Phase 5 routes:
//    api.registerPhase5Routes(router)
//
// 5. Run database migrations:
//    docker cp migrations/004_phase5_quality_metrics.up.sql transcode-postgres:/004_phase5_quality_metrics.up.sql
//    docker exec transcode-postgres psql -U postgres -d transcode -f /004_phase5_quality_metrics.up.sql

// initializeQualityService initializes the quality service with all dependencies
func initializeQualityService(
	cfg config.TranscoderConfig,
	stor *storage.Storage,
	repo *database.Repository,
) *transcoder.QualityService {
	return transcoder.NewQualityService(cfg, stor, repo)
}

// setupPhase5Routes sets up Phase 5 routes on the router
func setupPhase5Routes(api *API, router *gin.Engine) {
	api.registerPhase5Routes(router)
}

// Example of complete integration (pseudo-code)
/*
func main() {
	// ... existing setup code ...

	// Initialize quality service
	qualityService := transcoder.NewQualityService(cfg.Transcoder, stor, repo)

	// Create API instance with quality service
	api := &API{
		repo:           repo,
		storage:        stor,
		queue:          q,
		ffmpeg:         ffmpeg,
		qualityService: qualityService,
		// ... other fields
	}

	// Setup router with Phase 5 routes
	router := gin.Default()
	// ... register existing routes ...
	api.registerPhase5Routes(router)

	// ... rest of server setup ...
}
*/
