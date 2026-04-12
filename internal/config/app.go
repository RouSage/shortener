package config

import (
	"errors"
	"slices"

	"log/slog"
)

type App struct {
	Env            Environment
	ShortUrlLength int
}

type Environment = string

const (
	EnvLocal       Environment = "local"
	EnvDevelopment Environment = "development"
	EnvProduction  Environment = "production"
)

var allowedEnvs = []Environment{EnvLocal, EnvDevelopment, EnvProduction}

func loadAppConfig(logger *slog.Logger) (App, error) {
	env, err := getEnv("APP_ENV")
	if err != nil {
		return App{}, err
	}
	if !slices.Contains(allowedEnvs, env) {
		return App{}, errors.New("invalid environment")
	}

	shortUrlLength, err := getIntEnv("SHORT_URL_LENGTH")
	if err != nil {
		// default to 0 if the env var is not set
		// it's ok to not set this value, the generator package will use a default value
		logger.Warn("SHORT_URL_LENGTH environment variables is not set, setting to 0")
		shortUrlLength = 0
	}

	return App{
		Env:            Environment(env),
		ShortUrlLength: shortUrlLength,
	}, nil
}
