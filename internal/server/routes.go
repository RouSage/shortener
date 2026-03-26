package server

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	echootel "github.com/labstack/echo-opentelemetry"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	"github.com/rousage/shortener/internal/appvalidator"
	"github.com/rousage/shortener/internal/auth"
	"github.com/rousage/shortener/internal/otel"
	echoSwagger "github.com/swaggo/echo-swagger/v2"
	"golang.org/x/time/rate"

	_ "github.com/rousage/shortener/docs"
)

//	@title			Shortener API
//	@version		1.0
//	@description	This is URL shortener API

//	@license.name	MIT
//	@license.url	https://github.com/RouSage/shortener/blob/main/LICENSE

//	@host		localhost:3001
//	@BasePath	/
//	@accept		json
//	@produce	json
//	@schemes	http https

// @securityDefinitions.apikey	BearerAuth
// @in							header
// @name						Authorization
// @description				Type "Bearer" followed by a space and JWT token
func (s *Server) RegisterRoutes(logger *slog.Logger) http.Handler {
	e := echo.New()
	e.Logger = logger
	e.Validator = appvalidator.New()

	e.Use(echootel.NewMiddleware(otel.ServiceName.Value.AsString()))
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
		LogContentLength: true,
		LogResponseSize:  true,
		LogValuesFunc: func(c *echo.Context, v middleware.RequestLoggerValues) error {
			attrs := []slog.Attr{
				slog.Int64("latency", v.Latency.Milliseconds()),
				slog.String("protocol", v.Protocol),
				slog.String("remote_ip", v.RemoteIP),
				slog.String("host", v.Host),
				slog.String("method", v.Method),
				slog.String("uri", v.URI),
				slog.String("route", v.RoutePath),
				slog.String("request_id", v.RequestID),
				slog.String("referer", v.Referer),
				slog.String("user_agent", v.UserAgent),
				slog.Int("status", v.Status),
				slog.String("content_length", v.ContentLength),
				slog.Int64("response_size", v.ResponseSize),
			}
			if v.Error != nil {
				logger.LogAttrs(context.Background(), slog.LevelError, "REQUEST_ERROR",
					append(attrs, slog.String("error", v.Error.Error()))...,
				)
			} else {
				logger.LogAttrs(context.Background(), slog.LevelInfo, "REQUEST", attrs...)
			}

			return nil
		},
		Skipper: func(c *echo.Context) bool {
			path := c.Request().URL.Path
			return path == "/health" || path == "/health/metrics"
		},
	}))

	e.Use(middleware.RateLimiter(middleware.NewRateLimiterMemoryStoreWithConfig(middleware.RateLimiterMemoryStoreConfig{
		Rate:      float64(rate.Limit(s.cfg.Server.LimiterRPS)),
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

	authMw := auth.NewMiddleware(s.cfg.Auth)

	e.GET("/*", echoSwagger.EchoWrapHandlerV3(echoSwagger.PersistAuthorization(true), echoSwagger.SyntaxHighlight(true)))

	v1 := e.Group("/v1", authMw.Authenticate)
	v1.GET("/health", s.healthHandler)
	v1.GET("/health/metrics", s.healthMetricsHandler)

	v1.POST("/urls", s.createShortURLHandler)
	v1.GET("/urls/:code", s.getLongUrlHandler)
	v1.GET("/urls", s.getUserUrls, authMw.RequireAuthentication, authMw.RequirePermission(auth.GetOwnURLs))
	v1.DELETE("/urls/:code", s.deletShortUrlHandler, authMw.RequireAuthentication, authMw.RequirePermission(auth.DeleteOwnURLs))

	// Admin routes
	admin := v1.Group("/admin", authMw.RequireAuthentication)
	admin.GET("/urls", s.getURLs, authMw.RequirePermission(auth.GetURLs))
	admin.DELETE("/urls/:code", s.deleteURLHandler, authMw.RequirePermission(auth.DeleteURLs))
	admin.DELETE("/urls/user/:userId", s.deleteUserURLsHandler, authMw.RequirePermission(auth.DeleteURLs))

	adminUsers := admin.Group("/users")
	adminUsers.GET("/blocks", s.getUserBlocks, authMw.RequirePermission(auth.GetUserBlocks))
	adminUsers.POST("/block/:userId", s.blockUserHandler, authMw.RequirePermission(auth.UserBlock))
	adminUsers.POST("/unblock/:userId", s.unblockUserHandler, authMw.RequirePermission(auth.UserUnblock))

	return e
}
