package config

import (
	"errors"
	"strings"
)

type Server struct {
	Port         int
	AllowOrigins []string
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

	return Server{
		Port:         port,
		AllowOrigins: origins,
	}, nil
}
