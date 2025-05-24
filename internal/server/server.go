package server

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/joho/godotenv/autoload"
	"github.com/rousage/shortener/internal/database"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Server struct {
	port   int
	logger zerolog.Logger
	db     *pgxpool.Pool
}

func New() *http.Server {
	zerolog.TimestampFieldName = "timestamp"
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	port, err := strconv.Atoi(os.Getenv("PORT"))
	if err != nil {
		log.Fatal().Msgf("Error parsing PORT: %v", err)
	}

	srv := &Server{
		port:   port,
		logger: logger,
		db:     database.Connect(logger),
	}

	// Declare Server config
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", srv.port),
		Handler:      srv.RegisterRoutes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 20 * time.Second,
	}

	srv.logger.Info().Msgf("Server started on port %d", srv.port)

	return server
}
