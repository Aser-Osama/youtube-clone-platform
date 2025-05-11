package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"youtube-clone-platform/transcoder-service/internal/events"
	"youtube-clone-platform/transcoder-service/internal/storage"
	"youtube-clone-platform/transcoder-service/internal/transcoder"
)

// TranscoderService handles video transcoding operations
type TranscoderService struct {
	storage       storage.Storage
	transcoder    transcoder.Transcoder
	consumer      events.Consumer
	producer      events.Producer
	maxJobs       int
	jobTimeout    time.Duration
	tempDir       string
	activeJobs    map[string]context.CancelFunc
	activeJobsMux sync.Mutex
}

// NewTranscoderService creates a new TranscoderService instance
func NewTranscoderService(
	storage storage.Storage,
	transcoder transcoder.Transcoder,
	consumer events.Consumer,
	producer events.Producer,
	maxJobs int,
	jobTimeout time.Duration,
	tempDir string,
) *TranscoderService {
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		panic(fmt.Sprintf("failed to create temp directory: %v", err))
	}

	return &TranscoderService{
		storage:    storage,
		transcoder: transcoder,
		consumer:   consumer,
		producer:   producer,
		maxJobs:    maxJobs,
		jobTimeout: jobTimeout,
		tempDir:    tempDir,
		activeJobs: make(map[string]context.CancelFunc),
	}
}

// Start starts the transcoder service
func (s *TranscoderService) Start(ctx context.Context) error {
	return s.consumer.Start(ctx, func(ctx context.Context, event events.VideoUploadEvent) error {
		return s.handleVideoUpload(ctx, &event)
	})
}

// Stop stops the transcoder service
func (s *TranscoderService) Stop() {
	s.activeJobsMux.Lock()
	defer s.activeJobsMux.Unlock()

	for _, cancel := range s.activeJobs {
		cancel()
	}

	s.consumer.Close()
	s.producer.Close()
}

// handleVideoUpload handles a video upload event
func (s *TranscoderService) handleVideoUpload(ctx context.Context, event *events.VideoUploadEvent) error {
	s.activeJobsMux.Lock()
	defer s.activeJobsMux.Unlock()

	// Check if job already exists
	if _, exists := s.activeJobs[event.VideoID]; exists {
		return fmt.Errorf("job already exists for video %s", event.VideoID)
	}

	// Check if we've reached the maximum number of jobs
	if len(s.activeJobs) >= s.maxJobs {
		return fmt.Errorf("maximum number of concurrent jobs reached")
	}

	// Create a new context with timeout
	jobCtx, cancel := context.WithTimeout(ctx, s.jobTimeout)
	s.activeJobs[event.VideoID] = cancel

	// Start processing in a goroutine
	go func() {
		defer func() {
			s.activeJobsMux.Lock()
			delete(s.activeJobs, event.VideoID)
			s.activeJobsMux.Unlock()
		}()

		if err := s.processVideo(jobCtx, event); err != nil {
			fmt.Printf("Failed to process video %s: %v\n", event.VideoID, err)
		}
	}()

	return nil
}

// processVideo processes a video
func (s *TranscoderService) processVideo(ctx context.Context, event *events.VideoUploadEvent) error {
	// Create temporary directory for the video
	videoDir := filepath.Join(s.tempDir, event.VideoID)
	if err := os.MkdirAll(videoDir, 0755); err != nil {
		return fmt.Errorf("failed to create video directory: %w", err)
	}
	defer os.RemoveAll(videoDir)

	// Determine file extension from content type
	fileExtension := ".mp4" // Default
	if event.ContentType == "video/webm" {
		fileExtension = ".webm"
	} else if event.ContentType == "video/quicktime" {
		fileExtension = ".mov"
	}

	// Download video from MinIO
	videoPath := filepath.Join(videoDir, "original"+fileExtension)
	if err := s.storage.DownloadVideo(ctx, event.VideoID, fileExtension, videoPath); err != nil {
		return fmt.Errorf("failed to download video: %w", err)
	}

	// Get video dimensions
	width := event.Metadata.Width
	height := event.Metadata.Height

	// Create output directories
	hlsDir := filepath.Join(videoDir, "hls")
	mp4Dir := filepath.Join(videoDir, "mp4")
	if err := os.MkdirAll(hlsDir, 0755); err != nil {
		return fmt.Errorf("failed to create HLS directory: %w", err)
	}
	if err := os.MkdirAll(mp4Dir, 0755); err != nil {
		return fmt.Errorf("failed to create MP4 directory: %w", err)
	}

	// Transcode to HLS
	if err := s.transcoder.TranscodeToHLS(ctx, videoPath, hlsDir, width, height); err != nil {
		return fmt.Errorf("failed to transcode to HLS: %w", err)
	}

	// Transcode to MP4
	if err := s.transcoder.TranscodeToMP4(ctx, videoPath, mp4Dir, width, height); err != nil {
		return fmt.Errorf("failed to transcode to MP4: %w", err)
	}

	// Generate thumbnail
	localThumbnailPath := filepath.Join(videoDir, "thumbnail.jpg")
	if err := s.transcoder.GenerateThumbnail(ctx, videoPath, localThumbnailPath); err != nil {
		return fmt.Errorf("failed to generate thumbnail: %w", err)
	}

	// Upload HLS files
	hlsPath, err := s.storage.UploadHLSFiles(ctx, event.VideoID, hlsDir)
	if err != nil {
		return fmt.Errorf("failed to upload HLS files: %w", err)
	}

	// Upload MP4 files
	mp4Path := filepath.Join(s.storage.GetMP4Prefix(), event.VideoID)
	if err := s.storage.UploadMP4Files(ctx, event.VideoID, mp4Dir); err != nil {
		return fmt.Errorf("failed to upload MP4 files: %w", err)
	}

	// Upload thumbnail
	thumbnailPath, err := s.storage.UploadThumbnail(ctx, event.VideoID, localThumbnailPath)
	if err != nil {
		return fmt.Errorf("failed to upload thumbnail: %w", err)
	}

	// Publish completion event
	completionEvent := events.TranscodingCompleteEvent{
		VideoID:       event.VideoID,
		UserID:        event.UserID,
		Title:         event.Title,
		HLSPath:       hlsPath,
		MP4Path:       mp4Path,
		ThumbnailPath: thumbnailPath,
		Status:        "completed",
		CompletedAt:   time.Now().UTC().Format(time.RFC3339),
	}

	if err := s.producer.PublishTranscodingComplete(ctx, completionEvent); err != nil {
		return fmt.Errorf("failed to publish completion event: %w", err)
	}

	return nil
}
