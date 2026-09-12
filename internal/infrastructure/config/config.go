package config

import (
	"os"
)

// AppConfig holds the central configuration for the Astria platform.
type AppConfig struct {
	OpenAIKey   string `envconfig:"OPENAI_API_KEY" required:"true"`
	DatabaseURL string `envconfig:"DATABASE_URL" required:"true"`
	BatchSize   int    `envconfig:"BATCH_SIZE" default:"100"`
	MaxWorkers  int    `envconfig:"MAX_WORKERS" default:"5"`
}

// Load reads configuration from environment variables.
func Load() *AppConfig {
	return &AppConfig{
		OpenAIKey: os.Getenv("OPENAI_API_KEY"),
	}
}
