package config

import (
	"errors"
	"strings"

	"log/slog"
)

const (
	defaultRPS   = 10
	defaultBurst = 20
)

type Server struct {
	Port         int
	AllowOrigins []string

	LimiterRPS   int
	LimiterBurst int
}

func loadServerConfig(logger *slog.Logger) (Server, error) {
	port, err := getIntEnv("PORT")
	if err != nil {
		return Server{}, err
	}

	originEnv, err := getEnv("ALLOW_ORIGINS")
	if err != nil {
		return Server{}, err
	}
	origins := strings.Split(originEnv, ",")
	if len(origins) == 0 {
		return Server{}, errors.New("empty CORS origins configuration")
	}

	rps, err := getIntEnv("LIMITER_RPS")
	if err != nil {
		logger.Warn("LIMITER_RPS environment variable is not set, setting to default", slog.Int("defaultRPS", defaultRPS))
		rps = defaultRPS
	}
	burst, err := getIntEnv("LIMITER_BURST")
	if err != nil {
		logger.Warn("LIMITER_BURST environment variable is not set, setting to default", slog.Int("defaultBurst", defaultBurst))
		burst = defaultBurst
	}

	return Server{
		Port:         port,
		AllowOrigins: origins,
		LimiterRPS:   rps,
		LimiterBurst: burst,
	}, nil
}
