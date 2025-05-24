package database

import (
	"context"
	"fmt"
	"log"

	zerologadapter "github.com/jackc/pgx-zerolog"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/tracelog"
	_ "github.com/joho/godotenv/autoload"
	"github.com/rousage/shortener/internal/config"
	"github.com/rs/zerolog"
)

func Connect(logger zerolog.Logger, cfg config.Database) *pgxpool.Pool {
	connString := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable&search_path=%s", cfg.Username, cfg.Password, cfg.Host, cfg.Port, cfg.Database, cfg.Schema)

	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		log.Fatal(err)
	}
	config.ConnConfig.Tracer = &tracelog.TraceLog{
		Logger:   zerologadapter.NewLogger(logger),
		LogLevel: tracelog.LogLevelTrace,
	}

	db, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		log.Fatal(err)
	}
	err = db.Ping(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	return db
}
