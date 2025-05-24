package server

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	_ "github.com/joho/godotenv/autoload"
	"github.com/rousage/shortener/internal/database"
	"github.com/rousage/shortener/internal/repository"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Server struct {
	port   int
	logger zerolog.Logger
	db     database.Service

	// Repositories
	repository *repository.Queries
}

func NewServer() *http.Server {
	zerolog.TimestampFieldName = "timestamp"
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	port, err := strconv.Atoi(os.Getenv("PORT"))
	if err != nil {
		log.Fatal().Msgf("Error parsing PORT: %v", err)
	}

	db := database.New(logger)

	NewServer := &Server{
		port:       port,
		logger:     logger,
		db:         db,
		repository: repository.New(db.GetDB()),
	}

	// Declare Server config
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", NewServer.port),
		Handler:      NewServer.RegisterRoutes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 20 * time.Second,
	}

	NewServer.logger.Info().Msgf("Server started on port %d", NewServer.port)

	return server
}
