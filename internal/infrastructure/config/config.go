package config

import (
	"os"
)

// AppConfig holds the central configuration for the Astria platform.
type AppConfig struct {
	OpenAIKey string
}

// Load reads configuration from environment variables.
func Load() *AppConfig {
	return &AppConfig{
		OpenAIKey: os.Getenv("OPENAI_API_KEY"),
	}
}
