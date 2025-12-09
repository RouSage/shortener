package server

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rousage/shortener/internal/appvalidator"
	"github.com/rousage/shortener/internal/auth"
	echoSwagger "github.com/swaggo/echo-swagger"
	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
	"golang.org/x/time/rate"

	_ "github.com/rousage/shortener/docs"
)

// @title Shortener API
// @version 1.0
// @description This is URL shortener API

// @license.name MIT
// @license.url https://github.com/RouSage/shortener/blob/main/LICENSE

// @host localhost:3001
// @BasePath /
func (s *Server) RegisterRoutes() http.Handler {
	e := echo.New()
	e.Validator = appvalidator.New()

	e.Use(otelecho.Middleware(serviceName.Value.AsString()))
	e.Use(middleware.RequestID())

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
			logEvent := s.logger.Info()
			if v.Error != nil {
				logEvent = s.logger.Error()
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

	e.Use(middleware.RateLimiter(middleware.NewRateLimiterMemoryStoreWithConfig(middleware.RateLimiterMemoryStoreConfig{
		Rate:      rate.Limit(s.cfg.Server.LimiterRPS),
		Burst:     s.cfg.Server.LimiterBurst,
		ExpiresIn: 3 * time.Minute,
	})))

	e.Use(middleware.Recover())

	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     s.cfg.Server.AllowOrigins,
		AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions, http.MethodPatch},
		AllowHeaders:     []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	authMw := auth.NewAuthMiddleware(s.cfg.Auth, s.logger)

	e.GET("/swagger/*", echoSwagger.EchoWrapHandler())
	e.GET("/", s.helloWorldHandler)
	e.GET("/health", s.healthHandler)

	urlsApi := e.Group("/urls", authMw.Authenticate)
	urlsApi.POST("", s.createShortURLHandler)
	urlsApi.GET("", s.getUserUrls, authMw.RequireAuthentication)
	urlsApi.GET("/:code", s.getLongUrlHandler)
	urlsApi.DELETE("/:code", s.deletShortUrlHandler, authMw.RequireAuthentication)

	return e
}

func (s *Server) helloWorldHandler(c echo.Context) error {
	resp := map[string]string{
		"message": "Hello World",
	}

	return c.JSON(http.StatusOK, resp)
}
