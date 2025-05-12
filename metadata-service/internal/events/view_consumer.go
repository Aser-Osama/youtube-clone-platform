package kafka

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"youtube-clone-platform/metadata-service/internal/service"

	"github.com/segmentio/kafka-go"
)

// VideoViewEvent represents a video view event from Kafka
type VideoViewEvent struct {
	VideoID    string    `json:"video_id"`
	UserID     string    `json:"user_id"`
	ViewedAt   time.Time `json:"viewed_at"`
	ClientIP   string    `json:"client_ip"`
	UserAgent  string    `json:"user_agent"`
	SessionID  string    `json:"session_id,omitempty"`
	ViewerType string    `json:"viewer_type,omitempty"`
}

// StartViewConsumer starts consuming view events from Kafka
func StartViewConsumer(ctx context.Context, reader *kafka.Reader, metadataService *service.MetadataService) error {
	log.Printf("Starting view events consumer...")
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			msg, err := reader.ReadMessage(ctx)
			if err != nil {
				log.Printf("Error reading view event message: %v", err)
				continue
			}

			log.Printf("Received view event message: %s", string(msg.Value))

			var event VideoViewEvent
			if err := json.Unmarshal(msg.Value, &event); err != nil {
				log.Printf("Error unmarshaling view event message: %v", err)
				log.Printf("Message content: %s", string(msg.Value))
				continue
			}

			log.Printf("Successfully unmarshaled view event for video ID: %s", event.VideoID)

			// Process the view event by incrementing the view count
			if err := metadataService.IncrementViews(ctx, event.VideoID, event.UserID); err != nil {
				log.Printf("Error incrementing view count: %v", err)
				continue
			}

			log.Printf("Successfully processed view event for video ID: %s", event.VideoID)
		}
	}
}
