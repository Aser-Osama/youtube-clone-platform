package kafka

import (
	"context"
	"encoding/json"
	"log"

	"youtube-clone-platform/metadata-service/internal/service"
	"youtube-clone-platform/metadata-service/internal/types"

	"github.com/segmentio/kafka-go"
)

// VideoUploadEvent represents a video upload event from Kafka
type VideoUploadEvent struct {
	VideoID     string        `json:"video_id"`
	UserID      string        `json:"user_id"`
	Title       string        `json:"title"`
	ContentType string        `json:"content_type"`
	Size        int64         `json:"size"`
	Metadata    VideoMetadata `json:"metadata"`
	UploadedAt  string        `json:"uploaded_at"`
}

// VideoMetadata represents the metadata extracted from a video file
type VideoMetadata struct {
	Duration          float64 `json:"duration"`
	Width             int     `json:"width"`
	Height            int     `json:"height"`
	Format            string  `json:"format"`
	Bitrate           int64   `json:"bitrate"`
	FileSize          int64   `json:"file_size"`
	Checksum          string  `json:"checksum"`
	CreatedAt         string  `json:"created_at"`
	Codec             string  `json:"codec"`
	FrameRate         float64 `json:"frame_rate"`
	AspectRatio       string  `json:"aspect_ratio"`
	AudioCodec        string  `json:"audio_codec"`
	AudioBitrate      int64   `json:"audio_bitrate"`
	AudioChannels     int     `json:"audio_channels"`
	ContentType       string  `json:"content_type"`
	OriginalFilename  string  `json:"original_filename"`
	FileExtension     string  `json:"file_extension"`
	SanitizedFilename string  `json:"sanitized_filename"`
}

// NewConsumer creates a new Kafka consumer
func NewConsumer(brokers []string, topic string, groupID string) (*kafka.Reader, error) {
	log.Printf("Creating Kafka consumer with brokers: %v, topic: %s, groupID: %s", brokers, topic, groupID)
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		Topic:    topic,
		GroupID:  groupID,
		MinBytes: 10e3, // 10KB
		MaxBytes: 10e6, // 10MB
	})

	return reader, nil
}

// StartConsumer starts consuming messages from Kafka
func StartConsumer(ctx context.Context, reader *kafka.Reader, metadataService *service.MetadataService) error {
	log.Printf("Starting Kafka consumer...")
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			msg, err := reader.ReadMessage(ctx)
			if err != nil {
				log.Printf("Error reading message: %v", err)
				continue
			}

			log.Printf("Received message: %s", string(msg.Value))

			var event types.VideoUploadEvent
			if err := json.Unmarshal(msg.Value, &event); err != nil {
				log.Printf("Error unmarshaling message: %v", err)
				log.Printf("Message content: %s", string(msg.Value))
				continue
			}

			log.Printf("Successfully unmarshaled event for video ID: %s", event.VideoID)

			// Convert event to metadata
			metadata := metadataService.ConvertUploadMetadata(&event)

			log.Printf("Converted metadata for video ID: %s", metadata.ID)

			// Store metadata
			if err := metadataService.CreateVideoMetadata(ctx, metadata); err != nil {
				log.Printf("Error storing metadata: %v", err)
				continue
			}

			log.Printf("Successfully processed video upload event for video ID: %s", event.VideoID)
		}
	}
}
