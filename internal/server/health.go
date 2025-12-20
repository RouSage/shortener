package server

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
)

type HealthResponse struct {
	Status      string `json:"status" example:"ok"`
	Environment string `json:"environment" example:"production"`
}

// healthHandler godoc
//
//	@Summary		Simple Health Check
//	@Description	Returns basic health status of the application
//	@Tags			Health
//	@Produce		json
//	@Success		200	{object}	HealthResponse
//	@Failure		503	{object}	HealthResponse
//	@Router			/v1/health [get]
func (s *Server) healthHandler(c echo.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Quick database ping
	err := s.db.Ping(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("database health check failed")
		return c.JSON(http.StatusServiceUnavailable, &HealthResponse{
			Status:      "unavailable",
			Environment: s.cfg.App.Env,
		})
	}

	return c.JSON(http.StatusOK, &HealthResponse{
		Status:      "ok",
		Environment: s.cfg.App.Env,
	})
}

func (s *Server) healthMetricsHandler(c echo.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	stats := make(map[string]string)

	// Ping the database
	err := s.db.Ping(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("database health check failed")
		stats["status"] = "down"
		stats["error"] = fmt.Sprintf("db down: %s", err)
		return c.JSON(http.StatusServiceUnavailable, stats)
	}

	// Database is up, add more statistics
	stats["status"] = "up"
	stats["message"] = "It's healthy"

	// Get database stats (like open connections, in use, idle, etc.)
	dbStats := s.db.Stat()
	stats["open_connections"] = strconv.FormatInt(dbStats.NewConnsCount(), 10)
	stats["in_use"] = strconv.Itoa(int(dbStats.AcquiredConns()))
	stats["idle"] = strconv.Itoa(int(dbStats.IdleConns()))
	stats["wait_count"] = strconv.FormatInt(dbStats.EmptyAcquireCount(), 10)
	stats["wait_duration"] = dbStats.EmptyAcquireWaitTime().String()
	stats["max_idle_closed"] = strconv.FormatInt(dbStats.MaxIdleDestroyCount(), 10)
	stats["max_lifetime_closed"] = strconv.FormatInt(dbStats.MaxLifetimeDestroyCount(), 10)

	// Evaluate stats to provide a health message
	if dbStats.NewConnsCount() > 40 { // Assuming 50 is the max for this example
		stats["message"] = "The database is experiencing heavy load."
	}

	if dbStats.EmptyAcquireCount() > 1000 {
		stats["message"] = "The database has a high number of wait events, indicating potential bottlenecks."
	}

	if dbStats.MaxIdleDestroyCount() > dbStats.NewConnsCount()/2 {
		stats["message"] = "Many idle connections are being closed, consider revising the connection pool settings."
	}

	if dbStats.MaxLifetimeDestroyCount() > dbStats.NewConnsCount()/2 {
		stats["message"] = "Many connections are being closed due to max lifetime, consider increasing max lifetime or revising the connection usage pattern."
	}

	return c.JSON(http.StatusOK, stats)
}
