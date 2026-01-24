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
	Auth     Auth
	Database Database
	Cache    Cache
	Server   Server
	Otel     Otel
}

func Load() (*Config, error) {
	err := godotenv.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load environment variables")
	}

	config := &Config{}

	config.App, err = loadAppConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load app config: %w", err)
	}

	config.Auth, err = loadAuthConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load auth config: %w", err)
	}

	config.Database, err = loadDatabaseConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load database config: %w", err)
	}

	config.Cache, err = loadCacheConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load cache config: %w", err)
	}

	config.Server, err = loadServerConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load server config: %w", err)
	}

	config.Otel, err = loadOtelConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load otel config: %w", err)
	}

	return config, nil
}

func getOptionalEnv(key string) string {
	return os.Getenv(key)
}

func getEnv(key string) (string, error) {
	if value := getOptionalEnv(key); value != "" {
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
