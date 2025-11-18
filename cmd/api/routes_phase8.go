package main

import "github.com/gin-gonic/gin"

// setupPhase8Routes adds Phase 8 live streaming routes to the API
func setupPhase8Routes(router *gin.RouterGroup, api *API) {
	// Live Streams
	livestreams := router.Group("/livestreams")
	{
		livestreams.POST("", api.createLiveStream)          // Create a new live stream
		livestreams.GET("", api.listLiveStreams)            // List all live streams
		livestreams.GET("/:id", api.getLiveStream)          // Get live stream details
		livestreams.POST("/:id/start", api.startLiveStream) // Start a live stream
		livestreams.POST("/:id/stop", api.stopLiveStream)   // Stop a live stream
		livestreams.DELETE("/:id", api.deleteLiveStream)    // Delete a live stream

		// Variants
		livestreams.GET("/:id/variants", api.getLiveStreamVariants) // Get stream quality variants

		// Analytics
		livestreams.GET("/:id/analytics", api.getLiveStreamAnalytics) // Get stream analytics
		livestreams.GET("/:id/events", api.getLiveStreamEvents)       // Get stream events

		// DVR
		livestreams.GET("/:id/recordings", api.getDVRRecordings)              // List DVR recordings
		livestreams.GET("/:id/recordings/:recording_id", api.getDVRRecording) // Get specific recording

		// Viewers
		livestreams.GET("/:id/viewers", api.getActiveViewers)          // Get active viewers
		livestreams.POST("/:id/viewers/track", api.trackViewerSession) // Track viewer session
	}
}
