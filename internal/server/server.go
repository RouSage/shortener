package server

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rousage/shortener/internal/cache"
	"github.com/rousage/shortener/internal/config"
	"github.com/rousage/shortener/internal/database"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	glide "github.com/valkey-io/valkey-glide/go/v2"
)

type Server struct {
	cfg    *config.Config
	logger zerolog.Logger
	db     *pgxpool.Pool
	cache  *glide.Client
}

func New() *http.Server {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("could not load config")
	}

	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	zerolog.TimestampFieldName = "timestamp"
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	if cfg.App.Env == config.EnvProduction {
		logger = logger.Level(zerolog.InfoLevel)
	}

	srv := &Server{
		cfg:    cfg,
		logger: logger,
		db:     database.Connect(logger, cfg.Database),
		cache:  cache.Connect(logger, cfg.Cache),
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
