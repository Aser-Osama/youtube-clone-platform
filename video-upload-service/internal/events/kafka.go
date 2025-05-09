package events

import (
    "log"
    "github.com/segmentio/kafka-go"
)

func PublishUploadEvent(videoID string) {
    w := kafka.NewWriter(kafka.WriterConfig{
        Brokers: []string{"localhost:9092"},
        Topic:   "video-uploads",
    })
    err := w.WriteMessages(nil, kafka.Message{Value: []byte(videoID)})
    if err != nil {
        log.Println("Kafka publish error:", err)
    }
    w.Close()
}

