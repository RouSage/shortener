package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/auth0/go-auth0/v2/management"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rousage/shortener/internal/auth"
	"github.com/rousage/shortener/internal/cache"
	"github.com/rousage/shortener/internal/config"
	"github.com/rousage/shortener/internal/database"
	"github.com/rousage/shortener/internal/repository"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
)

var tracer = otel.Tracer("github.com/rousage/shortener/internal/server")

// AuthManager defines the interface for user management operations with Auth0
type AuthManager interface {
	BlockUser(ctx context.Context, userID string) (*management.UpdateUserResponseContent, error)
	UnblockUser(ctx context.Context, userID string) error
}

type Server struct {
	cfg            *config.Config
	logger         zerolog.Logger
	db             *pgxpool.Pool
	rep            *repository.Queries
	cache          *cache.Cache
	authManagement AuthManager
}

func New(cfg *config.Config) *http.Server {
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	zerolog.TimestampFieldName = "timestamp"
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	if cfg.App.Env == config.EnvProduction {
		logger = logger.Level(zerolog.InfoLevel)
	}

	db := database.Connect(logger, cfg.Database)
	cacheClient := cache.Connect(logger, cfg.Cache)

	srv := &Server{
		cfg:            cfg,
		logger:         logger,
		db:             db,
		rep:            repository.New(db),
		cache:          cache.New(cacheClient),
		authManagement: auth.NewManagement(logger, cfg.Auth),
	}

	// Declare Server config
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", srv.cfg.Server.Port),
		Handler:      srv.RegisterRoutes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 20 * time.Second,
	}

	srv.logger.Info().Msgf("server started on port %d", srv.cfg.Server.Port)

	return server
}
