package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/minio/minio-go/v7"
)

const (
	// Default part size for multipart uploads (10MB)
	DefaultPartSize = 10 * 1024 * 1024

	// Minimum part size for multipart uploads (5MB)
	MinPartSize = 5 * 1024 * 1024

	// Maximum number of concurrent parts
	MaxConcurrentParts = 10
)

// OptimizedStorage extends Storage with optimization features
type OptimizedStorage struct {
	*Storage
	partSize         int64
	maxConcurrentParts int
}

// NewOptimizedStorage creates a new optimized storage instance
func NewOptimizedStorage(storage *Storage, partSize int64) *OptimizedStorage {
	if partSize < MinPartSize {
		partSize = DefaultPartSize
	}

	return &OptimizedStorage{
		Storage:           storage,
		partSize:          partSize,
		maxConcurrentParts: MaxConcurrentParts,
	}
}

// UploadFileParallel uploads a file using parallel multipart upload
func (s *OptimizedStorage) UploadFileParallel(ctx context.Context, key, filePath string) error {
	// Get file info
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	fileSize := fileInfo.Size()

	// For small files, use standard upload
	if fileSize < s.partSize {
		return s.UploadFile(ctx, key, filePath)
	}

	// Open file
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Use MinIO's optimized PutObject with multipart upload
	_, err = s.client.PutObject(
		ctx,
		s.bucketName,
		key,
		file,
		fileSize,
		minio.PutObjectOptions{
			PartSize:    uint64(s.partSize),
			ContentType: "application/octet-stream",
			NumThreads:  uint(s.maxConcurrentParts),
		},
	)

	if err != nil {
		return fmt.Errorf("failed to upload file: %w", err)
	}

	return nil
}

// UploadStreamParallel uploads a stream using parallel multipart upload
func (s *OptimizedStorage) UploadStreamParallel(ctx context.Context, key string, reader io.Reader, size int64) error {
	if size < s.partSize {
		// For small files, use standard upload
		_, err := s.client.PutObject(
			ctx,
			s.bucketName,
			key,
			reader,
			size,
			minio.PutObjectOptions{
				ContentType: "application/octet-stream",
			},
		)
		return err
	}

	// Use MinIO's optimized PutObject with multipart upload
	_, err := s.client.PutObject(
		ctx,
		s.bucketName,
		key,
		reader,
		size,
		minio.PutObjectOptions{
			PartSize:    uint64(s.partSize),
			ContentType: "application/octet-stream",
			NumThreads:  uint(s.maxConcurrentParts),
		},
	)

	if err != nil {
		return fmt.Errorf("failed to upload stream: %w", err)
	}

	return nil
}

// DownloadFileParallel downloads a file using parallel connections
func (s *OptimizedStorage) DownloadFileParallel(ctx context.Context, key, destPath string) error {
	// Get object info
	objInfo, err := s.client.StatObject(ctx, s.bucketName, key, minio.StatObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to stat object: %w", err)
	}

	// For small files, use standard download
	if objInfo.Size < s.partSize {
		return s.DownloadFile(ctx, key, destPath)
	}

	// Download using range requests for large files
	return s.downloadWithRanges(ctx, key, destPath, objInfo.Size)
}

// downloadWithRanges downloads a file using multiple range requests
func (s *OptimizedStorage) downloadWithRanges(ctx context.Context, key, destPath string, totalSize int64) error {
	// Create output file
	outFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	// Calculate number of parts
	numParts := (totalSize + s.partSize - 1) / s.partSize
	if numParts > int64(s.maxConcurrentParts) {
		numParts = int64(s.maxConcurrentParts)
	}

	partSize := totalSize / numParts

	// Download parts concurrently
	var wg sync.WaitGroup
	errChan := make(chan error, numParts)
	partChan := make(chan *partData, numParts)

	// Worker pool
	for i := int64(0); i < numParts; i++ {
		wg.Add(1)
		go func(partNum int64) {
			defer wg.Done()

			start := partNum * partSize
			end := start + partSize - 1
			if partNum == numParts-1 {
				end = totalSize - 1
			}

			data, err := s.downloadPart(ctx, key, start, end)
			if err != nil {
				errChan <- fmt.Errorf("failed to download part %d: %w", partNum, err)
				return
			}

			partChan <- &partData{
				partNum: partNum,
				data:    data,
			}
		}(i)
	}

	// Wait for all parts to complete
	go func() {
		wg.Wait()
		close(errChan)
		close(partChan)
	}()

	// Check for errors
	if err := <-errChan; err != nil {
		return err
	}

	// Collect and write parts in order
	parts := make(map[int64][]byte)
	for part := range partChan {
		parts[part.partNum] = part.data
	}

	// Write parts to file in order
	for i := int64(0); i < numParts; i++ {
		if data, ok := parts[i]; ok {
			if _, err := outFile.Write(data); err != nil {
				return fmt.Errorf("failed to write part %d: %w", i, err)
			}
		} else {
			return fmt.Errorf("missing part %d", i)
		}
	}

	return nil
}

