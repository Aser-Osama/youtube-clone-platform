package events

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aser/youtube-clone-platform/video-upload-service/internal/metadata"
	"github.com/segmentio/kafka-go"
)

type VideoUploadEvent struct {
	VideoID     string                 `json:"video_id"`
	UserID      string                 `json:"user_id"`
	Title       string                 `json:"title"`
	ContentType string                 `json:"content_type"`
	Size        int64                  `json:"size"`
	Metadata    metadata.VideoMetadata `json:"metadata"`
	UploadedAt  string                 `json:"uploaded_at"`
}

type Publisher interface {
	PublishVideoUpload(ctx context.Context, event VideoUploadEvent) error
}

type KafkaPublisher struct {
	writer *kafka.Writer
	topic  string
}

func NewKafkaPublisher(brokers []string, topic string) *KafkaPublisher {
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

	return &KafkaPublisher{
		writer: &kafka.Writer{
			Addr:         kafka.TCP(brokers...),
			Topic:        topic,
			Balancer:     &kafka.LeastBytes{},
			RequiredAcks: kafka.RequireOne,
			Async:        false, // Synchronous writes for better reliability
			BatchTimeout: 10 * time.Millisecond,
			MaxAttempts:  3,
		},
		topic: topic,
	}
}

func (p *KafkaPublisher) PublishVideoUpload(ctx context.Context, event VideoUploadEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Add timeout to context
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err = p.writer.WriteMessages(ctx, kafka.Message{
		Value: payload,
	})
	if err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	return nil
}

func (p *KafkaPublisher) Close() error {
	return p.writer.Close()
}
