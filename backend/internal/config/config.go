package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL string
	ServerAddr  string
}

func Load() (Config, error) {
	// Системные переменные окружения имеют приоритет над .env
	if err := godotenv.Load(); err != nil {
		if !os.IsNotExist(err) {
			return Config{}, fmt.Errorf("load .env: %w", err)
		}
	}

	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}

	serverAddr := strings.TrimSpace(os.Getenv("SERVER_ADDR"))
	if serverAddr == "" {
		serverAddr = ":8080"
	}

	return Config{
		DatabaseURL: databaseURL,
		ServerAddr:  serverAddr,
	}, nil
}
