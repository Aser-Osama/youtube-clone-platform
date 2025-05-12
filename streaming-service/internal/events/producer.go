package events

import (
	"context"
	"encoding/json"
	"log"

	"github.com/segmentio/kafka-go"
)

// ViewEvent represents a video view event
type ViewEvent struct {
	VideoID string `json:"video_id"`
	UserID  string `json:"user_id"`
}

// Producer defines the interface for event producers
type Producer interface {
	// PublishViewEvent publishes a view event
	PublishViewEvent(ctx context.Context, event ViewEvent) error
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
	writer := &kafka.Writer{
		Addr:     kafka.TCP(brokers...),
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
	}

	return &KafkaProducer{
		writer: writer,
		topic:  topic,
	}
}

// PublishViewEvent publishes a view event to Kafka
func (p *KafkaProducer) PublishViewEvent(ctx context.Context, event ViewEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	err = p.writer.WriteMessages(ctx, kafka.Message{
		Value: data,
	})
	if err != nil {
		log.Printf("Failed to publish view event: %v", err)
		return err
	}

	return nil
}

// Close closes the Kafka producer
func (p *KafkaProducer) Close() error {
	return p.writer.Close()
}
