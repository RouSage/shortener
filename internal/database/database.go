package database

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/exaring/otelpgx"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rousage/shortener/internal/config"
)

//go:embed migrations
var migrations embed.FS

func Connect(logger *slog.Logger, cfg config.Database) *pgxpool.Pool {
	connString := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable&search_path=%s", cfg.Username, cfg.Password, cfg.Host, cfg.Port, cfg.Database, cfg.Schema)

	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		logger.Error("failed to parse connection string", "error", err)
		os.Exit(1)
	}
	config.MaxConns = 30
	config.MinIdleConns = 5
	config.MaxConnLifetimeJitter = 5 * time.Minute
	config.ConnConfig.Tracer = otelpgx.NewTracer()

	conn, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		logger.Error("failed to create DB pool", "error", err)
		os.Exit(1)
	}

	err = migrateDB(connString)
	if err != nil {
		logger.Error("failed to migrate DB")
		os.Exit(1)
	}

	err = conn.Ping(context.Background())
	if err != nil {
		logger.Error("failed to ping DB", "error", err)
		os.Exit(1)
	}

	err = otelpgx.RecordStats(conn, otelpgx.WithMinimumReadDBStatsInterval(5*time.Second))
	if err != nil {
		logger.Error("unable to record database stats", "error", err)
		os.Exit(1)
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
