package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
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
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file, reading env vars")
	}

	return &Config{
		GoogleClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		GoogleCallbackURL:  os.Getenv("GOOGLE_CALLBACK_URL"),
		DBPath:             os.Getenv("SQLITE_PATH"),
		PrivateKeyPath:     os.Getenv("PRIVATE_KEY_PATH"),
		PublicKeyPath:      os.Getenv("PUBLIC_KEY_PATH"),
		SessionSecret:      os.Getenv("SESSION_SECRET"),
	}
}
