package config

import (
	"errors"
	"strings"

	"github.com/rs/zerolog/log"
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

func loadServerConfig() (Server, error) {
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
		log.Warn().Msgf("LIMITER_RPS environment variables is not set, setting to %d", defaultRPS)
		rps = defaultRPS
	}
	burst, err := getIntEnv("LIMITER_BURST")
	if err != nil {
		log.Warn().Msgf("LIMITER_BURST environment variables is not set, setting to %d", defaultBurst)
		burst = defaultBurst
	}

	return Server{
		Port:         port,
		AllowOrigins: origins,
		LimiterRPS:   rps,
		LimiterBurst: burst,
	}, nil
}
