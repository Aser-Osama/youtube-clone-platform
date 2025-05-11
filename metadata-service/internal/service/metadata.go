package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"youtube-clone-platform/metadata-service/internal/db"
	sqlc "youtube-clone-platform/metadata-service/internal/db/sqlc"
	"youtube-clone-platform/metadata-service/internal/types"

	"github.com/minio/minio-go/v7"
)

// VideoMetadata represents a video's metadata
type VideoMetadata struct {
	ID                string         `json:"id"`
	UserID            string         `json:"user_id"`
	Title             string         `json:"title"`
	Description       sql.NullString `json:"description"`
	Duration          float64        `json:"duration"`
	Width             int64          `json:"width"`
	Height            int64          `json:"height"`
	Format            string         `json:"format"`
	Bitrate           int64          `json:"bitrate"`
	FileSize          int64          `json:"file_size"`
	Checksum          string         `json:"checksum"`
	CreatedAt         time.Time      `json:"created_at"`
	Codec             string         `json:"codec"`
	FrameRate         float64        `json:"frame_rate"`
	AspectRatio       string         `json:"aspect_ratio"`
	AudioCodec        sql.NullString `json:"audio_codec"`
	AudioBitrate      sql.NullInt64  `json:"audio_bitrate"`
	AudioChannels     sql.NullInt64  `json:"audio_channels"`
	ContentType       string         `json:"content_type"`
	OriginalFilename  string         `json:"original_filename"`
	FileExtension     string         `json:"file_extension"`
	SanitizedFilename string         `json:"sanitized_filename"`
	Views             sql.NullInt64  `json:"views"`
	Status            string         `json:"status"`
	MinioPath         string         `json:"minio_path"`
	HLSPath           sql.NullString `json:"hls_path"`
	ThumbnailPath     sql.NullString `json:"thumbnail_path"`
	MP4Path           sql.NullString `json:"mp4_path"`
	Tags              []string       `json:"tags"`
}

// MetadataService handles video metadata operations
type MetadataService struct {
	store       *db.Store
	minioClient *minio.Client
}

// NewMetadataService creates a new metadata service
func NewMetadataService(store *db.Store, minioClient *minio.Client) *MetadataService {
	return &MetadataService{
		store:       store,
		minioClient: minioClient,
	}
}

// ConvertUploadMetadata converts upload service metadata to metadata service format
func (s *MetadataService) ConvertUploadMetadata(event *types.VideoUploadEvent) *VideoMetadata {
	// Parse the CreatedAt string to time.Time
	createdAt, _ := time.Parse(time.RFC3339, event.Metadata.CreatedAt)

	// Set default values for required fields
	metadata := &VideoMetadata{
		ID:                event.VideoID,
		UserID:            event.UserID,
		Title:             event.Title,
		Description:       sql.NullString{String: "", Valid: false},
		Duration:          event.Metadata.Duration,
		Width:             int64(event.Metadata.Width),
		Height:            int64(event.Metadata.Height),
		Format:            event.Metadata.Format,
		Bitrate:           event.Metadata.Bitrate,
		FileSize:          event.Metadata.FileSize,
		Checksum:          event.Metadata.Checksum,
		CreatedAt:         createdAt,
		Codec:             event.Metadata.Codec,
		FrameRate:         event.Metadata.FrameRate,
		AspectRatio:       event.Metadata.AspectRatio,
		AudioCodec:        sql.NullString{String: event.Metadata.AudioCodec, Valid: event.Metadata.AudioCodec != ""},
		AudioBitrate:      sql.NullInt64{Int64: event.Metadata.AudioBitrate, Valid: event.Metadata.AudioBitrate != 0},
		AudioChannels:     sql.NullInt64{Int64: int64(event.Metadata.AudioChannels), Valid: event.Metadata.AudioChannels != 0},
		ContentType:       event.Metadata.ContentType,
		OriginalFilename:  event.Metadata.OriginalFilename,
		FileExtension:     event.Metadata.FileExtension,
		SanitizedFilename: event.Metadata.SanitizedFilename,
		Views:             sql.NullInt64{Int64: 0, Valid: false},
		Status:            "processing",
		MinioPath:         fmt.Sprintf("original/%s%s", event.VideoID, event.Metadata.FileExtension),
		HLSPath:           sql.NullString{String: "", Valid: false},
		ThumbnailPath:     sql.NullString{String: "", Valid: false},
		MP4Path:           sql.NullString{String: "", Valid: false},
		Tags:              []string{},
	}

	return metadata
}

