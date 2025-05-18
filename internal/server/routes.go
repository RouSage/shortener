package server

import (
	"net/http"
	"os"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog"
)

func (s *Server) RegisterRoutes() http.Handler {
	e := echo.New()

	e.Use(middleware.RequestID())

	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogLatency:       true,
		LogProtocol:      true,
		LogRemoteIP:      true,
		LogHost:          true,
		LogMethod:        true,
		LogURI:           true,
		LogURIPath:       false,
		LogRoutePath:     true,
		LogRequestID:     true,
		LogReferer:       true,
		LogUserAgent:     true,
		LogStatus:        true,
		LogError:         true,
		LogContentLength: true,
		LogResponseSize:  true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			logEvent := logger.Info()
			if v.Error != nil {
				logEvent = logger.Error()
			}

			logEvent.
				Int64("latency", v.Latency.Milliseconds()).
				Str("protocol", v.Protocol).
				Str("remote_ip", v.RemoteIP).
				Str("host", v.Host).
				Str("method", v.Method).
				Str("uri", v.URI).
				Str("route", v.RoutePath).
				Str("request_id", v.RequestID).
				Str("referer", v.Referer).
				Str("user_agent", v.UserAgent).
				Int("status", v.Status).
				Str("uri", v.URI).
				Int("status", v.Status).
				Str("content_length", v.ContentLength).
				Int64("response_size", v.ResponseSize)

			if v.Error != nil {
				logEvent.
					Err(v.Error).
					Msg("")
			} else {
				logEvent.Msg("request")
			}

			return nil
		},
		Skipper: func(c echo.Context) bool {
			return c.Request().URL.Path == "/health"
		},
	}))

	e.Use(middleware.Recover())

	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     []string{"https://*", "http://*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:     []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	e.GET("/", s.HelloWorldHandler)

	e.GET("/health", s.healthHandler)

	return e
}

func (s *Server) HelloWorldHandler(c echo.Context) error {
	resp := map[string]string{
		"message": "Hello World",
	}

	return c.JSON(http.StatusOK, resp)
}

func (s *Server) healthHandler(c echo.Context) error {
	return c.JSON(http.StatusOK, s.db.Health())
}
