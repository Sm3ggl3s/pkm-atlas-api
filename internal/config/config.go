package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

// Config contains the environment variable configuration for the API.
type Config struct {
	Port        string
	DatabaseURL string
}

// Loads the environment variables from a .env file and returns a Config struct.
func Load() (Config, error) {
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		return Config{}, fmt.Errorf("load .env file: %w", err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return Config{}, fmt.Errorf(
			"missing required environment variable: DATABASE_URL",
		)
	}

	cfg := Config{
		Port:        port,
		DatabaseURL: databaseURL,
	}

	return cfg, nil
}
