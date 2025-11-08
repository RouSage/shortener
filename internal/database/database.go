package database

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/exaring/otelpgx"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rousage/shortener/internal/config"
	"github.com/rs/zerolog"
)

//go:embed migrations
var migrations embed.FS

func Connect(logger zerolog.Logger, cfg config.Database) *pgxpool.Pool {
	connString := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable&search_path=%s", cfg.Username, cfg.Password, cfg.Host, cfg.Port, cfg.Database, cfg.Schema)

	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to parse connection string")
	}
	config.MaxConns = 30
	config.MinIdleConns = 5
	config.MaxConnLifetimeJitter = 5 * time.Minute
	config.ConnConfig.Tracer = otelpgx.NewTracer()

	conn, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to create DB pool")
	}

	err = migrateDB(connString)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to migrate DB")
	}

	err = conn.Ping(context.Background())
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to ping DB")
	}

	err = otelpgx.RecordStats(conn, otelpgx.WithMinimumReadDBStatsInterval(5*time.Second))
	if err != nil {
		logger.Fatal().Err(err).Msg("unable to record database stats")
	}

	return conn
}

func migrateDB(connString string) error {
	source, err := iofs.New(migrations, "migrations")
	if err != nil {
		return err
	}

	m, err := migrate.NewWithSourceInstance("iofs", source, strings.Replace(connString, "postgres://", "pgx5://", 1))
	if err != nil {
		return err
	}

	err = m.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}

	sourceErr, dbErr := m.Close()
	if sourceErr != nil {
		return err
	}
	if dbErr != nil {
		return dbErr
	}

	return nil
}
