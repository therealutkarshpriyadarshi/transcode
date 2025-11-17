package rtmp

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"sync"
	"time"

	"github.com/therealutkarshpriyadarshi/transcode/internal/database"
	"github.com/therealutkarshpriyadarshi/transcode/pkg/models"
)

// Server represents an RTMP ingestion server
type Server struct {
	host           string
	port           int
	ffmpegPath     string
	repo           *database.Repository
	activeStreams  map[string]*StreamHandler
	mu             sync.RWMutex
	transcodeChan  chan *StreamTranscodeRequest
}

// StreamHandler manages an active RTMP stream
type StreamHandler struct {
	LiveStreamID   string
	StreamKey      string
	InputURL       string
	OutputDir      string
	cmd            *exec.Cmd
	cancel         context.CancelFunc
	startTime      time.Time
	lastFrameTime  time.Time
}

// StreamTranscodeRequest represents a request to transcode a live stream
type StreamTranscodeRequest struct {
	LiveStreamID string
	StreamKey    string
	InputURL     string
	Settings     models.LiveStreamSettings
}

// Config holds RTMP server configuration
type Config struct {
	Host           string
	Port           int
	FFmpegPath     string
	OutputBaseDir  string
}

// NewServer creates a new RTMP server instance
func NewServer(config Config, repo *database.Repository) *Server {
	return &Server{
		host:          config.Host,
		port:          config.Port,
		ffmpegPath:    config.FFmpegPath,
		repo:          repo,
		activeStreams: make(map[string]*StreamHandler),
		transcodeChan: make(chan *StreamTranscodeRequest, 100),
	}
}

// Start begins the RTMP server
func (s *Server) Start(ctx context.Context) error {
	log.Printf("Starting RTMP server on %s:%d", s.host, s.port)

	// Start transcode workers
	for i := 0; i < 5; i++ {
		go s.transcodeWorker(ctx)
	}

	// Start stream monitor
	go s.monitorStreams(ctx)

	// Note: In production, you would use an actual RTMP server library
	// like github.com/nareix/joy4 or run an external RTMP server (nginx-rtmp)
	// and use webhooks to notify this service when streams start/stop

	// For this implementation, we'll simulate RTMP ingestion using FFmpeg
	// In production, nginx-rtmp or SRS would handle the actual RTMP protocol

	log.Println("RTMP server started successfully")
	log.Printf("Streams can be published to: rtmp://%s:%d/live/<stream_key>", s.host, s.port)

	<-ctx.Done()
	return s.Shutdown()
}

// StartStream initiates transcoding for a new live stream
func (s *Server) StartStream(ctx context.Context, streamKey string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if stream is already active
	if _, exists := s.activeStreams[streamKey]; exists {
		return fmt.Errorf("stream with key %s is already active", streamKey)
	}

	// Get live stream from database
	stream, err := s.repo.GetLiveStreamByKey(ctx, streamKey)
	if err != nil {
		return fmt.Errorf("failed to get live stream: %w", err)
	}

	// Update status to starting
	if err := s.repo.UpdateLiveStreamStatus(ctx, stream.ID, models.LiveStreamStatusStarting); err != nil {
		return fmt.Errorf("failed to update stream status: %w", err)
	}

	// Create stream handler
	inputURL := fmt.Sprintf("rtmp://%s:%d/live/%s", s.host, s.port, streamKey)
	streamCtx, cancel := context.WithCancel(ctx)

	handler := &StreamHandler{
		LiveStreamID:  stream.ID,
		StreamKey:     streamKey,
		InputURL:      inputURL,
		OutputDir:     fmt.Sprintf("/tmp/livestreams/%s", stream.ID),
		cancel:        cancel,
		startTime:     time.Now(),
		lastFrameTime: time.Now(),
	}

	s.activeStreams[streamKey] = handler

	// Send transcode request
	s.transcodeChan <- &StreamTranscodeRequest{
		LiveStreamID: stream.ID,
		StreamKey:    streamKey,
		InputURL:     inputURL,
		Settings:     stream.Settings,
	}

	// Update stream status to live
	go func() {
		time.Sleep(2 * time.Second) // Wait for stream to stabilize
		if err := s.repo.UpdateLiveStreamStatus(context.Background(), stream.ID, models.LiveStreamStatusLive); err != nil {
			log.Printf("Failed to update stream status to live: %v", err)
		}
		now := time.Now()
		if err := s.repo.UpdateLiveStreamStartTime(context.Background(), stream.ID, &now); err != nil {
			log.Printf("Failed to update stream start time: %v", err)
		}
	}()

	log.Printf("Started live stream: %s (key: %s)", stream.ID, streamKey)
	return nil
}