// downloadPart downloads a single part using range request
func (s *OptimizedStorage) downloadPart(ctx context.Context, key string, start, end int64) ([]byte, error) {
	opts := minio.GetObjectOptions{}
	if err := opts.SetRange(start, end); err != nil {
		return nil, fmt.Errorf("failed to set range: %w", err)
	}

	object, err := s.client.GetObject(ctx, s.bucketName, key, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get object: %w", err)
	}
	defer object.Close()

	data := make([]byte, end-start+1)
	_, err = io.ReadFull(object, data)
	if err != nil {
		return nil, fmt.Errorf("failed to read object: %w", err)
	}

	return data, nil
}

// partData holds downloaded part data
type partData struct {
	partNum int64
	data    []byte
}

// CopyObject copies an object within the bucket (useful for deduplication)
func (s *OptimizedStorage) CopyObject(ctx context.Context, srcKey, destKey string) error {
	src := minio.CopySrcOptions{
		Bucket: s.bucketName,
		Object: srcKey,
	}

	dst := minio.CopyDestOptions{
		Bucket: s.bucketName,
		Object: destKey,
	}

	_, err := s.client.CopyObject(ctx, dst, src)
	if err != nil {
		return fmt.Errorf("failed to copy object: %w", err)
	}

	return nil
}

// GetObjectMetadata retrieves object metadata without downloading
func (s *OptimizedStorage) GetObjectMetadata(ctx context.Context, key string) (map[string]string, error) {
	objInfo, err := s.client.StatObject(ctx, s.bucketName, key, minio.StatObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to stat object: %w", err)
	}

	metadata := make(map[string]string)
	for k, v := range objInfo.UserMetadata {
		if len(v) > 0 {
			metadata[k] = v[0]
		}
	}

	metadata["content-type"] = objInfo.ContentType
	metadata["etag"] = objInfo.ETag
	metadata["size"] = fmt.Sprintf("%d", objInfo.Size)

	return metadata, nil
}

// SetObjectMetadata sets object metadata
func (s *OptimizedStorage) SetObjectMetadata(ctx context.Context, key string, metadata map[string]string) error {
	// MinIO requires copying the object to itself to update metadata
	src := minio.CopySrcOptions{
		Bucket: s.bucketName,
		Object: key,
	}

	dst := minio.CopyDestOptions{
		Bucket:          s.bucketName,
		Object:          key,
		UserMetadata:    metadata,
		ReplaceMetadata: true,
	}

	_, err := s.client.CopyObject(ctx, dst, src)
	if err != nil {
		return fmt.Errorf("failed to update metadata: %w", err)
	}

	return nil
}

// BatchDelete deletes multiple objects
func (s *OptimizedStorage) BatchDelete(ctx context.Context, keys []string) error {
	objectsCh := make(chan minio.ObjectInfo, len(keys))

	// Send object keys to channel
	go func() {
		defer close(objectsCh)
		for _, key := range keys {
			objectsCh <- minio.ObjectInfo{Key: key}
		}
	}()

	// Delete objects
	errorCh := s.client.RemoveObjects(ctx, s.bucketName, objectsCh, minio.RemoveObjectsOptions{})

	// Check for errors
	for err := range errorCh {
		if err.Err != nil {
			return fmt.Errorf("failed to delete object %s: %w", err.ObjectName, err.Err)
		}
	}

	return nil
}

// CalculateChecksum calculates checksum for a file
func (s *OptimizedStorage) CalculateChecksum(ctx context.Context, key string) (string, error) {
	objInfo, err := s.client.StatObject(ctx, s.bucketName, key, minio.StatObjectOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to stat object: %w", err)
	}

	return objInfo.ETag, nil
}

// CompareChecksums compares checksums of two objects
func (s *OptimizedStorage) CompareChecksums(ctx context.Context, key1, key2 string) (bool, error) {
	checksum1, err := s.CalculateChecksum(ctx, key1)
	if err != nil {
		return false, err
	}

	checksum2, err := s.CalculateChecksum(ctx, key2)
	if err != nil {
		return false, err
	}

	return checksum1 == checksum2, nil
}
