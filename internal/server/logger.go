package server

import (
	"context"
	"errors"
	"log/slog"
	"os"

	"github.com/rousage/shortener/internal/config"
	"go.opentelemetry.io/contrib/bridges/otelslog"
)

// A fan out approach to logs
// - Log with a default slog in the local environment for easier development/debugging
// - Use otelslog bridge to export logs as OTel signal to the Collector and further down the pipe (Grafana)
func newLogger(env config.Environment) *slog.Logger {
	otelHandler := otelslog.NewHandler(name)
	if env == config.EnvLocal {
		stdoutHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})
		return slog.New(&multiHandler{handlers: []slog.Handler{otelHandler, stdoutHandler}})
	}
	return slog.New(otelHandler)
}

type multiHandler struct {
	handlers []slog.Handler
}

func (h *multiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (h *multiHandler) Handle(ctx context.Context, r slog.Record) error {
	var errs []error
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, r.Level) {
			if err := handler.Handle(ctx, r.Clone()); err != nil {
				errs = append(errs, err)
			}
		}
	}
	return errors.Join(errs...)
}

func (h *multiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newHandlers := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		newHandlers[i] = handler.WithAttrs(attrs)
	}
	return &multiHandler{handlers: newHandlers}
}

func (h *multiHandler) WithGroup(name string) slog.Handler {
	newHandlers := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		newHandlers[i] = handler.WithGroup(name)
	}
	return &multiHandler{handlers: newHandlers}
}