// CreateVideoMetadata creates a new video metadata record
func (s *MetadataService) CreateVideoMetadata(ctx context.Context, metadata *VideoMetadata) error {
	tagsJSON, err := json.Marshal(metadata.Tags)
	if err != nil {
		return fmt.Errorf("failed to marshal tags: %w", err)
	}

	params := sqlc.CreateVideoParams{
		ID:                metadata.ID,
		UserID:            metadata.UserID,
		Title:             metadata.Title,
		Description:       metadata.Description,
		Duration:          metadata.Duration,
		Width:             metadata.Width,
		Height:            metadata.Height,
		Format:            metadata.Format,
		Bitrate:           metadata.Bitrate,
		FileSize:          metadata.FileSize,
		Checksum:          metadata.Checksum,
		CreatedAt:         metadata.CreatedAt,
		Codec:             metadata.Codec,
		FrameRate:         metadata.FrameRate,
		AspectRatio:       metadata.AspectRatio,
		AudioCodec:        metadata.AudioCodec,
		AudioBitrate:      metadata.AudioBitrate,
		AudioChannels:     metadata.AudioChannels,
		ContentType:       metadata.ContentType,
		OriginalFilename:  metadata.OriginalFilename,
		FileExtension:     metadata.FileExtension,
		SanitizedFilename: metadata.SanitizedFilename,
		Status:            metadata.Status,
		MinioPath:         metadata.MinioPath,
		HlsPath:           metadata.HLSPath,
		ThumbnailPath:     metadata.ThumbnailPath,
		Mp4Path:           metadata.MP4Path,
		Tags:              sql.NullString{String: string(tagsJSON), Valid: true},
	}

	return s.store.CreateVideo(ctx, params)
}

// GetVideoMetadata retrieves video metadata by ID
func (s *MetadataService) GetVideoMetadata(ctx context.Context, id string) (*VideoMetadata, error) {
	video, err := s.store.GetVideo(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get video metadata: %w", err)
	}

	var tags []string
	if video.Tags.Valid && video.Tags.String != "" {
		if err := json.Unmarshal([]byte(video.Tags.String), &tags); err != nil {
			return nil, fmt.Errorf("failed to unmarshal tags: %w", err)
		}
	}

	return &VideoMetadata{
		ID:                video.ID,
		UserID:            video.UserID,
		Title:             video.Title,
		Description:       video.Description,
		Duration:          video.Duration,
		Width:             video.Width,
		Height:            video.Height,
		Format:            video.Format,
		Bitrate:           video.Bitrate,
		FileSize:          video.FileSize,
		Checksum:          video.Checksum,
		CreatedAt:         video.CreatedAt,
		Codec:             video.Codec,
		FrameRate:         video.FrameRate,
		AspectRatio:       video.AspectRatio,
		AudioCodec:        video.AudioCodec,
		AudioBitrate:      video.AudioBitrate,
		AudioChannels:     video.AudioChannels,
		ContentType:       video.ContentType,
		OriginalFilename:  video.OriginalFilename,
		FileExtension:     video.FileExtension,
		SanitizedFilename: video.SanitizedFilename,
		Views:             video.Views,
		Status:            video.Status,
		MinioPath:         video.MinioPath,
		HLSPath:           video.HlsPath,
		ThumbnailPath:     video.ThumbnailPath,
		MP4Path:           video.Mp4Path,
		Tags:              tags,
	}, nil
}

// UpdateVideoStatus updates the status of a video
func (s *MetadataService) UpdateVideoStatus(ctx context.Context, id string, status string) error {
	params := sqlc.UpdateVideoStatusParams{
		Status: status,
		ID:     id,
	}
	return s.store.UpdateVideoStatus(ctx, params)
}

