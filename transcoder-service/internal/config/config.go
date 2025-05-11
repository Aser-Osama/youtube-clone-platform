package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// Config holds all configuration for the transcoder service
type Config struct {
	// Server configuration
	Port string

	// Kafka configuration
	Kafka KafkaConfig

	// MinIO configuration
	MinIO MinIOConfig

	// FFmpeg configuration
	FFmpeg FFmpegConfig

	// Processing configuration
	Processing ProcessingConfig
}

type MinIOConfig struct {
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	UseSSL          bool
	BucketName      string
	ProcessedBucket string
	OriginalPrefix  string
	HLSPrefix       string
	MP4Prefix       string
	ThumbnailPrefix string
}

type KafkaConfig struct {
	Brokers []string
	Topic   string
	GroupID string
}

type FFmpegConfig struct {
	Path            string
	Threads         int
	Preset          string
	CRF             int
	SegmentLength   int
	OutputFormats   []string
	OutputQualities []string
}

type ProcessingConfig struct {
	MaxConcurrentJobs int
	JobTimeout        time.Duration
	TempDir           string
}

func Load() (*Config, error) {
	// Setup viper to read from .env file
	viper.SetConfigFile(".env")
	viper.SetConfigType("env")

	// Read from .env file (ignoring error if file doesn't exist)
	_ = viper.ReadInConfig()

	// Set default values
	viper.SetDefault("PORT", "8083")
	viper.SetDefault("MINIO_ENDPOINT", "localhost:9000")
	viper.SetDefault("MINIO_ACCESS_KEY", "minioadmin")
	viper.SetDefault("MINIO_SECRET_KEY", "minioadmin")
	viper.SetDefault("MINIO_USE_SSL", false)
	viper.SetDefault("MINIO_BUCKET", "rawvideos")
	viper.SetDefault("MINIO_PROCESSED_BUCKET", "processedvideos")
	viper.SetDefault("MINIO_ORIGINAL_PREFIX", "original")
	viper.SetDefault("MINIO_HLS_PREFIX", "hls")
	viper.SetDefault("MINIO_MP4_PREFIX", "mp4")
	viper.SetDefault("MINIO_THUMBNAIL_PREFIX", "thumbnails")
	viper.SetDefault("KAFKA_BROKERS", []string{"localhost:29092"})
	viper.SetDefault("KAFKA_TOPIC", "video-uploads")
	viper.SetDefault("KAFKA_GROUP_ID", "transcoder-service")
	viper.SetDefault("FFMPEG_PATH", "ffmpeg")
	viper.SetDefault("FFMPEG_THREADS", 4)
	viper.SetDefault("FFMPEG_PRESET", "medium")
	viper.SetDefault("FFMPEG_CRF", 23)
	viper.SetDefault("FFMPEG_SEGMENT_LENGTH", 10)
	viper.SetDefault("FFMPEG_OUTPUT_FORMATS", []string{"h264"})
	viper.SetDefault("FFMPEG_OUTPUT_QUALITIES", []string{"1080p", "720p", "480p", "360p"})
	viper.SetDefault("MAX_CONCURRENT_JOBS", 2)
	viper.SetDefault("JOB_TIMEOUT", "30m")
	viper.SetDefault("TEMP_DIR", "/tmp/transcoder")

	// Also read from environment variables
	viper.AutomaticEnv()

	// Parse job timeout
	jobTimeout, err := time.ParseDuration(viper.GetString("JOB_TIMEOUT"))
	if err != nil {
		jobTimeout = 30 * time.Minute // Default fallback
	}

	return &Config{
		Port: viper.GetString("PORT"),
		Kafka: KafkaConfig{
			Brokers: viper.GetStringSlice("KAFKA_BROKERS"),
			Topic:   viper.GetString("KAFKA_TOPIC"),
			GroupID: viper.GetString("KAFKA_GROUP_ID"),
		},
		MinIO: MinIOConfig{
			Endpoint:        viper.GetString("MINIO_ENDPOINT"),
			AccessKeyID:     viper.GetString("MINIO_ACCESS_KEY"),
			SecretAccessKey: viper.GetString("MINIO_SECRET_KEY"),
			UseSSL:          viper.GetBool("MINIO_USE_SSL"),
			BucketName:      viper.GetString("MINIO_BUCKET"),
			ProcessedBucket: viper.GetString("MINIO_PROCESSED_BUCKET"),
			OriginalPrefix:  viper.GetString("MINIO_ORIGINAL_PREFIX"),
			HLSPrefix:       viper.GetString("MINIO_HLS_PREFIX"),
			MP4Prefix:       viper.GetString("MINIO_MP4_PREFIX"),
			ThumbnailPrefix: viper.GetString("MINIO_THUMBNAIL_PREFIX"),
		},
		FFmpeg: FFmpegConfig{
			Path:            viper.GetString("FFMPEG_PATH"),
			Threads:         viper.GetInt("FFMPEG_THREADS"),
			Preset:          viper.GetString("FFMPEG_PRESET"),
			CRF:             viper.GetInt("FFMPEG_CRF"),
			SegmentLength:   viper.GetInt("FFMPEG_SEGMENT_LENGTH"),
			OutputFormats:   viper.GetStringSlice("FFMPEG_OUTPUT_FORMATS"),
			OutputQualities: viper.GetStringSlice("FFMPEG_OUTPUT_QUALITIES"),
		},
		Processing: ProcessingConfig{
			MaxConcurrentJobs: viper.GetInt("MAX_CONCURRENT_JOBS"),
			JobTimeout:        jobTimeout,
			TempDir:           viper.GetString("TEMP_DIR"),
		},
	}, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if len(c.Kafka.Brokers) == 0 {
		return fmt.Errorf("Kafka brokers cannot be empty")
	}

	if c.Kafka.Topic == "" {
		return fmt.Errorf("Kafka topic cannot be empty")
	}

	if c.Kafka.GroupID == "" {
		return fmt.Errorf("Kafka group ID cannot be empty")
	}

	if c.MinIO.Endpoint == "" {
		return fmt.Errorf("MinIO endpoint cannot be empty")
	}

	if c.MinIO.AccessKeyID == "" {
		return fmt.Errorf("MinIO access key cannot be empty")
	}

	if c.MinIO.SecretAccessKey == "" {
		return fmt.Errorf("MinIO secret key cannot be empty")
	}

	if c.MinIO.BucketName == "" {
		return fmt.Errorf("MinIO bucket name cannot be empty")
	}

	if c.MinIO.ProcessedBucket == "" {
		return fmt.Errorf("MinIO processed bucket name cannot be empty")
	}

	if c.FFmpeg.Path == "" {
		return fmt.Errorf("FFmpeg path cannot be empty")
	}

	if c.FFmpeg.Threads <= 0 {
		return fmt.Errorf("FFmpeg threads must be greater than 0")
	}

	if c.FFmpeg.CRF < 0 || c.FFmpeg.CRF > 51 {
		return fmt.Errorf("FFmpeg CRF must be between 0 and 51")
	}

	if c.FFmpeg.SegmentLength <= 0 {
		return fmt.Errorf("FFmpeg segment length must be greater than 0")
	}

	if len(c.FFmpeg.OutputFormats) == 0 {
		return fmt.Errorf("FFmpeg output formats cannot be empty")
	}

	if len(c.FFmpeg.OutputQualities) == 0 {
		return fmt.Errorf("FFmpeg output qualities cannot be empty")
	}

	if c.Processing.MaxConcurrentJobs <= 0 {
		return fmt.Errorf("Max concurrent jobs must be greater than 0")
	}

	if c.Processing.JobTimeout <= 0 {
		return fmt.Errorf("Job timeout must be greater than 0")
	}

	if c.Processing.TempDir == "" {
		return fmt.Errorf("Temp directory cannot be empty")
	}

	return nil
}
