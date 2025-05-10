package config

import (
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	DatabasePath string
	KafkaBrokers []string
	KafkaTopic   string
	KafkaGroupID string
	MinIO        MinIOConfig
	ServerPort   string
}

type MinIOConfig struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	UseSSL    bool
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
		DatabasePath: viper.GetString("DATABASE_PATH"),
		KafkaBrokers: viper.GetStringSlice("KAFKA_BROKERS"),
		KafkaTopic:   viper.GetString("KAFKA_TOPICS_VIDEO_UPLOAD"),
		KafkaGroupID: viper.GetString("KAFKA_GROUP_ID"),
		MinIO: MinIOConfig{
			Endpoint:  viper.GetString("MINIO_ENDPOINT"),
			AccessKey: viper.GetString("MINIO_ACCESS_KEY"),
			SecretKey: viper.GetString("MINIO_SECRET_KEY"),
			UseSSL:    viper.GetBool("MINIO_USE_SSL"),
		},
		ServerPort: viper.GetString("SERVER_PORT"),
	}, nil
}
