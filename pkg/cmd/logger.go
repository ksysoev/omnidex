package cmd

import (
	"context"
	"log/slog"
	"os"

	"github.com/ksysoev/omnidex/pkg/api/middleware"
)

// ContextHandler is a custom slog.Handler that enriches log records with application-specific attributes.
// It embeds a slog.Handler and adds attributes like application name and version, as well as request-specific context data.
type ContextHandler struct {
	slog.Handler
	ver string
	app string
}

// Handle processes a log record by enriching it with context and application-specific attributes.
// It adds attributes such as "req_id" from the context, "app", and "ver" before delegating to the embedded handler.
// Returns error if the embedded handler fails.

//nolint:gocritic // ignore this linting rule
func (h ContextHandler) Handle(ctx context.Context, r slog.Record) error {
	if reqID := middleware.GetReqID(ctx); reqID != "" {
		r.AddAttrs(slog.String("req_id", reqID))
	}

	r.AddAttrs(slog.String("app", h.app), slog.String("ver", h.ver))

	return h.Handler.Handle(ctx, r)
}

// initLogger initializes the default logger for the application using slog.
// It does not take any parameters.
// It returns an error if the logger initialization fails, although in this implementation, it always returns nil.
func initLogger(flags *cmdFlags) error {
	var logLevel slog.Level
	if err := logLevel.UnmarshalText([]byte(flags.LogLevel)); err != nil {
		return err
	}

	options := &slog.HandlerOptions{
		Level: logLevel,
	}

	var logHandler slog.Handler
	if flags.TextFormat {
		logHandler = slog.NewTextHandler(os.Stdout, options)
	} else {
		logHandler = slog.NewJSONHandler(os.Stdout, options)
	}

	ctxHandler := &ContextHandler{
		Handler: logHandler,
		ver:     flags.version,
		app:     flags.appName,
	}

	logger := slog.New(ctxHandler)

	slog.SetDefault(logger)

	return nil
}
