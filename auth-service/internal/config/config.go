package config

import (
	"github.com/spf13/viper"
)

type Config struct {
	GoogleClientID     string
	GoogleClientSecret string
	GoogleCallbackURL  string
	DBPath             string
	PrivateKeyPath     string
	PublicKeyPath      string
	SessionSecret      string
}

func LoadConfig() *Config {
	// Setup viper to read from .env file
	viper.SetConfigFile(".env")
	viper.SetConfigType("env")

	// Read from .env file (ignoring error if file doesn't exist)
	_ = viper.ReadInConfig()

	// Set default values
	viper.SetDefault("GOOGLE_CLIENT_ID", "")
	viper.SetDefault("GOOGLE_CLIENT_SECRET", "")
	viper.SetDefault("GOOGLE_CALLBACK_URL", "http://localhost:8081/auth/google/callback")
	viper.SetDefault("SQLITE_PATH", "./data/auth.db")
	viper.SetDefault("PRIVATE_KEY_PATH", "./keys/app.rsa")
	viper.SetDefault("PUBLIC_KEY_PATH", "./keys/app.rsa.pub")
	viper.SetDefault("SESSION_SECRET", "youtube-clone-secret")

	// Also read from environment variables (higher priority than .env)
	viper.AutomaticEnv()

	return &Config{
		GoogleClientID:     viper.GetString("GOOGLE_CLIENT_ID"),
		GoogleClientSecret: viper.GetString("GOOGLE_CLIENT_SECRET"),
		GoogleCallbackURL:  viper.GetString("GOOGLE_CALLBACK_URL"),
		DBPath:             viper.GetString("SQLITE_PATH"),
		PrivateKeyPath:     viper.GetString("PRIVATE_KEY_PATH"),
		PublicKeyPath:      viper.GetString("PUBLIC_KEY_PATH"),
		SessionSecret:      viper.GetString("SESSION_SECRET"),
	}
}
