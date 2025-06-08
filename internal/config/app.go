package config

import "github.com/rs/zerolog/log"

type App struct {
	Env            string
	ShortUrlLength int
}

func loadAppConfig() (App, error) {
	env, err := getEnv("APP_ENV")
	if err != nil {
		return App{}, err
	}

	shortUrlLength, err := getIntEnv("SHORT_URL_LENGTH")
	if err != nil {
		// default to 0 if the env var is not set
		// it's ok to not set this value, the generator package will use a default value
		log.Warn().Msg("SHORT_URL_LENGTH environment variables is not set, setting to 0")
		shortUrlLength = 0
	}

	return App{
		Env:            env,
		ShortUrlLength: shortUrlLength,
	}, nil
}
