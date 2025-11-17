package upload

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
)

// MultipartUpload represents a multipart upload session
type MultipartUpload struct {
	ID          string             `json:"id"`
	Filename    string             `json:"filename"`
	TotalSize   int64              `json:"total_size"`
	PartSize    int64              `json:"part_size"`
	TotalParts  int                `json:"total_parts"`
	Parts       map[int]*UploadPart `json:"parts"`
	Status      string             `json:"status"`
	CreatedAt   time.Time          `json:"created_at"`
	ExpiresAt   time.Time          `json:"expires_at"`
	CompletedAt *time.Time         `json:"completed_at,omitempty"`
	mu          sync.RWMutex
}

// UploadPart represents a single part of a multipart upload
type UploadPart struct {
	PartNumber int       `json:"part_number"`
	Size       int64     `json:"size"`
	ETag       string    `json:"etag"`
	Uploaded   bool      `json:"uploaded"`
	UploadedAt time.Time `json:"uploaded_at,omitempty"`
}

// MultipartUploadService manages multipart uploads
type MultipartUploadService struct {
	uploads  map[string]*MultipartUpload
	mu       sync.RWMutex
	tempDir  string
	partSize int64
}

const (
	DefaultPartSize          = 5 * 1024 * 1024  // 5MB
	MaxPartSize              = 100 * 1024 * 1024 // 100MB
	DefaultUploadExpiration  = 24 * time.Hour
	MultipartUploadStatusActive   = "active"
	MultipartUploadStatusCompleted = "completed"
	MultipartUploadStatusAborted  = "aborted"
)

// NewMultipartUploadService creates a new multipart upload service
func NewMultipartUploadService(tempDir string, partSize int64) *MultipartUploadService {
	if partSize == 0 {
		partSize = DefaultPartSize
	}

	if partSize > MaxPartSize {
		partSize = MaxPartSize
	}

	return &MultipartUploadService{
		uploads:  make(map[string]*MultipartUpload),
		tempDir:  tempDir,
		partSize: partSize,
	}
}

// InitiateUpload starts a new multipart upload
func (s *MultipartUploadService) InitiateUpload(filename string, totalSize int64) (*MultipartUpload, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	uploadID := uuid.New().String()
	totalParts := int((totalSize + s.partSize - 1) / s.partSize)

	upload := &MultipartUpload{
		ID:         uploadID,
		Filename:   filename,
		TotalSize:  totalSize,
		PartSize:   s.partSize,
		TotalParts: totalParts,
		Parts:      make(map[int]*UploadPart),
		Status:     MultipartUploadStatusActive,
		CreatedAt:  time.Now(),
		ExpiresAt:  time.Now().Add(DefaultUploadExpiration),
	}

	// Create upload directory
	uploadDir := filepath.Join(s.tempDir, "uploads", uploadID)
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create upload directory: %w", err)
	}

	s.uploads[uploadID] = upload

	log.Printf("Initiated multipart upload %s for %s (%d bytes, %d parts)",
		uploadID, filename, totalSize, totalParts)

	return upload, nil
}

// UploadPart uploads a single part
func (s *MultipartUploadService) UploadPart(uploadID string, partNumber int, data io.Reader) (*UploadPart, error) {
	s.mu.RLock()
	upload, exists := s.uploads[uploadID]
	s.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("upload not found: %s", uploadID)
	}

	if upload.Status != MultipartUploadStatusActive {
		return nil, fmt.Errorf("upload is not active")
	}

	if time.Now().After(upload.ExpiresAt) {
		return nil, fmt.Errorf("upload has expired")
	}

	if partNumber < 1 || partNumber > upload.TotalParts {
		return nil, fmt.Errorf("invalid part number: %d", partNumber)
	}

	// Save part to disk
	uploadDir := filepath.Join(s.tempDir, "uploads", uploadID)
	partPath := filepath.Join(uploadDir, fmt.Sprintf("part_%d", partNumber))

	file, err := os.Create(partPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create part file: %w", err)
	}
	defer file.Close()

	// Calculate MD5 while writing
	hash := md5.New()
	writer := io.MultiWriter(file, hash)

	size, err := io.Copy(writer, data)
	if err != nil {
		return nil, fmt.Errorf("failed to write part: %w", err)
	}

	etag := hex.EncodeToString(hash.Sum(nil))

	part := &UploadPart{
		PartNumber: partNumber,
		Size:       size,
		ETag:       etag,
		Uploaded:   true,
		UploadedAt: time.Now(),
	}

	upload.mu.Lock()
	upload.Parts[partNumber] = part
	upload.mu.Unlock()

	log.Printf("Uploaded part %d/%d for upload %s (%d bytes, etag: %s)",
		partNumber, upload.TotalParts, uploadID, size, etag)

	return part, nil
}

