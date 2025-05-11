package config

import (
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	ServerPort string
	MinIO      MinIOConfig
	Logging    LoggingConfig
}

type MinIOConfig struct {
	Endpoint        string
	AccessKey       string
	SecretKey       string
	UseSSL          bool
	BucketName      string
	HLSPrefix       string
	MP4Prefix       string
	ThumbnailPrefix string
	URLExpiry       int // URL expiry time in seconds
}

type LoggingConfig struct {
	Level string
}

func LoadConfig() (*Config, error) {
	viper.SetConfigFile(".env")
	viper.SetConfigType("env")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	return &Config{
		ServerPort: viper.GetString("SERVER_PORT"),
		MinIO: MinIOConfig{
			Endpoint:        viper.GetString("MINIO_ENDPOINT"),
			AccessKey:       viper.GetString("MINIO_ACCESS_KEY"),
			SecretKey:       viper.GetString("MINIO_SECRET_KEY"),
			UseSSL:          viper.GetBool("MINIO_USE_SSL"),
			BucketName:      viper.GetString("MINIO_BUCKET"),
			HLSPrefix:       viper.GetString("MINIO_HLS_PREFIX"),
			MP4Prefix:       viper.GetString("MINIO_MP4_PREFIX"),
			ThumbnailPrefix: viper.GetString("MINIO_THUMBNAIL_PREFIX"),
			URLExpiry:       viper.GetInt("MINIO_URL_EXPIRY"),
		},
		Logging: LoggingConfig{
			Level: viper.GetString("LOG_LEVEL"),
		},
	}, nil
}
