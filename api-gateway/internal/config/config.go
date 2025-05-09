package config

import (
    "os"
)

type ServiceConfig struct {
    AuthURL  string
    VideoURL string
}

func LoadConfig() ServiceConfig {
    return ServiceConfig{
        AuthURL:  os.Getenv("AUTH_SERVICE_URL"),
        VideoURL: os.Getenv("VIDEO_SERVICE_URL"),
    }
}