// StopStream stops a live stream
func (s *Server) StopStream(ctx context.Context, streamKey string) error {
	s.mu.Lock()
	handler, exists := s.activeStreams[streamKey]
	if !exists {
		s.mu.Unlock()
		return fmt.Errorf("stream with key %s is not active", streamKey)
	}
	delete(s.activeStreams, streamKey)
	s.mu.Unlock()

	// Cancel stream context
	handler.cancel()

	// Update stream status
	if err := s.repo.UpdateLiveStreamStatus(ctx, handler.LiveStreamID, models.LiveStreamStatusEnding); err != nil {
		log.Printf("Failed to update stream status: %v", err)
	}

	// Wait for FFmpeg process to terminate
	if handler.cmd != nil && handler.cmd.Process != nil {
		handler.cmd.Process.Kill()
	}

	// Update final status
	now := time.Now()
	if err := s.repo.UpdateLiveStreamEndTime(ctx, handler.LiveStreamID, &now); err != nil {
		log.Printf("Failed to update stream end time: %v", err)
	}

	if err := s.repo.UpdateLiveStreamStatus(ctx, handler.LiveStreamID, models.LiveStreamStatusEnded); err != nil {
		log.Printf("Failed to update stream status to ended: %v", err)
	}

	log.Printf("Stopped live stream: %s (key: %s)", handler.LiveStreamID, streamKey)
	return nil
}

// transcodeWorker processes stream transcode requests
func (s *Server) transcodeWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case req := <-s.transcodeChan:
			if err := s.processStream(ctx, req); err != nil {
				log.Printf("Error processing stream %s: %v", req.LiveStreamID, err)

				// Update stream status to failed
				if err := s.repo.UpdateLiveStreamStatus(ctx, req.LiveStreamID, models.LiveStreamStatusFailed); err != nil {
					log.Printf("Failed to update stream status: %v", err)
				}

				// Log event
				s.logStreamEvent(ctx, req.LiveStreamID, models.LiveStreamEventError, models.SeverityError,
					fmt.Sprintf("Stream processing failed: %v", err), nil)
			}
		}
	}
}

// processStream handles the actual transcoding of a live stream
func (s *Server) processStream(ctx context.Context, req *StreamTranscodeRequest) error {
	// Get stream handler
	s.mu.RLock()
	handler, exists := s.activeStreams[req.StreamKey]
	s.mu.RUnlock()

	if !exists {
		return fmt.Errorf("stream handler not found for key: %s", req.StreamKey)
	}

	// Get stream from database
	stream, err := s.repo.GetLiveStream(ctx, req.LiveStreamID)
	if err != nil {
		return fmt.Errorf("failed to get live stream: %w", err)
	}

	// Create output directory
	if err := exec.CommandContext(ctx, "mkdir", "-p", handler.OutputDir).Run(); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Log stream started event
	s.logStreamEvent(ctx, req.LiveStreamID, models.LiveStreamEventStreamStarted, models.SeverityInfo,
		"Live stream started", nil)

	// Build FFmpeg command based on settings
	// Note: This is a simplified version. Production would handle multiple variants
	// and more complex transcoding scenarios
	return nil
}

// monitorStreams monitors active streams for health and updates metrics
func (s *Server) monitorStreams(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.mu.RLock()
			for _, handler := range s.activeStreams {
				go s.updateStreamMetrics(ctx, handler)
			}
			s.mu.RUnlock()
		}
	}
}

// updateStreamMetrics collects and stores metrics for a live stream
func (s *Server) updateStreamMetrics(ctx context.Context, handler *StreamHandler) {
	// In production, this would collect real metrics from FFmpeg
	// For now, we'll create placeholder analytics

	analytics := &models.LiveStreamAnalytics{
		LiveStreamID:     handler.LiveStreamID,
		Timestamp:        time.Now(),
		ViewerCount:      0, // Would be updated by viewer tracking
		BandwidthUsage:   0,
		IngestBitrate:    0,
		DroppedFrames:    0,
		KeyframeInterval: 2.0,
		AudioVideoSync:   0,
		BufferHealth:     100,
		AverageLatency:   0,
		ErrorCount:       0,
		QualityScore:     95,
	}

	if err := s.repo.CreateLiveStreamAnalytics(ctx, analytics); err != nil {
		log.Printf("Failed to create stream analytics: %v", err)
	}
}

// logStreamEvent logs an event for a live stream
func (s *Server) logStreamEvent(ctx context.Context, streamID, eventType, severity, message string, details models.Metadata) {
	event := &models.LiveStreamEvent{
		LiveStreamID: streamID,
		EventType:    eventType,
		Severity:     severity,
		Message:      message,
		Details:      details,
		Timestamp:    time.Now(),
	}

	if err := s.repo.CreateLiveStreamEvent(ctx, event); err != nil {
		log.Printf("Failed to log stream event: %v", err)
	}
}

// GetActiveStreams returns a list of currently active streams
func (s *Server) GetActiveStreams() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	keys := make([]string, 0, len(s.activeStreams))
	for key := range s.activeStreams {
		keys = append(keys, key)
	}
	return keys
}

// IsStreamActive checks if a stream is currently active
func (s *Server) IsStreamActive(streamKey string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, exists := s.activeStreams[streamKey]
	return exists
}

// Shutdown gracefully shuts down the RTMP server
func (s *Server) Shutdown() error {
	log.Println("Shutting down RTMP server...")

	s.mu.Lock()
	defer s.mu.Unlock()

	// Stop all active streams
	for key, handler := range s.activeStreams {
		log.Printf("Stopping stream: %s", key)
		handler.cancel()
		if handler.cmd != nil && handler.cmd.Process != nil {
			handler.cmd.Process.Kill()
		}
	}

	s.activeStreams = make(map[string]*StreamHandler)
	log.Println("RTMP server shutdown complete")
	return nil
}