// IncrementViews increments the view count for a video
func (s *MetadataService) IncrementViews(ctx context.Context, videoID, userID string) error {
	// Check if user has already viewed the video
	count, err := s.store.CheckVideoView(ctx, sqlc.CheckVideoViewParams{
		VideoID: videoID,
		UserID:  userID,
	})
	if err != nil {
		return fmt.Errorf("failed to check existing view: %w", err)
	}

	if count == 0 {
		// Create view record
		if err := s.store.CreateVideoView(ctx, sqlc.CreateVideoViewParams{
			VideoID: videoID,
			UserID:  userID,
		}); err != nil {
			return fmt.Errorf("failed to create view record: %w", err)
		}

		// Increment view count
		if err := s.store.IncrementViews(ctx, videoID); err != nil {
			return fmt.Errorf("failed to increment view count: %w", err)
		}
	}

	return nil
}

// GetRecentVideos retrieves the most recent videos
func (s *MetadataService) GetRecentVideos(ctx context.Context, limit int) ([]*VideoMetadata, error) {
	videos, err := s.store.GetRecentVideos(ctx, int64(limit))
	if err != nil {
		return nil, fmt.Errorf("failed to get recent videos: %w", err)
	}

	return s.convertVideos(videos)
}

// SearchVideos searches for videos by title, description, or tags
func (s *MetadataService) SearchVideos(ctx context.Context, query string, limit int) ([]*VideoMetadata, error) {
	searchPattern := "%" + query + "%"
	params := sqlc.SearchVideosParams{
		Title:       searchPattern,
		Description: sql.NullString{String: searchPattern, Valid: true},
		Tags:        sql.NullString{String: searchPattern, Valid: true},
		Limit:       int64(limit),
	}

	videos, err := s.store.SearchVideos(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to search videos: %w", err)
	}

	return s.convertVideos(videos)
}

// GetVideosByUser retrieves videos for a specific user
func (s *MetadataService) GetVideosByUser(ctx context.Context, userID string, limit int) ([]*VideoMetadata, error) {
	params := sqlc.GetVideosByUserParams{
		UserID: userID,
		Limit:  int64(limit),
	}

	videos, err := s.store.GetVideosByUser(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get user videos: %w", err)
	}

	return s.convertVideos(videos)
}

// convertVideos converts a slice of SQLC Video models to VideoMetadata
func (s *MetadataService) convertVideos(videos []sqlc.Video) ([]*VideoMetadata, error) {
	result := make([]*VideoMetadata, len(videos))
	for i, video := range videos {
		var tags []string
		if video.Tags.Valid {
			if err := json.Unmarshal([]byte(video.Tags.String), &tags); err != nil {
				tags = []string{}
			}
		}

		result[i] = &VideoMetadata{
			ID:                video.ID,
			UserID:            video.UserID,
			Title:             video.Title,
			Description:       video.Description,
			Duration:          video.Duration,
			Width:             video.Width,
			Height:            video.Height,
			Format:            video.Format,
			Bitrate:           video.Bitrate,
			FileSize:          video.FileSize,
			Checksum:          video.Checksum,
			CreatedAt:         video.CreatedAt,
			Codec:             video.Codec,
			FrameRate:         video.FrameRate,
			AspectRatio:       video.AspectRatio,
			AudioCodec:        video.AudioCodec,
			AudioBitrate:      video.AudioBitrate,
			AudioChannels:     video.AudioChannels,
			ContentType:       video.ContentType,
			OriginalFilename:  video.OriginalFilename,
			FileExtension:     video.FileExtension,
			SanitizedFilename: video.SanitizedFilename,
			Views:             video.Views,
			Status:            video.Status,
			MinioPath:         video.MinioPath,
			HLSPath:           video.HlsPath,
			ThumbnailPath:     video.ThumbnailPath,
			MP4Path:           video.Mp4Path,
			Tags:              tags,
		}
	}
	return result, nil
}

// UpdateVideoFromTranscodingComplete updates a video's metadata after transcoding is complete
func (s *MetadataService) UpdateVideoFromTranscodingComplete(ctx context.Context, event *types.TranscodingCompleteEvent) error {
	// Update path information and status
	params := sqlc.UpdateVideoTranscodingCompleteParams{
		ID:            event.VideoID,
		Status:        event.Status,
		HlsPath:       sql.NullString{String: event.HLSPath, Valid: event.HLSPath != ""},
		ThumbnailPath: sql.NullString{String: event.ThumbnailPath, Valid: event.ThumbnailPath != ""},
		Mp4Path:       sql.NullString{String: event.MP4Path, Valid: event.MP4Path != ""},
	}

	return s.store.UpdateVideoTranscodingComplete(ctx, params)
}
