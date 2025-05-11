package config

import (
	"github.com/spf13/viper"
)

type Config struct {
	Server    ServerConfig    `mapstructure:"server"`
	Services  ServicesConfig  `mapstructure:"services"`
	JWT       JWTConfig       `mapstructure:"jwt"`
	RateLimit RateLimitConfig `mapstructure:"rate_limit"`
}

type ServerConfig struct {
	Port int `mapstructure:"port"`
}

type ServicesConfig struct {
	Auth      string `mapstructure:"auth"`
	Metadata  string `mapstructure:"metadata"`
	Streaming string `mapstructure:"streaming"`
	Upload    string `mapstructure:"upload"`
}

type JWTConfig struct {
	PublicKeyPath string `mapstructure:"public_key_path"`
}

type RateLimitConfig struct {
	Requests int    `mapstructure:"requests"`
	Period   string `mapstructure:"period"`
}

func LoadConfig() (*Config, error) {
	viper.AutomaticEnv()

	// Set default values
	viper.SetDefault("SERVER_PORT", 8080)
	viper.SetDefault("RATE_LIMIT_REQUESTS", 100)
	viper.SetDefault("RATE_LIMIT_PERIOD", "1m")

	// Load .env file if it exists
	viper.SetConfigFile(".env")
	viper.ReadInConfig()

	config := &Config{
		Server: ServerConfig{
			Port: viper.GetInt("SERVER_PORT"),
		},
		Services: ServicesConfig{
			Auth:      viper.GetString("AUTH_SERVICE_URL"),
			Metadata:  viper.GetString("METADATA_SERVICE_URL"),
			Streaming: viper.GetString("STREAMING_SERVICE_URL"),
			Upload:    viper.GetString("UPLOAD_SERVICE_URL"),
		},
		JWT: JWTConfig{
			PublicKeyPath: viper.GetString("JWT_PUBLIC_KEY_PATH"),
		},
		RateLimit: RateLimitConfig{
			Requests: viper.GetInt("RATE_LIMIT_REQUESTS"),
			Period:   viper.GetString("RATE_LIMIT_PERIOD"),
		},
	}

	return config, nil
}
