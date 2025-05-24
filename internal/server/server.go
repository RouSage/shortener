package server

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	_ "github.com/joho/godotenv/autoload"
	"github.com/rousage/shortener/internal/database"
	"github.com/rs/zerolog"
)

type Server struct {
	port   int
	logger zerolog.Logger
	db     database.Service
}

func NewServer() *http.Server {
	zerolog.TimestampFieldName = "timestamp"
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	port, _ := strconv.Atoi(os.Getenv("PORT"))
	NewServer := &Server{
		port:   port,
		db:     database.New(logger),
		logger: logger,
	}

	// Declare Server config
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", NewServer.port),
		Handler:      NewServer.RegisterRoutes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 20 * time.Second,
	}

	return server
}
