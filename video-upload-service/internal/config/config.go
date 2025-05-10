package config

import (
	"github.com/spf13/viper"
)

type Config struct {
	Port     string
	MinIO    MinIOConfig
	Kafka    KafkaConfig
	MaxBytes int64
}

type MinIOConfig struct {
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	UseSSL          bool
	BucketName      string
}

type KafkaConfig struct {
	Brokers []string
	Topic   string
}

func Load() (*Config, error) {
	// Setup viper to read from .env file
	viper.SetConfigFile(".env")
	viper.SetConfigType("env")

	// Read from .env file (ignoring error if file doesn't exist)
	_ = viper.ReadInConfig()

	// Set default values (used if not found in .env)
	viper.SetDefault("PORT", "8080")
	viper.SetDefault("MAX_BYTES", 1024*1024*1024*5) // 1GB default max upload size
	viper.SetDefault("MINIO_ENDPOINT", "localhost:9000")
	viper.SetDefault("MINIO_ACCESS_KEY", "minioadmin")
	viper.SetDefault("MINIO_SECRET_KEY", "minioadmin")
	viper.SetDefault("MINIO_USE_SSL", false)
	viper.SetDefault("MINIO_BUCKET", "rawvideos")
	viper.SetDefault("KAFKA_BROKERS", []string{"localhost:29092"})
	viper.SetDefault("KAFKA_TOPIC", "video-uploads")

	// Also read from environment variables (higher priority than .env)
	viper.AutomaticEnv()

	return &Config{
		Port: viper.GetString("PORT"),
		MinIO: MinIOConfig{
			Endpoint:        viper.GetString("MINIO_ENDPOINT"),
			AccessKeyID:     viper.GetString("MINIO_ACCESS_KEY"),
			SecretAccessKey: viper.GetString("MINIO_SECRET_KEY"),
			UseSSL:          viper.GetBool("MINIO_USE_SSL"),
			BucketName:      viper.GetString("MINIO_BUCKET"),
		},
		Kafka: KafkaConfig{
			Brokers: viper.GetStringSlice("KAFKA_BROKERS"),
			Topic:   viper.GetString("KAFKA_TOPIC"),
		},
		MaxBytes: viper.GetInt64("MAX_BYTES"),
	}, nil
}
