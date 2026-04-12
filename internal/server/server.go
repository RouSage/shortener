package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/auth0/go-auth0/v2/management"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rousage/shortener/internal/auth"
	"github.com/rousage/shortener/internal/cache"
	"github.com/rousage/shortener/internal/config"
	"github.com/rousage/shortener/internal/database"
	"github.com/rousage/shortener/internal/repository"
	"go.opentelemetry.io/otel"
)

const name = "github.com/rousage/shortener/internal/server"

var tracer = otel.Tracer(name)

// AuthManager defines the interface for user management operations with Auth0
type AuthManager interface {
	BlockUser(ctx context.Context, userID string) (*management.UpdateUserResponseContent, error)
	UnblockUser(ctx context.Context, userID string) error
}

type Server struct {
	cfg            *config.Config
	db             *pgxpool.Pool
	rep            *repository.Queries
	cache          *cache.Cache
	authManagement AuthManager
}

func New(cfg *config.Config) *http.Server {
	logger := newLogger(cfg.App.Env)
	db := database.Connect(logger, cfg.Database)
	cacheClient := cache.Connect(logger, cfg.Cache)

	srv := &Server{
		cfg:            cfg,
		db:             db,
		rep:            repository.New(db),
		cache:          cache.New(cacheClient),
		authManagement: auth.NewManagement(logger, cfg.Auth),
	}

	// Declare Server config
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", srv.cfg.Server.Port),
		Handler:      srv.RegisterRoutes(logger),
		IdleTimeout:  time.Minute,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 20 * time.Second,
	}

	logger.Info("server started on port", slog.Int("port", srv.cfg.Server.Port))

	return server
}