// CompleteUpload finalizes the multipart upload
func (s *MultipartUploadService) CompleteUpload(uploadID string) (string, error) {
	s.mu.RLock()
	upload, exists := s.uploads[uploadID]
	s.mu.RUnlock()

	if !exists {
		return "", fmt.Errorf("upload not found: %s", uploadID)
	}

	upload.mu.Lock()
	defer upload.mu.Unlock()

	if upload.Status != MultipartUploadStatusActive {
		return "", fmt.Errorf("upload is not active")
	}

	// Verify all parts are uploaded
	for i := 1; i <= upload.TotalParts; i++ {
		if part, ok := upload.Parts[i]; !ok || !part.Uploaded {
			return "", fmt.Errorf("missing part %d", i)
		}
	}

	// Combine parts into final file
	uploadDir := filepath.Join(s.tempDir, "uploads", uploadID)
	finalPath := filepath.Join(uploadDir, upload.Filename)

	finalFile, err := os.Create(finalPath)
	if err != nil {
		return "", fmt.Errorf("failed to create final file: %w", err)
	}
	defer finalFile.Close()

	// Concatenate all parts
	for i := 1; i <= upload.TotalParts; i++ {
		partPath := filepath.Join(uploadDir, fmt.Sprintf("part_%d", i))
		partFile, err := os.Open(partPath)
		if err != nil {
			return "", fmt.Errorf("failed to open part %d: %w", i, err)
		}

		if _, err := io.Copy(finalFile, partFile); err != nil {
			partFile.Close()
			return "", fmt.Errorf("failed to copy part %d: %w", i, err)
		}

		partFile.Close()

		// Delete part file
		os.Remove(partPath)
	}

	upload.Status = MultipartUploadStatusCompleted
	now := time.Now()
	upload.CompletedAt = &now

	log.Printf("Completed multipart upload %s (%s)", uploadID, finalPath)

	return finalPath, nil
}

// AbortUpload cancels a multipart upload
func (s *MultipartUploadService) AbortUpload(uploadID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	upload, exists := s.uploads[uploadID]
	if !exists {
		return fmt.Errorf("upload not found: %s", uploadID)
	}

	upload.mu.Lock()
	upload.Status = MultipartUploadStatusAborted
	upload.mu.Unlock()

	// Clean up upload directory
	uploadDir := filepath.Join(s.tempDir, "uploads", uploadID)
	if err := os.RemoveAll(uploadDir); err != nil {
		log.Printf("Failed to remove upload directory %s: %v", uploadDir, err)
	}

	delete(s.uploads, uploadID)

	log.Printf("Aborted multipart upload %s", uploadID)

	return nil
}

// GetUpload retrieves upload status
func (s *MultipartUploadService) GetUpload(uploadID string) (*MultipartUpload, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	upload, exists := s.uploads[uploadID]
	if !exists {
		return nil, fmt.Errorf("upload not found: %s", uploadID)
	}

	return upload, nil
}

// ListParts lists all uploaded parts for an upload
func (s *MultipartUploadService) ListParts(uploadID string) ([]*UploadPart, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	upload, exists := s.uploads[uploadID]
	if !exists {
		return nil, fmt.Errorf("upload not found: %s", uploadID)
	}

	upload.mu.RLock()
	defer upload.mu.RUnlock()

	parts := make([]*UploadPart, 0, len(upload.Parts))
	for _, part := range upload.Parts {
		parts = append(parts, part)
	}

	return parts, nil
}

// CleanupExpired removes expired uploads
func (s *MultipartUploadService) CleanupExpired(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.cleanupExpiredUploads()
		}
	}
}

func (s *MultipartUploadService) cleanupExpiredUploads() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for uploadID, upload := range s.uploads {
		if now.After(upload.ExpiresAt) && upload.Status == MultipartUploadStatusActive {
			upload.mu.Lock()
			upload.Status = MultipartUploadStatusAborted
			upload.mu.Unlock()

			// Clean up upload directory
			uploadDir := filepath.Join(s.tempDir, "uploads", uploadID)
			if err := os.RemoveAll(uploadDir); err != nil {
				log.Printf("Failed to remove expired upload directory %s: %v", uploadDir, err)
			}

			delete(s.uploads, uploadID)
			log.Printf("Cleaned up expired upload %s", uploadID)
		}
	}
}
