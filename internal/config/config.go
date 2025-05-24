package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
)

type Config struct {
	App      App
	Database Database
	Server   Server
}

func Load() (*Config, error) {
	err := godotenv.Load()
	if err != nil {
		log.Fatal().Msgf("failed to load environment variables: %s", err)
	}

	config := &Config{}

	config.App, err = loadAppConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load app config: %w", err)
	}

	config.Database, err = loadDatabaseConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load database config: %w", err)
	}

	config.Server, err = loadServerConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load server config: %w", err)
	}

	return config, nil
}

func getEnv(key string) (string, error) {
	if value := os.Getenv(key); value != "" {
		return value, nil
	}

	return "", fmt.Errorf("required environment variable %s is not set", key)
}

func getIntEnv(key string) (int, error) {
	value, err := getEnv(key)
	if err != nil {
		return 0, err
	}

	intVal, err := strconv.Atoi(value)
	if err != nil {
		return 0, err
	}

	return intVal, nil
}
