package server

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rousage/shortener/internal/config"
	"github.com/rousage/shortener/internal/database"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Server struct {
	cfg    *config.Config
	logger zerolog.Logger
	db     *pgxpool.Pool
}

func New() *http.Server {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Msgf("Error loading config: %s", err)
	}

	zerolog.TimestampFieldName = "timestamp"
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	srv := &Server{
		cfg:    cfg,
		logger: logger,
		db:     database.Connect(logger, cfg.Database),
	}

	// Declare Server config
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", srv.cfg.Server.Port),
		Handler:      srv.RegisterRoutes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 20 * time.Second,
	}

	srv.logger.Info().Msgf("Server started on port %d", srv.cfg.Server.Port)

	return server
}
