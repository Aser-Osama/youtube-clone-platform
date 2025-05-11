package events

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

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

// Consumer defines the interface for consuming events
type Consumer interface {
	// Start starts consuming messages from Kafka
	Start(ctx context.Context, handler func(ctx context.Context, event VideoUploadEvent) error) error

	// Close closes the consumer
	Close() error
}

// KafkaConsumer implements the Consumer interface using Kafka
type KafkaConsumer struct {
	reader  *kafka.Reader
	topic   string
	groupID string
}

// NewKafkaConsumer creates a new Kafka consumer
func NewKafkaConsumer(brokers []string, topic string, groupID string) (*KafkaConsumer, error) {
	// Try to create topic with retries
	var conn *kafka.Conn
	var err error
	for i := 0; i < 3; i++ {
		conn, err = kafka.Dial("tcp", brokers[0])
		if err == nil {
			break
		}
		fmt.Printf("Failed to connect to Kafka (attempt %d/3): %v\n", i+1, err)
		time.Sleep(time.Second)
	}

	if err != nil {
		fmt.Printf("Failed to connect to Kafka after 3 attempts: %v\n", err)
	} else {
		defer conn.Close()

		// Create topic with 3 partitions and replication factor of 1
		topicConfigs := []kafka.TopicConfig{
			{
				Topic:             topic,
				NumPartitions:     3,
				ReplicationFactor: 1,
			},
		}

		err = conn.CreateTopics(topicConfigs...)
		if err != nil {
			fmt.Printf("Failed to create topic (this is normal if it already exists): %v\n", err)
		} else {
			fmt.Printf("Created Kafka topic: %s\n", topic)
		}
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		Topic:    topic,
		GroupID:  groupID,
		MinBytes: 10e3, // 10KB
		MaxBytes: 10e6, // 10MB
	})

	return &KafkaConsumer{
		reader:  reader,
		topic:   topic,
		groupID: groupID,
	}, nil
}

// Start starts consuming messages from Kafka
func (c *KafkaConsumer) Start(ctx context.Context, handler func(ctx context.Context, event VideoUploadEvent) error) error {
	log.Printf("Starting Kafka consumer for topic: %s, group: %s", c.topic, c.groupID)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			msg, err := c.reader.ReadMessage(ctx)
			if err != nil {
				log.Printf("Error reading message: %v", err)
				continue
			}

			log.Printf("Received message: %s", string(msg.Value))

			var event VideoUploadEvent
			if err := json.Unmarshal(msg.Value, &event); err != nil {
				log.Printf("Error unmarshaling message: %v", err)
				log.Printf("Message content: %s", string(msg.Value))
				continue
			}

			log.Printf("Successfully unmarshaled event for video ID: %s", event.VideoID)

			// Process the event
			if err := handler(ctx, event); err != nil {
				log.Printf("Error processing event: %v", err)
				continue
			}

			log.Printf("Successfully processed video upload event for video ID: %s", event.VideoID)
		}
	}
}

// Close closes the consumer
func (c *KafkaConsumer) Close() error {
	return c.reader.Close()
}
