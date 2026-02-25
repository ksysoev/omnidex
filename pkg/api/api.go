// Package api provides the implementation of the API server for the application.
package api

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/ksysoev/omnidex/pkg/core"
)

const (
	defaultTimeout  = 5 * time.Second
	shutdownTimeout = 10 * time.Second
)

// API is the main HTTP server that serves both the ingest API and the documentation portal.
type API struct {
	svc    Service
	views  ViewRenderer
	config Config
}

// Config holds the configuration for the API server.
type Config struct {
	Listen  string   `mapstructure:"listen"`
	APIKeys []string `mapstructure:"api_keys"` //nolint:gosec // This is a config struct, not a secret value
}

// Service defines the interface for core business logic operations.
type Service interface {
	IngestDocuments(ctx context.Context, req core.IngestRequest) (*core.IngestResponse, error)
	GetDocument(ctx context.Context, repo, path string) (core.Document, []byte, error)
	SearchDocs(ctx context.Context, query string, opts core.SearchOpts) (*core.SearchResults, error)
	ListRepos(ctx context.Context) ([]core.RepoInfo, error)
	ListDocuments(ctx context.Context, repo string) ([]core.DocumentMeta, error)
}

// ViewRenderer defines the interface for rendering HTML views.
type ViewRenderer interface {
	RenderHome(w io.Writer, repos []core.RepoInfo, partial bool) error
	RenderRepoIndex(w io.Writer, repo string, docs []core.DocumentMeta, partial bool) error
	RenderDoc(w io.Writer, doc core.Document, html []byte, navDocs []core.DocumentMeta, partial bool) error
	RenderSearch(w io.Writer, query string, results *core.SearchResults, partial bool) error
	RenderNotFound(w io.Writer) error
}

// New creates a new API instance with the provided configuration, service, and view renderer.
// It validates the configuration and returns an error if the listen address is not specified.
func New(cfg Config, svc Service, views ViewRenderer) (*API, error) {
	if cfg.Listen == "" {
		return nil, fmt.Errorf("listen address must be specified")
	}

	api := &API{
		config: cfg,
		svc:    svc,
		views:  views,
	}

	return api, nil
}

// Run starts the API server with the provided configuration.
// It listens on the address specified in the configuration and handles graceful shutdown.
// When the context is cancelled, in-flight requests are given a grace period to complete
// before the server is forcefully closed.
func (a *API) Run(ctx context.Context) error {
	s := &http.Server{
		Addr:              a.config.Listen,
		ReadHeaderTimeout: defaultTimeout,
		WriteTimeout:      defaultTimeout,
		Handler:           a.newMux(),
	}

	go func() {
		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		slog.WarnContext(ctx, "shutting down API server")

		if err := s.Shutdown(shutdownCtx); err != nil {
			slog.ErrorContext(ctx, "graceful shutdown failed, forcing close", "error", err)

			if closeErr := s.Close(); closeErr != nil {
				slog.ErrorContext(ctx, "forced close failed", "error", closeErr)
			}
		}
	}()

	if err := s.ListenAndServe(); err != http.ErrServerClosed {
		return err
	}

	return nil
}
