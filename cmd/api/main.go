package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rousage/shortener/internal/config"
	"github.com/rousage/shortener/internal/otel"
	"github.com/rousage/shortener/internal/server"
)

func gracefulShutdown(ctx context.Context, logger *slog.Logger, apiServer *http.Server, done chan bool) {
	// Create context that listens for the interrupt signal from the OS.
	ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Listen for the interrupt signal.
	<-ctx.Done()

	logger.Info("shutting down gracefully, press Ctrl+C again to force")
	stop() // Allow Ctrl+C to force shutdown

	// The context is used to inform the server it has 5 seconds to finish
	// the request it is currently handling
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := apiServer.Shutdown(ctx); err != nil {
		logger.Debug("server forced to shutdown with error", "error", err)
	}

	logger.Info("server exiting")

	// Notify the main goroutine that the shutdown is complete
	done <- true
}

func main() {
	ctx := context.Background()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := config.Load(logger)
	if err != nil {
		logger.Error("could not load config", "error", err)
		os.Exit(1)
	}

	srv := server.New(cfg)

	otelShutdown, err := otel.SetupOTelSDK(ctx, cfg.Otel)
	if err != nil {
		logger.Error("error setting up OpenTelemetry SDK", "error", err)
		os.Exit(1)
	}
	// Handle shutdown properly so nothing leaks
	defer func() {
		err = errors.Join(err, otelShutdown(context.Background()))
	}()

	// Create a done channel to signal when the shutdown is complete
	done := make(chan bool, 1)

	// Run graceful shutdown in a separate goroutine
	go gracefulShutdown(ctx, logger, srv, done)

	err = srv.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		panic(fmt.Sprintf("http server error: %s", err))
	}

	// Wait for the graceful shutdown to complete
	<-done
	logger.Info("graceful shutdown complete")
}
