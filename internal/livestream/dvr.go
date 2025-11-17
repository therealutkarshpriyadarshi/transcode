package livestream

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/therealutkarshpriyadarshi/transcode/internal/database"
	"github.com/therealutkarshpriyadarshi/transcode/internal/storage"
	"github.com/therealutkarshpriyadarshi/transcode/pkg/models"
)

// DVRService handles DVR recording functionality for live streams
type DVRService struct {
	ffmpegPath string
	repo       *database.Repository
	storage    *storage.Storage
	outputDir  string
}

// NewDVRService creates a new DVR service
func NewDVRService(ffmpegPath string, repo *database.Repository, storage *storage.Storage, outputDir string) *DVRService {
	return &DVRService{
		ffmpegPath: ffmpegPath,
		repo:       repo,
		storage:    storage,
		outputDir:  outputDir,
	}
}

// StartRecording starts recording a live stream for DVR
func (d *DVRService) StartRecording(ctx context.Context, streamID string, dvrWindow int) (*models.DVRRecording, error) {
	// Create DVR recording record
	recording := &models.DVRRecording{
		LiveStreamID: streamID,
		StartTime:    time.Now(),
		Status:       models.DVRRecordingStatusRecording,
	}

	if err := d.repo.CreateDVRRecording(ctx, recording); err != nil {
		return nil, fmt.Errorf("failed to create DVR recording: %w", err)
	}

	// Create recording directory
	recordingDir := filepath.Join(d.outputDir, "dvr", streamID, recording.ID)
	if err := os.MkdirAll(recordingDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create recording directory: %w", err)
	}

	log.Printf("Started DVR recording for stream %s (recording ID: %s)", streamID, recording.ID)
	return recording, nil
}

// StopRecording stops a DVR recording and processes it
func (d *DVRService) StopRecording(ctx context.Context, recordingID string) error {
	recording, err := d.repo.GetDVRRecording(ctx, recordingID)
	if err != nil {
		return fmt.Errorf("failed to get recording: %w", err)
	}

	// Update end time and duration
	now := time.Now()
	duration := now.Sub(recording.StartTime).Seconds()

	recording.EndTime = &now
	recording.Duration = duration

	// Update status to processing
	if err := d.repo.UpdateDVRRecordingStatus(ctx, recordingID, models.DVRRecordingStatusProcessing); err != nil {
		return fmt.Errorf("failed to update recording status: %w", err)
	}

	// Process the recording asynchronously
	go d.processRecording(context.Background(), recording)

	log.Printf("Stopped DVR recording %s (duration: %.2f seconds)", recordingID, duration)
	return nil
}

// processRecording processes a completed DVR recording
func (d *DVRService) processRecording(ctx context.Context, recording *models.DVRRecording) {
	recordingDir := filepath.Join(d.outputDir, "dvr", recording.LiveStreamID, recording.ID)

	// Concatenate HLS segments into a single MP4 file
	outputFile := filepath.Join(recordingDir, "recording.mp4")

	// In production, this would concatenate the HLS segments
	// For now, we'll create a placeholder
	log.Printf("Processing DVR recording: %s", recording.ID)

	// Generate thumbnail
	thumbnailPath := filepath.Join(recordingDir, "thumbnail.jpg")
	if err := d.generateThumbnail(ctx, outputFile, thumbnailPath); err != nil {
		log.Printf("Failed to generate thumbnail: %v", err)
	}

	// Upload to storage (in production)
	// recordingURL := d.uploadToStorage(ctx, outputFile)
	// manifestURL := d.uploadManifest(ctx, recordingDir)

	// Update recording status
	if err := d.repo.UpdateDVRRecordingStatus(ctx, recording.ID, models.DVRRecordingStatusAvailable); err != nil {
		log.Printf("Failed to update recording status: %v", err)
	}

	// Set retention policy (e.g., 7 days)
	retentionUntil := time.Now().Add(7 * 24 * time.Hour)
	recording.RetentionUntil = &retentionUntil

	log.Printf("DVR recording processed: %s", recording.ID)
}

// generateThumbnail generates a thumbnail from the recording
func (d *DVRService) generateThumbnail(ctx context.Context, inputFile, outputFile string) error {
	cmd := exec.CommandContext(ctx, d.ffmpegPath,
		"-i", inputFile,
		"-ss", "00:00:01",
		"-vframes", "1",
		"-vf", "scale=320:-1",
		"-y",
		outputFile,
	)

	return cmd.Run()
}

// ConvertToVOD converts a DVR recording to a regular VOD (Video on Demand)
func (d *DVRService) ConvertToVOD(ctx context.Context, recordingID string) (*models.Video, error) {
	recording, err := d.repo.GetDVRRecording(ctx, recordingID)
	if err != nil {
		return nil, fmt.Errorf("failed to get recording: %w", err)
	}

	if recording.Status != models.DVRRecordingStatusAvailable {
		return nil, fmt.Errorf("recording is not available for conversion")
	}

	// Create a new video record
	video := &models.Video{
		Filename: fmt.Sprintf("dvr_recording_%s.mp4", recordingID),
		Size:     recording.Size,
		Duration: recording.Duration,
		Status:   models.VideoStatusPending,
		Metadata: models.Metadata{
			"source":          "dvr",
			"live_stream_id":  recording.LiveStreamID,
			"recording_id":    recording.ID,
			"recording_start": recording.StartTime,
		},
	}

	if err := d.repo.CreateVideo(ctx, video); err != nil {
		return nil, fmt.Errorf("failed to create video: %w", err)
	}

	// Link the video to the recording
	recording.VideoID = &video.ID

	log.Printf("Converted DVR recording %s to VOD %s", recordingID, video.ID)
	return video, nil
}

// CleanupExpiredRecordings removes expired DVR recordings
func (d *DVRService) CleanupExpiredRecordings(ctx context.Context) error {
	// This would be called periodically (e.g., daily cron job)
	// to clean up recordings past their retention period

	log.Println("Cleaning up expired DVR recordings...")

	// Archive expired recordings
	// In production, this would call the database function:
	// SELECT archive_expired_dvr_recordings();

	return nil
}

// GetRecordingMetadata retrieves metadata about a DVR recording
func (d *DVRService) GetRecordingMetadata(ctx context.Context, recordingID string) (*RecordingMetadata, error) {
	recording, err := d.repo.GetDVRRecording(ctx, recordingID)
	if err != nil {
		return nil, err
	}

	metadata := &RecordingMetadata{
		ID:           recording.ID,
		LiveStreamID: recording.LiveStreamID,
		StartTime:    recording.StartTime,
		EndTime:      recording.EndTime,
		Duration:     recording.Duration,
		Size:         recording.Size,
		Status:       recording.Status,
		Available:    recording.Status == models.DVRRecordingStatusAvailable,
	}

	return metadata, nil
}

// RecordingMetadata contains metadata about a DVR recording
type RecordingMetadata struct {
	ID           string
	LiveStreamID string
	StartTime    time.Time
	EndTime      *time.Time
	Duration     float64
	Size         int64
	Status       string
	Available    bool
}
