package testhelpers

import (
	"context"
	"time"

	"github.com/rousage/shortener/internal/config"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/modules/valkey"
	"github.com/testcontainers/testcontainers-go/wait"
)

type PostgresContainer struct {
	*postgres.PostgresContainer
	DatabaseConfig config.Database
}

func CreatePostgresContainer(ctx context.Context) (*PostgresContainer, error) {
	cfg := config.Database{
		Database: "database",
		Password: "password",
		Username: "user",
		Schema:   "public",
	}

	dbContainer, err := postgres.Run(
		ctx,
		"postgres:latest",
		postgres.WithDatabase(cfg.Database),
		postgres.WithUsername(cfg.Username),
		postgres.WithPassword(cfg.Password),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(5*time.Second)),
		postgres.WithSQLDriver("pgx"),
	)
	if err != nil {
		return nil, err
	}

	cfg.Host, err = dbContainer.Host(ctx)
	if err != nil {
		return nil, err
	}

	dbPort, err := dbContainer.MappedPort(ctx, "5432/tcp")
	if err != nil {
		return nil, err
	}

	cfg.Port = dbPort.Int()

	return &PostgresContainer{
		PostgresContainer: dbContainer,
		DatabaseConfig:    cfg,
	}, err
}

type ValkeyContainer struct {
	*valkey.ValkeyContainer
	CacheConfig config.Cache
}

func CreateValkeyContainer(ctx context.Context) (*ValkeyContainer, error) {
	cacheContainer, err := valkey.Run(
		ctx,
		"valkey/valkey:latest",
		valkey.WithLogLevel(valkey.LogLevelVerbose),
	)
	if err != nil {
		return nil, err
	}

	cfg := config.Cache{}

	cfg.Host, err = cacheContainer.Host(ctx)
	if err != nil {
		return nil, err
	}

	port, err := cacheContainer.MappedPort(ctx, "6379/tcp")
	if err != nil {
		return nil, err
	}
	cfg.Port = port.Int()

	return &ValkeyContainer{
		ValkeyContainer: cacheContainer,
		CacheConfig:     cfg,
	}, err
}
