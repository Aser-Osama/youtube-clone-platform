package events

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
)

// TranscodingCompleteEvent represents a transcoding completion event
type TranscodingCompleteEvent struct {
	VideoID       string `json:"video_id"`
	UserID        string `json:"user_id"`
	Title         string `json:"title"`
	HLSPath       string `json:"hls_path"`
	MP4Path       string `json:"mp4_path"`
	ThumbnailPath string `json:"thumbnail_path"`
	Status        string `json:"status"`
	CompletedAt   string `json:"completed_at"`
}

// Producer defines the interface for producing events
type Producer interface {
	// PublishTranscodingComplete publishes a transcoding completion event
	PublishTranscodingComplete(ctx context.Context, event TranscodingCompleteEvent) error

	// Close closes the producer
	Close() error
}

// KafkaProducer implements the Producer interface using Kafka
type KafkaProducer struct {
	writer *kafka.Writer
	topic  string
}

// NewKafkaProducer creates a new Kafka producer
func NewKafkaProducer(brokers []string, topic string) *KafkaProducer {
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

	return &KafkaProducer{
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

// PublishTranscodingComplete publishes a transcoding completion event
func (p *KafkaProducer) PublishTranscodingComplete(ctx context.Context, event TranscodingCompleteEvent) error {
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

// Close closes the producer
func (p *KafkaProducer) Close() error {
	return p.writer.Close()
}
