package service

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	sharedlog "github.com/aser/youtube-clone-platform/internal/shared/log"
	"github.com/aser/youtube-clone-platform/video-upload-service/internal/events"
	"github.com/aser/youtube-clone-platform/video-upload-service/internal/metadata"
	"github.com/aser/youtube-clone-platform/video-upload-service/internal/storage"
	"github.com/aser/youtube-clone-platform/video-upload-service/internal/validation"
)

type UploadService struct {
	storage   storage.Storage
	publisher events.Publisher
	maxBytes  int64
}

func NewUploadService(storage storage.Storage, publisher events.Publisher, maxBytes int64) *UploadService {
	return &UploadService{
		storage:   storage,
		publisher: publisher,
		maxBytes:  maxBytes,
	}
}

type UploadResult struct {
	VideoID  string
	Warning  string
	Metadata *metadata.VideoMetadata
}

func (s *UploadService) HandleUpload(ctx context.Context, userID string, title string, file io.Reader, size int64, contentType string, originalFilename string) (*UploadResult, error) {
	// Validate inputs
	if err := validation.ValidateTitle(title); err != nil {
		return nil, err
	}
	if err := validation.ValidateUserID(userID); err != nil {
		return nil, err
	}
	if size > s.maxBytes {
		return nil, fmt.Errorf("file too large: max size is %d bytes", s.maxBytes)
	}

	// Create temporary file for metadata
	tmpFile, err := os.CreateTemp("", "video-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// Create a pipe to stream into the upload service
	uploadReader, uploadWriter := io.Pipe()
	multiWriter := io.MultiWriter(tmpFile, uploadWriter)

	// Channels to collect results from both goroutines
	type uploadResult struct {
		videoID string
		err     error
	}
	type metadataResult struct {
		meta *metadata.VideoMetadata
		err  error
	}

	uploadCh := make(chan uploadResult, 1)
	metadataCh := make(chan metadataResult, 1)

	// Start upload in background
	go func() {
		videoID, err := s.storage.UploadVideo(ctx, uploadReader, size, contentType, originalFilename)
		uploadCh <- uploadResult{videoID, err}
	}()

	// Copy to tmpFile and upload stream
	go func() {
		_, err := io.Copy(multiWriter, file)
		uploadWriter.Close() // Signal upload end
		if err != nil {
			uploadReader.CloseWithError(err)
			metadataCh <- metadataResult{nil, fmt.Errorf("copy error: %w", err)}
			return
		}

		// Rewind temp file
		_, err = tmpFile.Seek(0, 0)
		if err != nil {
			metadataCh <- metadataResult{nil, fmt.Errorf("seek error: %w", err)}
			return
		}

		meta, mErr := metadata.ExtractMetadata(ctx, tmpFile.Name())
		metadataCh <- metadataResult{meta, mErr}
	}()

	// Wait for both to finish
	uResult := <-uploadCh
	mResult := <-metadataCh

	if uResult.err != nil {
		return nil, fmt.Errorf("failed to upload video: %w", uResult.err)
	}

	var meta *metadata.VideoMetadata
	var warning string

	if mResult.err != nil {
		sharedlog.Warn(fmt.Sprintf("Warning: failed to extract metadata: %v", mResult.err))
		warning = fmt.Sprintf("failed to extract metadata: %v", mResult.err)
	}

	if mResult.meta == nil {
		meta = &metadata.VideoMetadata{
			FileSize:  size,
			Format:    filepath.Ext(originalFilename),
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
		}
	} else {
		meta = mResult.meta
	}

	// Finalize metadata
	meta.OriginalFilename = originalFilename
	meta.FileExtension = filepath.Ext(originalFilename)
	meta.SanitizedFilename = metadata.SanitizeFilename(originalFilename)

	// Publish upload event
	err = s.publisher.PublishVideoUpload(ctx, events.VideoUploadEvent{
		VideoID:     uResult.videoID,
		UserID:      userID,
		Title:       title,
		ContentType: contentType,
		Size:        size,
		Metadata:    *meta,
		UploadedAt:  time.Now().UTC().Format(time.RFC3339),
	})

	result := &UploadResult{
		VideoID:  uResult.videoID,
		Metadata: meta,
	}

	if warning != "" {
		result.Warning = warning
	}
	if err != nil {
		if result.Warning != "" {
			result.Warning += "; "
		}
		result.Warning += fmt.Sprintf("video uploaded but event publishing failed: %v", err)
	}

	return result, nil
}
